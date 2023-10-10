package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kofuk/premises/runner/backup"
	"github.com/kofuk/premises/runner/cleanup"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/gamesrv"
	"github.com/kofuk/premises/runner/keepsystemutd"
	"github.com/kofuk/premises/runner/metadata"
	"github.com/kofuk/premises/runner/privileged"
	"github.com/kofuk/premises/runner/serverprop"
	"github.com/kofuk/premises/runner/serversetup"
	"github.com/kofuk/premises/runner/statusapi"
	"github.com/kofuk/premises/runner/systemstat"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

//go:embed i18n/*.json
var i18nData embed.FS

func LoadI18nData(ctx *config.PMCMContext) error {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	ents, err := i18nData.ReadDir("i18n")
	if err != nil {
		return err
	}
	for _, ent := range ents {
		if _, err := bundle.LoadMessageFileFS(i18nData, "i18n/"+ent.Name()); err != nil {
			return err
		}
	}
	ctx.Localize = bundle
	return nil
}

func generateServerProps(ctx *config.PMCMContext, srv *gamesrv.ServerInstance) error {
	serverProps := serverprop.New()
	serverProps.SetMotd(ctx.Cfg.Motd)
	serverProps.SetDifficulty(ctx.Cfg.World.Difficulty)
	serverProps.SetLevelType(ctx.Cfg.World.LevelType)
	serverProps.SetSeed(ctx.Cfg.World.Seed)
	serverPropsFile, err := os.Create(ctx.LocateWorldData("server.properties"))
	if err != nil {
		return err
	}
	defer serverPropsFile.Close()
	if err := serverProps.Write(serverPropsFile); err != nil {
		return err
	}
	return nil
}

func downloadWorldIfNeeded(ctx *config.PMCMContext) error {
	if ctx.Cfg.World.ShouldGenerate {
		return nil
	}

	lastWorldHash, exists, err := backup.GetLastWorldHash(ctx)
	if err != nil {
		return err
	}

	if !ctx.Cfg.World.UseCache || !exists || ctx.Cfg.World.GenerationId != lastWorldHash {
		if err := backup.RemoveLastWorldHash(ctx); err != nil {
			log.WithError(err).Error("Failed to remove last world hash")
		}

		ctx.NotifyStatus(ctx.L("world.downloading"), false)
		if err := backup.DownloadWorldData(ctx); err != nil {
			return err
		}
		return nil
	}

	ctx.NotifyStatus(ctx.L("world.download.not_needed"), false)

	return nil
}

func downloadServerJarIfNeeded(ctx *config.PMCMContext) error {
	if _, err := os.Stat(ctx.LocateServer(ctx.Cfg.Server.Version)); err == nil {
		log.Info("No need to download server.jar")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(ctx.LocateDataFile("servers.d"), 0755); err != nil {
		return err
	}

	log.WithField("url", ctx.Cfg.Server.DownloadUrl).Info("Downloading Minecraft server...")
	resp, err := http.Get(ctx.Cfg.Server.DownloadUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return errors.New(fmt.Sprintf("Download failed with status: %d", resp.StatusCode))
	}

	outFile, err := os.Create(ctx.LocateServer(ctx.Cfg.Server.Version) + ".download")
	if err != nil {
		resp.Body.Close()
		return err
	}

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		outFile.Close()
		return err
	}
	outFile.Close()

	if err := os.Rename(ctx.LocateServer(ctx.Cfg.Server.Version)+".download", ctx.LocateServer(ctx.Cfg.Server.Version)); err != nil {
		return err
	}

	log.Info("Downloading Minecraft server...Done")

	return nil
}

func runServer() {
	log.Printf("Starting Premises Runner (revision: %s)\n", metadata.Revision)

	ctx := new(config.PMCMContext)

	if err := config.LoadConfig(ctx); err != nil {
		log.Fatal(err)
	}

	if err := LoadI18nData(ctx); err != nil {
		log.WithError(err).Error("Failed to load i18n data")
	}

	srv := new(gamesrv.ServerInstance)
	go statusapi.LaunchStatusServer(ctx, srv)

	ctx.NotifyStatus(ctx.L("mc.downloading"), srv.StartupFailed)
	if err := downloadServerJarIfNeeded(ctx); err != nil {
		log.WithError(err).Error("Couldn't download server.jar")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("mc.download.error"), srv.StartupFailed)
		goto out
	}

	if err := generateServerProps(ctx, srv); err != nil {
		log.WithError(err).Error("Couldn't generate server.properties")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("serverprops.generate.error"), srv.StartupFailed)
		goto out
	}

	if strings.Contains(ctx.Cfg.Server.Version, "/") {
		log.Error("ServerName can't contain /")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("mc.invalid_server_name"), srv.StartupFailed)
		goto out
	}

	if err := downloadWorldIfNeeded(ctx); err != nil {
		log.WithError(err).Error("Failed to download world data")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("world.download.error"), srv.StartupFailed)
		goto out
	}

	if err := gamesrv.LaunchServer(ctx, srv); err != nil {
		log.WithError(err).Error("Failed to launch Minecraft server")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("game.launch.error"), srv.StartupFailed)
		goto out
	}

	srv.AddToWhiteList(ctx.Cfg.Whitelist)
	srv.AddToOp(ctx.Cfg.Operators)

	srv.IsServerInitialized = true

	srv.Wait()

	srv.IsGameFinished = true

	ctx.NotifyStatus(ctx.L("world.processing"), srv.StartupFailed)
	if err := backup.PrepareUploadData(ctx, backup.UploadOptions{}); err != nil {
		log.WithError(err).Error("Failed to create world archive")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("world.archive.error"), srv.StartupFailed)
		goto out
	}

	ctx.NotifyStatus(ctx.L("world.uploading"), srv.StartupFailed)
	if err := backup.UploadWorldData(ctx, backup.UploadOptions{}); err != nil {
		log.WithError(err).Error("Failed to upload world data")
		srv.StartupFailed = true
		ctx.NotifyStatus(ctx.L("world.upload.error"), srv.StartupFailed)
		goto out
	}

out:
	if srv.RestartRequested {
		log.Info("Restart...")
		ctx.NotifyStatus(ctx.L("process.restarting"), srv.StartupFailed)

		os.Exit(100)
	} else if srv.Crashed && !srv.ShouldStop {
		ctx.NotifyStatus(ctx.L("game.crashed"), srv.StartupFailed)

		// User may reconfigure the server
		for {
			time.Sleep(time.Second)
			if srv.RestartRequested || srv.ShouldStop {
				goto out
			}
		}
	} else {
		ctx.NotifyStatus(ctx.L("game.stopped"), srv.StartupFailed)
	}
}

func main() {
	log.SetReportCaller(true)

	printVersion := flag.Bool("version", false, "Print version (in machine-readable way) and exit.")
	runRcon := flag.Bool("rcon", false, "Launch rcon client.")
	runPrivilegedHelper := flag.Bool("privileged-helper", false, "Run this process as internal helper process")
	runServerSetup := flag.Bool("server-setup", false, "Run this process as server setup process")
	runKeepSystemUpToDate := flag.Bool("keep-system-up-to-date", false, "Run this process as keep-system-up-to-date process")
	runCleanUp := flag.Bool("clean", false, "Run this process as clean up process")
	runSystemStat := flag.Bool("system-stat", false, "Run this process as clean up process")

	flag.Parse()

	if *printVersion {
		fmt.Print(metadata.Revision)
		return
	}
	if *runRcon {
		gamesrv.LaunchInteractiveRcon()
		return
	}
	if *runPrivilegedHelper {
		privileged.Run()
		return
	}
	if *runServerSetup {
		serverSetup := serversetup.ServerSetup{}
		serverSetup.Run()
		return
	}
	if *runKeepSystemUpToDate {
		keepsystemutd.KeepSystemUpToDate()
		return
	}
	if *runCleanUp {
		cleanup.CleanUp()
		return
	}
	if *runSystemStat {
		systemstat.Run()
		return
	}

	runServer()
}

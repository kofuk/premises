package mclauncher

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/backup"
	"github.com/kofuk/premises/runner/commands/mclauncher/config"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/mclauncher/serverprop"
	"github.com/kofuk/premises/runner/commands/mclauncher/statusapi"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/metadata"
)

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

	backupService := backup.New(ctx.Cfg.AWS.AccessKey, ctx.Cfg.AWS.SecretKey, ctx.Cfg.S3.Endpoint, ctx.Cfg.S3.Bucket)

	if ctx.Cfg.World.GenerationId == "@/latest" {
		genId, err := backupService.GetLatestKey(ctx.Cfg.World.Name)
		if err != nil {
			return err
		}
		ctx.Cfg.World.GenerationId = genId
	}

	lastWorldHash, exists, err := backup.GetLastWorldHash(ctx)
	if err != nil {
		return err
	}

	if !exists || ctx.Cfg.World.GenerationId != lastWorldHash {
		if err := backup.RemoveLastWorldHash(ctx); err != nil {
			slog.Error("Failed to remove last world hash", slog.Any("error", err))
		}

		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}

		if err := backupService.DownloadWorldData(ctx); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func downloadServerJarIfNeeded(ctx *config.PMCMContext) error {
	if _, err := os.Stat(ctx.LocateServer(ctx.Cfg.Server.Version)); err == nil {
		slog.Info("No need to download server.jar")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(ctx.LocateDataFile("servers.d"), 0755); err != nil {
		return err
	}

	slog.Info("Downloading Minecraft server...", slog.String("url", ctx.Cfg.Server.DownloadUrl))
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

	slog.Info("Downloading Minecraft server...Done")

	return nil
}

func Run() {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	ctx := new(config.PMCMContext)

	if err := config.LoadConfig(ctx); err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	srv := gamesrv.New()
	go statusapi.LaunchStatusServer(ctx, srv)

	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: entity.EventGameDownload,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
	if err := downloadServerJarIfNeeded(ctx); err != nil {
		slog.Error("Couldn't download server.jar", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := generateServerProps(ctx, srv); err != nil {
		slog.Error("Couldn't generate server.properties", slog.Any("error", err))
		srv.StartupFailed = true
		goto out
	}

	if strings.Contains(ctx.Cfg.Server.Version, "/") {
		slog.Error("ServerName can't contain /")
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := downloadWorldIfNeeded(ctx); err != nil {
		slog.Error("Failed to download world data", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}

	if err := gamesrv.LaunchServer(ctx, srv); err != nil {
		slog.Error("Failed to launch Minecraft server", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	srv.AddToWhiteList(ctx.Cfg.Whitelist)
	srv.AddToOp(ctx.Cfg.Operators)

	srv.IsServerInitialized = true

	srv.Wait()

	srv.IsGameFinished = true

	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
	if err := backup.PrepareUploadData(ctx, backup.UploadOptions{}); err != nil {
		slog.Error("Failed to create world archive", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: entity.EventWorldUpload,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
	if err := backup.New(ctx.Cfg.AWS.AccessKey, ctx.Cfg.AWS.SecretKey, ctx.Cfg.S3.Endpoint, ctx.Cfg.S3.Bucket).UploadWorldData(ctx, backup.UploadOptions{}); err != nil {
		slog.Error("Failed to upload world data", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}
	if err := backup.New(ctx.Cfg.AWS.AccessKey, ctx.Cfg.AWS.SecretKey, ctx.Cfg.S3.Endpoint, ctx.Cfg.S3.Bucket).RemoveOldBackups(ctx); err != nil {
		slog.Error("Unable to delete outdated backups", slog.Any("error", err))
	}

out:
	if srv.RestartRequested {
		slog.Info("Restart...")

		os.Exit(100)
	} else if srv.Crashed && !srv.ShouldStop {
		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventStatus,
			Status: &entity.StatusExtra{
				EventCode: entity.EventCrashed,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}

		// User may reconfigure the server
		for {
			time.Sleep(time.Second)
			if srv.RestartRequested || srv.ShouldStop {
				goto out
			}
		}
	}
}

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

	"github.com/kofuk/premises/common/entity/runner"
	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/backup"
	"github.com/kofuk/premises/runner/commands/mclauncher/fs"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/mclauncher/serverprop"
	"github.com/kofuk/premises/runner/commands/mclauncher/statusapi"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/metadata"
)

func generateServerProps(config *runner.Config, srv *gamesrv.ServerInstance) error {
	serverProps := serverprop.New()
	serverProps.SetMotd(config.Motd)
	serverProps.SetDifficulty(config.World.Difficulty)
	serverProps.SetLevelType(config.World.LevelType)
	serverProps.SetSeed(config.World.Seed)
	serverPropsFile, err := os.Create(fs.LocateWorldData("server.properties"))
	if err != nil {
		return err
	}
	defer serverPropsFile.Close()
	if err := serverProps.Write(serverPropsFile); err != nil {
		return err
	}
	return nil
}

func downloadWorldIfNeeded(config *runner.Config) error {
	if config.World.ShouldGenerate {
		return nil
	}

	backupService := backup.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket)

	if config.World.GenerationId == "@/latest" {
		genId, err := backupService.GetLatestKey(config.World.Name)
		if err != nil {
			return err
		}
		config.World.GenerationId = genId
	}

	lastWorldHash, exists, err := backup.GetLastWorldHash(config)
	if err != nil {
		return err
	}

	if !exists || config.World.GenerationId != lastWorldHash {
		if err := backup.RemoveLastWorldHash(config); err != nil {
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

		if err := backupService.DownloadWorldData(config); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func downloadServerJarIfNeeded(config *runner.Config) error {
	if _, err := os.Stat(fs.LocateServer(config.Server.Version)); err == nil {
		slog.Info("No need to download server.jar")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(fs.LocateDataFile("servers.d"), 0755); err != nil {
		return err
	}

	slog.Info("Downloading Minecraft server...", slog.String("url", config.Server.DownloadUrl))
	resp, err := http.Get(config.Server.DownloadUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return errors.New(fmt.Sprintf("Download failed with status: %d", resp.StatusCode))
	}

	outFile, err := os.Create(fs.LocateServer(config.Server.Version) + ".download")
	if err != nil {
		resp.Body.Close()
		return err
	}

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		outFile.Close()
		return err
	}
	outFile.Close()

	if err := os.Rename(fs.LocateServer(config.Server.Version)+".download", fs.LocateServer(config.Server.Version)); err != nil {
		return err
	}

	slog.Info("Downloading Minecraft server...Done")

	return nil
}

func Run() {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	config, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	srv := gamesrv.New()
	go statusapi.LaunchStatusServer(config, srv)

	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: entity.EventGameDownload,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
	if err := downloadServerJarIfNeeded(config); err != nil {
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

	if err := generateServerProps(config, srv); err != nil {
		slog.Error("Couldn't generate server.properties", slog.Any("error", err))
		srv.StartupFailed = true
		goto out
	}

	if strings.Contains(config.Server.Version, "/") {
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

	if err := downloadWorldIfNeeded(config); err != nil {
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

	if err := gamesrv.LaunchServer(config, srv); err != nil {
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

	srv.AddToWhiteList(config.Whitelist)
	srv.AddToOp(config.Operators)

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
	if err := backup.PrepareUploadData(backup.UploadOptions{}); err != nil {
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
	if err := backup.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket).UploadWorldData(config, backup.UploadOptions{}); err != nil {
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
	if err := backup.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket).RemoveOldBackups(config); err != nil {
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

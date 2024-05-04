package mclauncher

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/backup"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/metadata"
	"github.com/kofuk/premises/runner/rpc"
)

func downloadWorldIfNeeded(config *runner.Config) error {
	if config.World.ShouldGenerate {
		if err := fs.RemoveIfExists(fs.LocateWorldData("world")); err != nil {
			slog.Error("Unable to remove world directory", slog.Any("error", err))
		}

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

		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		})

		if err := backupService.DownloadWorldData(config); err != nil {
			return err
		}

		if err := backup.ExtractWorldArchiveIfNeeded(); err != nil {
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

	slog.Info("Downloading Minecraft server...", slog.String("url", config.Server.DownloadUrl))

	if err := gamesrv.DownloadServerJar(config.Server.DownloadUrl, fs.LocateServer(config.Server.Version)); err != nil {
		return err
	}

	slog.Info("Downloading Minecraft server...Done")

	return nil
}

func Run(args []string) int {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	config, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		return 1
	}

	srv := gamesrv.New()

	rpcHandler := NewRPCHandler(rpc.DefaultServer, srv)
	rpcHandler.Bind()

	if err := downloadWorldIfNeeded(config); err != nil {
		slog.Error("Failed to download world data", slog.Any("error", err))
		srv.StartupFailed = true
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		goto out
	}

	if config.Server.PreferDetected {
		slog.Info("Read server version from level.dat")
		if err := gamesrv.DetectAndUpdateVersion(config); err != nil {
			slog.Error("Error detecting Minecraft version", slog.Any("error", err))
		}
	}

	if strings.Contains(config.Server.Version, "/") {
		slog.Error("ServerName can't contain /")
		srv.StartupFailed = true
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		})
		goto out
	}

	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventGameDownload,
		},
	})

	if err := downloadServerJarIfNeeded(config); err != nil {
		slog.Error("Couldn't download server.jar", slog.Any("error", err))
		srv.StartupFailed = true
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		})
		goto out
	}

	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})

	if err := gamesrv.LaunchServer(config, srv); err != nil {
		slog.Error("Failed to launch Minecraft server", slog.Any("error", err))
		srv.StartupFailed = true
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLaunchErr,
			},
		})
		goto out
	}

	srv.AddToWhiteList(config.Whitelist)
	srv.AddToOp(config.Operators)

	srv.IsServerInitialized = true

	srv.Wait()

	srv.IsGameFinished = true

	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	})
	if err := backup.PrepareUploadData(); err != nil {
		slog.Error("Failed to create world archive", slog.Any("error", err))
		srv.StartupFailed = true
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		goto out
	}

	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldUpload,
		},
	})
	if err := backup.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket).UploadWorldData(config); err != nil {
		slog.Error("Failed to upload world data", slog.Any("error", err))
		srv.StartupFailed = true
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		})
		goto out
	}
	if err := backup.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket).RemoveOldBackups(config); err != nil {
		slog.Error("Unable to delete outdated backups", slog.Any("error", err))
	}

out:
	if srv.RestartRequested {
		slog.Info("Restart...")

		return 100
	} else if srv.Crashed && !srv.ShouldStop {
		exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventCrashed,
			},
		})

		// User may reconfigure the server
		for {
			time.Sleep(time.Second)
			if srv.RestartRequested || srv.ShouldStop {
				goto out
			}
		}
	}

	return 0
}

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

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/backup"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/mclauncher/statusapi"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/metadata"
)

func downloadWorldIfNeeded(config *runner.Config) error {
	if config.World.ShouldGenerate {
		for _, dir := range []string{"world", "world_nether", "world_the_end"} {
			if _, err := os.Stat(fs.LocateWorldData(dir)); err == nil {
				if err := os.RemoveAll(fs.LocateWorldData(dir)); err != nil {
					slog.Error("Failed to remove world folder", slog.Any("error", err))
				}
			} else if !os.IsNotExist(err) {
				slog.Error("Failed to stat world dir", slog.Any("error", err))
			}
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

		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldDownload,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}

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

func isExecutableFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("Unable to open file", slog.Any("error", err))
		return false
	}
	defer file.Close()

	buf := make([]byte, 4)
	if _, err := file.Read(buf); err != nil {
		slog.Error("Unable to read file", slog.Any("error", err))
		return false
	}

	// ELF or shell script
	return (buf[0] == 0x7F && buf[1] == 'E' && buf[2] == 'L' && buf[3] == 'F') || (buf[0] == '#' && buf[1] == '!')
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

	if isExecutableFile(fs.LocateServer(config.Server.Version) + ".download") {
		if err := os.Chmod(fs.LocateServer(config.Server.Version)+".download", 0755); err != nil {
			return err
		}
	}

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

	if err := downloadWorldIfNeeded(config); err != nil {
		slog.Error("Failed to download world data", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
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
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventGameDownload,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}

	if err := downloadServerJarIfNeeded(config); err != nil {
		slog.Error("Couldn't download server.jar", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventGameErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}

	if err := gamesrv.LaunchServer(config, srv); err != nil {
		slog.Error("Failed to launch Minecraft server", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
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

	if err := exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldPrepare,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
	if err := backup.PrepareUploadData(backup.UploadOptions{}); err != nil {
		slog.Error("Failed to create world archive", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventWorldErr,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
		goto out
	}

	if err := exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventWorldUpload,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
	if err := backup.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket).UploadWorldData(config, backup.UploadOptions{}); err != nil {
		slog.Error("Failed to upload world data", slog.Any("error", err))
		srv.StartupFailed = true
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
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
		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
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

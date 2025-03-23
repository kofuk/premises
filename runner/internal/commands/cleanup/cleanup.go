package cleanup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/exterior"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/system"
)

func removeFilesIgnoreError(paths ...string) {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			slog.Info("Failed to clean up file", slog.Any("error", err), slog.String("path", path))
		}
	}
}

func removeTempFiles() {
	dirent, err := os.ReadDir(env.DataPath("tmp"))
	if err != nil {
		slog.Error("Error reading temp dir", slog.Any("error", err))
		return
	}

	var paths []string
	for _, e := range dirent {
		paths = append(paths, filepath.Join(env.DataPath("tmp"), e.Name()))
	}

	removeFilesIgnoreError(paths...)
}

func removeSnapshots(ctx context.Context) {
	dirent, err := os.ReadDir(env.DataPath("gamedata"))
	if err != nil {
		slog.Error("Error reading data dir", slog.Any("error", err))
		return
	}

	args := []string{"subvolume", "delete", "--commit-after"}
	needsClean := false
	for _, ent := range dirent {
		if len(ent.Name()) > 3 && ent.Name()[:3] == "ss@" {
			needsClean = true
			args = append(args, env.DataPath("gamedata", ent.Name()))
		}
	}

	if needsClean {
		if err := system.DefaultExecutor.Run(ctx, "btrfs", args); err != nil {
			slog.Error("Failed to remove snapshots", slog.Any("error", err))
		}
	}
}

func unmountData() {
	if err := syscall.Unmount(env.DataPath("gamedata"), 0); err != nil {
		slog.Error("Error unmounting data dir", slog.Any("error", err))
	}
}

func notifyStatus(ctx context.Context, eventCode entity.EventCode) {
	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: eventCode,
		},
	})
}

func copyLogData() {
	if _, err := os.Stat("/premises-dev"); err != nil {
		return
	}

	logFile, err := os.Open("/exteriord.log")
	if err != nil {
		slog.Error("Error creating log file", slog.Any("error", err))
		return
	}
	defer logFile.Close()

	out, err := os.Create(fmt.Sprintf("/premises-dev/exteriord-%s.log", time.Now().Format("2006-01-02T15-04-05")))
	if err != nil {
		slog.Error("Error creating copy file", slog.Any("error", err))
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, logFile); err != nil {
		slog.Error("Error copying log file", slog.Any("error", err))
		return
	}

	if err := os.Remove("/exteriord.log"); err != nil {
		slog.Error("Error removing unneeded log file", slog.Any("error", err))
	}
}

func Run(ctx context.Context, args []string) int {
	notifyStatus(ctx, entity.EventClean)

	slog.Info("Removing snaphots...")
	removeSnapshots(ctx)

	slog.Info("Unmounting data dir...")
	unmountData()

	slog.Info("Removing temp files...")
	removeTempFiles()

	slog.Info("Removing config files...")
	removeFilesIgnoreError(
		env.DataPath("config.json"),
		"/userdata",
		"/userdata_decoded.sh",
	)

	slog.Info("Copying log file if it is dev runner")
	copyLogData()

	notifyStatus(ctx, entity.EventShutdown)

	// XXX
	time.Sleep(5 * time.Second)

	rpc.ToExteriord.Notify(ctx, "proc/done", nil)

	return 0
}

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

	"github.com/kofuk/premises/backend/common/entity"
	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/env"
	"github.com/kofuk/premises/backend/runner/exterior"
	"github.com/kofuk/premises/backend/runner/rpc"
	"github.com/kofuk/premises/backend/runner/system"
)

func removeFilesIgnoreError(ctx context.Context, paths ...string) {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			slog.InfoContext(ctx, "Failed to clean up file", slog.Any("error", err), slog.String("path", path))
		}
	}
}

func removeTempFiles(ctx context.Context) {
	dirent, err := os.ReadDir(env.DataPath("tmp"))
	if err != nil {
		slog.ErrorContext(ctx, "Error reading temp dir", slog.Any("error", err))
		return
	}

	var paths []string
	for _, e := range dirent {
		paths = append(paths, filepath.Join(env.DataPath("tmp"), e.Name()))
	}

	removeFilesIgnoreError(ctx, paths...)
}

func removeSnapshots(ctx context.Context) {
	dirent, err := os.ReadDir(env.DataPath("gamedata"))
	if err != nil {
		slog.ErrorContext(ctx, "Error reading data dir", slog.Any("error", err))
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
			slog.ErrorContext(ctx, "Failed to remove snapshots", slog.Any("error", err))
		}
	}
}

func unmountData(ctx context.Context) {
	if err := syscall.Unmount(env.DataPath("gamedata"), 0); err != nil {
		slog.ErrorContext(ctx, "Error unmounting data dir", slog.Any("error", err))
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

func copyLogData(ctx context.Context) {
	if _, err := os.Stat("/premises-dev"); err != nil {
		return
	}

	logFile, err := os.Open("/exteriord.log")
	if err != nil {
		slog.ErrorContext(ctx, "Error creating log file", slog.Any("error", err))
		return
	}
	defer logFile.Close()

	out, err := os.Create(fmt.Sprintf("/premises-dev/exteriord-%s.log", time.Now().Format("2006-01-02T15-04-05")))
	if err != nil {
		slog.ErrorContext(ctx, "Error creating copy file", slog.Any("error", err))
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, logFile); err != nil {
		slog.ErrorContext(ctx, "Error copying log file", slog.Any("error", err))
		return
	}

	if err := os.Remove("/exteriord.log"); err != nil {
		slog.ErrorContext(ctx, "Error removing unneeded log file", slog.Any("error", err))
	}
}

func Run(ctx context.Context, args []string) int {
	notifyStatus(ctx, entity.EventClean)

	slog.InfoContext(ctx, "Removing snaphots...")
	removeSnapshots(ctx)

	slog.InfoContext(ctx, "Unmounting data dir...")
	unmountData(ctx)

	slog.InfoContext(ctx, "Removing temp files...")
	removeTempFiles(ctx)

	slog.InfoContext(ctx, "Removing config files...")
	removeFilesIgnoreError(
		ctx,
		env.DataPath("config.json"),
		"/userdata",
		"/userdata_decoded.sh",
	)

	slog.InfoContext(ctx, "Copying log file if it is dev runner")
	copyLogData(ctx)

	notifyStatus(ctx, entity.EventShutdown)

	// XXX
	time.Sleep(5 * time.Second)

	rpc.ToExteriord.Notify(ctx, "proc/done", nil)

	return 0
}

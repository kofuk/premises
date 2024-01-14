package cleanup

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kofuk/premises/common/entity"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
)

func removeFilesIgnoreError(paths ...string) {
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			slog.Info("Failed to clean up file", slog.Any("error", err), slog.String("path", path))
		}
	}
}

func removeSnapshots() {
	dirent, err := os.ReadDir("/opt/premises/gamedata")
	if err != nil {
		slog.Error("Error reading data dir", slog.Any("error", err))
		return
	}

	args := []string{"subvolume", "delete", "--commit-after"}
	needsClean := false
	for _, ent := range dirent {
		if len(ent.Name()) > 3 && ent.Name()[:3] == "ss@" {
			needsClean = true
			args = append(args, filepath.Join("/opt/premises/gamedata", ent.Name()))
		}
	}

	if needsClean {
		if err := systemutil.Cmd("btrfs", args, nil); err != nil {
			slog.Error("Failed to remove snapshots", slog.Any("error", err))
		}
	}
}

func unmountData() {
	if err := syscall.Unmount("/opt/premises/gamedata", 0); err != nil {
		slog.Error("Error unmounting data dir", slog.Any("error", err))
	}
}

func notifyStatus(eventCode entity.EventCode) {
	if err := exterior.SendMessage("serverStatus", runnerEntity.Event{
		Type: runnerEntity.EventStatus,
		Status: &runnerEntity.StatusExtra{
			EventCode: eventCode,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}
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

func CleanUp() {
	notifyStatus(runnerEntity.EventClean)

	slog.Info("Removing snaphots...")
	removeSnapshots()

	slog.Info("Unmounting data dir...")
	unmountData()

	slog.Info("Removing config files...")
	removeFilesIgnoreError(
		"/opt/premises/config.json",
		"/userdata",
		"/userdata_decoded.sh",
	)

	slog.Info("Copying log file if it is dev runner")
	copyLogData()

	notifyStatus(runnerEntity.EventShutdown)
}

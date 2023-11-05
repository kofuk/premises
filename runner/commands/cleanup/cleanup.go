package cleanup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
	log "github.com/sirupsen/logrus"
)

func removeFilesIgnoreError(paths ...string) {
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			log.WithError(err).WithField("path", path).Info("Failed to clean up file")
		}
	}
}

func removeSnapshots() {
	dirent, err := os.ReadDir("/opt/premises/gamedata")
	if err != nil {
		log.WithError(err).Error("Error reading data dir")
		return
	}

	args := []string{"subvolume", "delete", "--commit-after"}
	needsClean := false
	for _, ent := range dirent {
		if ent.Name()[:3] == "ss@" {
			needsClean = true
			args = append(args, filepath.Join("/opt/premises/gamedata", ent.Name()))
		}
	}

	if needsClean {
		if err := systemutil.Cmd("btrfs", args, nil); err != nil {
			log.WithError(err).Info("Failed to remove snapshots")
		}
	}
}

func unmountData() {
	if err := syscall.Unmount("/opt/premises/gamedata", 0); err != nil {
		log.WithError(err).Error("Error unmounting data dir")
	}
}

func notifyStatus(eventCode entity.EventCode) {
	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: eventCode,
			LegacyMsg: "サーバを終了する準備をしています…",
		},
	}); err != nil {
		log.WithError(err).Error("Unable to write send message")
	}
}

func copyLogData() {
	if _, err := os.Stat("/premises-dev"); err != nil {
		return
	}

	logFile, err := os.Open("/exteriord.log")
	if err != nil {
		log.WithError(err).Error("Error creating log file")
		return
	}
	defer logFile.Close()

	out, err := os.Create(fmt.Sprintf("/premises-dev/exteriord-%s.log", time.Now().Format("2006-01-02T15-04-05")))
	if err != nil {
		log.WithError(err).Error("Error creating copy file")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, logFile); err != nil {
		log.WithError(err).Error("Error copying log file")
		return
	}

	if err := os.Remove("/exteriord.log"); err != nil {
		log.WithError(err).Error("Error removing unneeded log file")
	}
}

func CleanUp() {
	notifyStatus(entity.EventClean)

	log.Info("Removing config files...")
	removeFilesIgnoreError(
		"/opt/premises/server.key",
		"/opt/premises/server.crt",
		"/opt/premises/config.json",
		"/userdata",
		"/userdata_decoded.sh",
	)

	log.Info("Removing snaphots...")
	removeSnapshots()

	log.Info("Unmounting data dir...")
	unmountData()

	log.Info("Copying log file if it is dev runner")
	copyLogData()

	notifyStatus(entity.EventShutdown)
}

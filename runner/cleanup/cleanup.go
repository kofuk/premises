package cleanup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kofuk/premises/runner/config"
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
	for _, ent := range dirent {
		if ent.Name()[:3] == "ss@" {
			args = append(args, filepath.Join("/opt/premises/gamedata", ent.Name()))
		}
	}

	if err := systemutil.Cmd("btrfs", args, nil); err != nil {
		log.WithError(err).Info("Failed to remove snapshots")
	}
}

func unmountData() {
	if err := syscall.Unmount("/opt/premises/gamedata", 0); err != nil {
		log.WithError(err).Error("Error unmounting data dir")
	}
}

func notifyStatus(finished bool) {
	statusData := config.StatusData{
		Type:     config.StatusTypeLegacyEvent,
		Status:   "サーバを終了する準備をしています…",
		Shutdown: finished,
		HasError: false,
	}
	statusJson, _ := json.Marshal(statusData)

	if err := exterior.SendMessage(exterior.Message{
		Type:     "serverStatus",
		UserData: string(statusJson),
	}); err != nil {
		log.WithError(err).Error("Unable to write send message")
	}
}

func CleanUp() {
	notifyStatus(false)

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

	notifyStatus(true)
}

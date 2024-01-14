package serversetup

import (
	"log/slog"
	"os"
	"os/exec"
	"os/user"

	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
)

var requiredProgs = []string{
	"mkfs.btrfs",
	"java",
	"ufw",
	"unzip",
}

type ServerSetup struct{}

func isServerInitialized() bool {
	for _, prog := range requiredProgs {
		_, err := exec.LookPath(prog)
		if err != nil {
			slog.Info("Required executable not found", slog.String("name", prog))
			return false
		}
	}

	if _, err := os.Stat("/opt/premises/gamedata"); os.IsNotExist(err) {
		return false
	}

	return true
}

func isDevEnv() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func (self *ServerSetup) notifyStatus() {
	if err := exterior.SendMessage("serverStatus", entity.Event{
		Type: entity.EventStatus,
		Status: &entity.StatusExtra{
			EventCode: entity.EventSysInit,
		},
	}); err != nil {
		slog.Error("Unable to write server status", slog.Any("error", err))
	}
}

func (self *ServerSetup) initializeServer() {
	self.notifyStatus()

	slog.Info("Updating package indices")
	systemutil.AptGet("update", "-y")

	slog.Info("Installing packages")
	systemutil.AptGet("install", "-y", "btrfs-progs", "openjdk-17-jre-headless", "ufw", "unzip")

	if _, err := user.LookupId("1000"); err != nil {
		slog.Info("Adding user")
		systemutil.Cmd("useradd", []string{"-U", "-s", "/bin/bash", "-u", "1000", "premises"}, nil)
	}

	if !isDevEnv() {
		slog.Info("Enabling ufw")
		systemutil.Cmd("systemctl", []string{"enable", "--now", "ufw.service"}, nil)
		systemutil.Cmd("ufw", []string{"enable"}, nil)

		slog.Info("Adding ufw rules")
		systemutil.Cmd("ufw", []string{"allow", "25565/tcp"}, nil)
		systemutil.Cmd("ufw", []string{"allow", "8521/tcp"}, nil)
	}

	slog.Info("Creating data directories")
	os.MkdirAll("/opt/premises/servers.d/../gamedata", 0755)

	if _, err := os.Stat("/opt/premises/gamedata.img"); os.IsNotExist(err) {
		slog.Info("Creating image file to save game data")
		size := "8G"
		if isDevEnv() {
			size = "1G"
		}
		systemutil.Cmd("fallocate", []string{"-l", size, "/opt/premises/gamedata.img"}, nil)

		slog.Info("Creating filesystem for gamedata.img")
		systemutil.Cmd("mkfs.btrfs", []string{"/opt/premises/gamedata.img"}, nil)
	}
}

func (self ServerSetup) Run() {
	if !isServerInitialized() {
		slog.Info("Server seems not to be initialized. Will run full initialization")
		self.initializeServer()
	}

	slog.Info("Mounting gamedata.img")
	systemutil.Cmd("mount", []string{"/opt/premises/gamedata.img", "/opt/premises/gamedata"}, nil)

	slog.Info("Ensure data directory owned by execution user")
	systemutil.Cmd("chown", []string{"-R", "1000:1000", "/opt/premises"}, nil)
}

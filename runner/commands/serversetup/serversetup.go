package serversetup

import (
	"encoding/json"
	"os"
	"os/exec"
	"os/user"

	"github.com/kofuk/premises/runner/commands/mclauncher/config"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
	log "github.com/sirupsen/logrus"
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
			log.Println("Required executable not found:", prog)
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
	statusData := config.StatusData{
		Type:     config.StatusTypeLegacyEvent,
		Status:   "サーバを初期化しています…",
		Shutdown: false,
		HasError: false,
	}
	statusJson, _ := json.Marshal(statusData)

	if err := exterior.SendMessage(exterior.Message{
		Type:     "serverStatus",
		UserData: string(statusJson),
	}); err != nil {
		log.Error(err)
	}
}

func (self *ServerSetup) initializeServer() {
	self.notifyStatus()

	log.Println("Updating package indices")
	systemutil.AptGet("update", "-y")

	log.Println("Installing packages")
	systemutil.AptGet("install", "-y", "btrfs-progs", "openjdk-17-jre-headless", "ufw", "unzip")

	if _, err := user.LookupId("1000"); err != nil {
		log.Println("Adding user")
		systemutil.Cmd("useradd", []string{"-U", "-s", "/bin/bash", "-u", "1000", "premises"}, nil)
	}

	if !isDevEnv() {
		log.Println("Enabling ufw")
		systemutil.Cmd("systemctl", []string{"enable", "--now", "ufw.service"}, nil)
		systemutil.Cmd("ufw", []string{"enable"}, nil)

		log.Println("Adding ufw rules")
		systemutil.Cmd("ufw", []string{"allow", "25565/tcp"}, nil)
		systemutil.Cmd("ufw", []string{"allow", "8521/tcp"}, nil)
	}

	log.Println("Creating data directories")
	os.MkdirAll("/opt/premises/servers.d/../gamedata", 0755)

	if _, err := os.Stat("/opt/premises/gamedata.img"); os.IsNotExist(err) {
		log.Println("Creating image file to save game data")
		size := "8G"
		if isDevEnv() {
			size = "1G"
		}
		systemutil.Cmd("fallocate", []string{"-l", size, "/opt/premises/gamedata.img"}, nil)

		log.Println("Creating filesystem for gamedata.img")
		systemutil.Cmd("mkfs.btrfs", []string{"/opt/premises/gamedata.img"}, nil)
	}
}

func (self ServerSetup) Run() {
	if !isServerInitialized() {
		log.Println("Server seems not to be initialized. Will run full initialization")
		self.initializeServer()
	}

	log.Println("Mounting gamedata.img")
	systemutil.Cmd("mount", []string{"/opt/premises/gamedata.img", "/opt/premises/gamedata"}, nil)

	log.Println("Ensure data directory owned by execution user")
	systemutil.Cmd("chown", []string{"-R", "1000:1000", "/opt/premises"}, nil)
}

package serversetup

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/user"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/systemutil"
)

const (
	// The latest JRE should be installed, as it is used as a fallback version for Java installations.
	// It is important to keep Java up-to-date as it is backwards compatible (not forward compatible).
	latestAvailableJre = "openjdk-21-jre-headless"
)

var requiredProgs = []string{
	"mkfs.btrfs",
	"java",
	"ufw",
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

	if _, err := os.Stat(fs.DataPath("gamedata.img")); os.IsNotExist(err) {
		return false
	}

	return true
}

func isDevEnv() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func getIPAddr() (v4Addrs []string, v6Addrs []string, err error) {
	if isDevEnv() {
		return []string{"127.0.0.2"}, nil, nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.IsLoopback() {
				slog.Info("Address is loopback", slog.String("addr", ipnet.IP.String()))
				continue
			}

			if v4Addr := ipnet.IP.To4(); v4Addr != nil {
				v4Addrs = append(v4Addrs, v4Addr.String())
			}
			v6Addrs = append(v6Addrs, ipnet.IP.To16().String())
		}
	}
	return
}

func (self *ServerSetup) sendServerHello() {
	systemVersion := systemutil.GetSystemVersion()

	eventData := runner.Event{
		Type: runner.EventHello,
		Hello: &runner.HelloExtra{
			Version: systemVersion.PremisesVersion,
			Host:    systemVersion.HostOS,
		},
	}

	var err error
	eventData.Hello.Addr.IPv4, eventData.Hello.Addr.IPv6, err = getIPAddr()
	if err != nil {
		slog.Error("Failed to get IP addresses for network interface", slog.Any("error", err))
	}

	exterior.DispatchMessage("serverStatus", eventData)
}

func (self *ServerSetup) notifyStatus() {
	exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventSysInit,
		},
	})
}

func (self *ServerSetup) initializeServer() {
	slog.Info("Installing packages")
	systemutil.AptGet("install", "-y", "btrfs-progs", latestAvailableJre, "ufw")

	if _, err := user.LookupId("1000"); err != nil {
		slog.Info("Adding user")
		systemutil.Cmd("useradd", []string{"-U", "-s", "/bin/bash", "-u", "1000", "premises"})
	}

	if !isDevEnv() {
		slog.Info("Enabling ufw")
		systemutil.Cmd("systemctl", []string{"enable", "--now", "ufw.service"})
		systemutil.Cmd("ufw", []string{"enable"})
	}

	if _, err := os.Stat(fs.DataPath("gamedata.img")); os.IsNotExist(err) {
		slog.Info("Creating image file to save game data")
		size := 8 * 1024 * 1024 * 1024 // 8 GiB
		if isDevEnv() {
			size = 1 * 1024 * 1024 * 1024 // 1 GiB
		}
		if err := fs.Fallocate(fs.DataPath("gamedata.img"), int64(size)); err != nil {
			slog.Error("Unable to create gamedata.img", slog.Any("error", err))
			return
		}

		slog.Info("Creating filesystem for gamedata.img")
		systemutil.Cmd("mkfs.btrfs", []string{fs.DataPath("gamedata.img")})
	}
}

func (self *ServerSetup) updateFirewallRules() {
	systemutil.Cmd("ufw", []string{"allow", "25565/tcp"})
	// Old runner requires 8521 to be exposed. Now, it's not needed so we delete it here.
	systemutil.Cmd("ufw", []string{"delete", "allow", "8521/tcp"})
}

func (self *ServerSetup) installRequiredJavaVersion() {
	config, err := config.Load()
	if err != nil {
		slog.Error("Unable to load config", slog.Any("error", err))
		return
	}

	installArgs := []string{"install", "-y", latestAvailableJre}
	if config.Server.JavaVersion != 0 {
		installArgs = append(installArgs, fmt.Sprintf("openjdk-%d-jre-headless", config.Server.JavaVersion))
	}

	systemutil.AptGet(installArgs...)
}

func (self ServerSetup) Run() {
	self.sendServerHello()
	self.notifyStatus()

	slog.Info("Creating required directories (if not exists)")
	os.MkdirAll(fs.DataPath("servers.d/../gamedata/../tmp"), 0755)

	slog.Info("Updating package indices")
	systemutil.AptGet("update", "-y")

	if !isServerInitialized() {
		slog.Info("Server seems not to be initialized. Will run full initialization")
		self.initializeServer()
	}

	slog.Info("Updating ufw rules")
	self.updateFirewallRules()

	slog.Info("Installing required Java version")
	self.installRequiredJavaVersion()

	slog.Info("Mounting gamedata.img")
	systemutil.Cmd("mount", []string{fs.DataPath("gamedata.img"), fs.DataPath("gamedata")})

	slog.Info("Ensure data directory owned by execution user")
	if err := fs.ChownRecursive(fs.DataPath(), 1000, 1000); err != nil {
		slog.Error("Error changing ownership", slog.Any("error", err))
	}
}

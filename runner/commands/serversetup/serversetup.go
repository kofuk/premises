package serversetup

import (
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/user"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
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

	if _, err := os.Stat("/opt/premises/gamedata"); os.IsNotExist(err) {
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
		return []string{"127.0.0.1"}, []string{"::1"}, nil
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

	if err := exterior.DispatchMessage("serverStatus", eventData); err != nil {
		slog.Error("Unable to write server hello", slog.Any("error", err))
	}
}

func (self *ServerSetup) notifyStatus() {
	if err := exterior.SendMessage("serverStatus", runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
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
	systemutil.AptGet("install", "-y", "btrfs-progs", "openjdk-17-jre-headless", "ufw")

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
		size := 8 * 1024 * 1024 * 1024 // 8 GiB
		if isDevEnv() {
			size = 1 * 1024 * 1024 * 1024 // 1 GiB
		}
		if err := systemutil.Fallocate("/opt/premises/gamedata.img", int64(size)); err != nil {
			slog.Error("Unable to create gamedata.img", slog.Any("error", err))
			return
		}

		slog.Info("Creating filesystem for gamedata.img")
		systemutil.Cmd("mkfs.btrfs", []string{"/opt/premises/gamedata.img"}, nil)
	}
}

func (self ServerSetup) Run() {
	self.sendServerHello()

	if !isServerInitialized() {
		slog.Info("Server seems not to be initialized. Will run full initialization")
		self.initializeServer()
	}

	slog.Info("Mounting gamedata.img")
	systemutil.Cmd("mount", []string{"/opt/premises/gamedata.img", "/opt/premises/gamedata"}, nil)

	slog.Info("Ensure data directory owned by execution user")
	if err := systemutil.ChownRecursive("/opt/premises", 1000, 1000); err != nil {
		slog.Error("Error changing ownership", slog.Any("error", err))
	}
}

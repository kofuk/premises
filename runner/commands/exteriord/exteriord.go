package exteriord

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kofuk/premises/runner/commands/exteriord/exterior"
	"github.com/kofuk/premises/runner/commands/exteriord/interior"
	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
	"github.com/kofuk/premises/runner/commands/exteriord/outbound"
	"github.com/kofuk/premises/runner/commands/exteriord/proc"
)

type Config struct {
	AuthKey string `json:"authKey"`
}

func getServerAuthKey() (string, error) {
	data, err := os.ReadFile("/opt/premises/config.json")
	if err != nil {
		return "", err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	return config.AuthKey, nil
}

func Run() {
	signal.Ignore(syscall.SIGHUP)

	msgRouter := msgrouter.NewMsgRouter()

	authKey, err := getServerAuthKey()
	if err != nil {
		slog.Error("Unable to get server auth key", slog.Any("error", err))
		os.Exit(1)
	}

	ob := outbound.NewServer("0.0.0.0:8521", authKey, msgRouter)
	go func() {
		if err := ob.Start(); err != nil {
			slog.Error("Unable to start outbound server", slog.Any("error", err))
		}
	}()

	interior := interior.NewServer("127.0.0.1:2000", msgRouter)
	go func() {
		if err := interior.Start(); err != nil {
			slog.Error("Unable to start interior server", slog.Any("error", err))
		}
	}()

	e := exterior.New()

	setupTask := e.RegisterTask("Initialize Server",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--setup"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		))
	e.RegisterTask("Syatem Statistics",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--sysstat"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		))
	monitoring := e.RegisterTask("Game Monitoring Service",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--launcher"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	systemUpdate := e.RegisterTask("Keep System Up-to-date",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--update-packages"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Snapshot Service",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--snapshot-helper"),
			proc.Restart(proc.RestartAlways),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Clean Up",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--clean"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), monitoring, systemUpdate)

	e.Run()
}

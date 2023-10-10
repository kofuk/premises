package main

import (
	"encoding/json"
	"log"
	"os"
	"syscall"

	"github.com/kofuk/premises/exteriord/exterior"
	"github.com/kofuk/premises/exteriord/interior"
	"github.com/kofuk/premises/exteriord/msgrouter"
	"github.com/kofuk/premises/exteriord/outbound"
	"github.com/kofuk/premises/exteriord/proc"
)

func IAmRoot() bool {
	return syscall.Getuid() == 0
}

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
	msgRouter := msgrouter.NewMsgRouter()

	authKey, err := getServerAuthKey()
	if err != nil {
		log.Fatal(err)
	}

	ob := outbound.NewServer("0.0.0.0:8521", authKey, msgRouter)
	go func() {
		if err := ob.Start(); err != nil {
			log.Println("Unable to start outbound server:", err)
		}
	}()

	interior := interior.NewServer("127.0.0.1:2000", msgRouter)
	go func() {
		if err := interior.Start(); err != nil {
			log.Println("Unable to start interior server:", err)
		}
	}()

	e := exterior.New()

	setupTask := e.RegisterTask("Initialize Server",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--server-setup"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		))
	e.RegisterTask("Syatem Statistics",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--system-stat"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		))
	monitoring := e.RegisterTask("Game Monitoring Service",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	systemUpdate := e.RegisterTask("Keep System Up-to-date",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--keep-system-up-to-date"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Snapshot Service",
		proc.NewProc("/opt/premises/bin/premises-runner",
			proc.Args("--privileged-helper"),
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

func main() {
	if !IAmRoot() {
		log.Println("exteriord must be executed as root")
		os.Exit(1)
	}

	Run()
}

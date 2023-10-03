package main

import (
	"log"
	"os"
	"syscall"

	"github.com/kofuk/premises/exteriord/exterior"
	"github.com/kofuk/premises/exteriord/proc"
)

func IAmRoot() bool {
	return syscall.Getuid() == 0
}

func main() {
	if !IAmRoot() {
		log.Println("exteriord must be executed as root")
		os.Exit(1)
	}

	e := exterior.New()

	setupTask := e.RegisterTask("Initialize Server",
		proc.NewProc("/opt/premises/bin/premises-mcmanager",
			proc.Args("--server-setup"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		))
	e.RegisterTask("Game Monitoring Service",
		proc.NewProc("/opt/premises/bin/premises-mcmanager",
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	e.RegisterTask("Keep System Up-to-date",
		proc.NewProc("/opt/premises/bin/premises-mcmanager",
			proc.Args("--keep-system-up-to-date"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Snapshot Service",
		proc.NewProc("/opt/premises/bin/premises-mcmanager",
			proc.Args("--privileged-helper"),
			proc.Restart(proc.RestartAlways),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserPrivileged),
		), setupTask)

	e.Run()
}

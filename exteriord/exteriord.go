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

	exterior := exterior.New()
	exterior.RegisterTask(*proc.NewProc("/opt/premises/bin/premises-mcmanager", proc.Restart(proc.RestartAlways), proc.RestartRandomDelay()))
	exterior.Run()
}

package main

import (
	"flag"
	"fmt"

	"github.com/kofuk/premises/runner/commands/cleanup"
	"github.com/kofuk/premises/runner/commands/keepsystemutd"
	"github.com/kofuk/premises/runner/commands/mclauncher"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/privileged"
	"github.com/kofuk/premises/runner/commands/serversetup"
	"github.com/kofuk/premises/runner/commands/systemstat"
	"github.com/kofuk/premises/runner/metadata"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetReportCaller(true)

	printVersion := flag.Bool("version", false, "Print version (in machine-readable way) and exit.")
	runRcon := flag.Bool("rcon", false, "Launch rcon client.")
	runPrivilegedHelper := flag.Bool("privileged-helper", false, "Run this process as internal helper process")
	runServerSetup := flag.Bool("server-setup", false, "Run this process as server setup process")
	runKeepSystemUpToDate := flag.Bool("keep-system-up-to-date", false, "Run this process as keep-system-up-to-date process")
	runCleanUp := flag.Bool("clean", false, "Run this process as clean up process")
	runSystemStat := flag.Bool("system-stat", false, "Run this process as clean up process")

	flag.Parse()

	if *printVersion {
		fmt.Print(metadata.Revision)
		return
	}
	if *runRcon {
		gamesrv.LaunchInteractiveRcon()
		return
	}
	if *runPrivilegedHelper {
		privileged.Run()
		return
	}
	if *runServerSetup {
		serverSetup := serversetup.ServerSetup{}
		serverSetup.Run()
		return
	}
	if *runKeepSystemUpToDate {
		keepsystemutd.KeepSystemUpToDate()
		return
	}
	if *runCleanUp {
		cleanup.CleanUp()
		return
	}
	if *runSystemStat {
		systemstat.Run()
		return
	}

	mclauncher.Run()
}

package main

import (
	"fmt"
	"os"
	"sort"
	"syscall"

	"github.com/kofuk/premises/runner/commands/cleanup"
	"github.com/kofuk/premises/runner/commands/exteriord"
	"github.com/kofuk/premises/runner/commands/keepsystemutd"
	"github.com/kofuk/premises/runner/commands/mclauncher"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/privileged"
	"github.com/kofuk/premises/runner/commands/serversetup"
	"github.com/kofuk/premises/runner/commands/systemstat"
	"github.com/kofuk/premises/runner/metadata"
	log "github.com/sirupsen/logrus"
)

type Command struct {
	Description  string
	Run          func()
	RequiresRoot bool
}

type App struct {
	Commands map[string]Command
}

func (self App) printUsage() {
	var keys []string
	for k := range self.Commands {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("premises-runner\nCommands:")
	for _, key := range keys {
		fmt.Printf("  %s\t%s\n", key, self.Commands[key].Description)
	}
}

func (self App) Run(args []string) {
	if len(args) < 2 {
		self.printUsage()
		os.Exit(1)
	}

	cmdName := args[1]
	if cmdName[0:2] == "--" {
		cmdName = cmdName[2:]
	}

	cmd, ok := self.Commands[cmdName]
	if !ok {
		fmt.Printf("Command '%s' not found.", cmdName)
		self.printUsage()
		os.Exit(1)
	}

	if cmd.RequiresRoot {
		if syscall.Getuid() != 0 {
			fmt.Println("This command requires root")
			os.Exit(1)
		}
	}

	cmd.Run()
}

func main() {
	log.SetReportCaller(true)

	app := App{
		Commands: map[string]Command{
			"clean": {
				Description:  "Cleanup before shutdown",
				Run:          cleanup.CleanUp,
				RequiresRoot: true,
			},
			"exteriord": {
				Description:  "Exteriord",
				Run:          exteriord.Run,
				RequiresRoot: true,
			},
			"launcher": {
				Description:  "Launch game server",
				Run:          mclauncher.Run,
				RequiresRoot: false,
			},
			"rcon": {
				Description:  "Interactive Rcon",
				Run:          gamesrv.LaunchInteractiveRcon,
				RequiresRoot: false,
			},
			"setup": {
				Description: "Setup server",
				Run: func() {
					serverSetup := serversetup.ServerSetup{}
					serverSetup.Run()
				},
				RequiresRoot: true,
			},
			"snapshot-helper": {
				Description:  "Privileged snapshot helper",
				Run:          privileged.Run,
				RequiresRoot: true,
			},
			"sysstat": {
				Description:  "Monitor system load",
				Run:          systemstat.Run,
				RequiresRoot: false,
			},
			"update-packages": {
				Description:  "Update system packages",
				Run:          keepsystemutd.KeepSystemUpToDate,
				RequiresRoot: true,
			},
			"version": {
				Description: "Print version (in machine-readable way) and exit",
				Run: func() {
					fmt.Println(metadata.Revision)
				},
				RequiresRoot: false,
			},
		},
	}

	app.Run(os.Args)
}

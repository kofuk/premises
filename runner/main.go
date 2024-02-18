package main

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"syscall"

	"github.com/kofuk/premises/runner/commands/cleanup"
	"github.com/kofuk/premises/runner/commands/exteriord"
	"github.com/kofuk/premises/runner/commands/keepsystemutd"
	"github.com/kofuk/premises/runner/commands/levelinspect"
	"github.com/kofuk/premises/runner/commands/mclauncher"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/privileged"
	"github.com/kofuk/premises/runner/commands/serversetup"
	"github.com/kofuk/premises/runner/commands/systemstat"
	"github.com/kofuk/premises/runner/metadata"
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
		slog.Error("Subcommand not found.", slog.String("cmd", cmdName))
		self.printUsage()
		os.Exit(1)
	}

	slog.SetDefault(slog.Default().With(slog.String("runner_command", cmdName)))

	if cmd.RequiresRoot {
		if syscall.Getuid() != 0 {
			slog.Error("This command requires root")
			os.Exit(1)
		}
	}

	cmd.Run()
}

func main() {
	logLevel := slog.LevelInfo
	if os.Getenv("PREMISES_VERBOSE") != "" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	})))

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
			"level-inspect": {
				Description:  "Parse level.dat",
				Run:          levelinspect.Run,
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

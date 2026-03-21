package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/commands/cleanup"
	"github.com/kofuk/premises/backend/runner/commands/cli"
	"github.com/kofuk/premises/backend/runner/commands/connector"
	"github.com/kofuk/premises/backend/runner/commands/exteriord"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher"
	"github.com/kofuk/premises/backend/runner/commands/meter"
	"github.com/kofuk/premises/backend/runner/commands/serversetup"
	"github.com/kofuk/premises/backend/runner/commands/snapshot"
	"github.com/kofuk/premises/backend/runner/commands/sysupdate"
	"github.com/kofuk/premises/backend/runner/config"
	"github.com/kofuk/premises/backend/runner/env"
	"github.com/kofuk/premises/backend/runner/metadata"
	"github.com/kofuk/premises/backend/runner/rpc"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"golang.org/x/sync/errgroup"
)

type Command struct {
	Description  string
	Run          func(ctx context.Context, config *runner.Config, args []string) int
	RequiresRoot bool
}

type App struct {
	Commands map[string]Command
}

func (app App) printUsage() {
	var keys []string
	for k := range app.Commands {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("premises-runner\nCommands:")
	for _, key := range keys {
		fmt.Printf("  %s\t%s\n", key, app.Commands[key].Description)
	}
}

func (app App) Run(ctx context.Context, args []string) int {
	if len(args) < 2 {
		app.printUsage()
		os.Exit(1)
	}

	cmdName := strings.TrimPrefix(args[1], "--")

	cmd, ok := app.Commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmdName)
		app.printUsage()
		os.Exit(1)
	}

	slog.SetDefault(slog.Default().With(slog.String("runner_command", cmdName)))
	os.Setenv("PREMISES_RUNNER_COMMAND", cmdName)

	config, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	instr := initInstrumentation(ctx, cmdName, config.Observability.OtlpEndpoint, config.Observability.MetricExportIntervalMs)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		instr.shutdown(ctx)
	}()

	slog.SetDefault(slog.New(otelslog.NewHandler("premises-runner")))

	rpc.InitializeDefaultServer(env.DataPath("rpc@" + cmdName))

	ctx, cancel := context.WithCancel(ctx)

	var eg errgroup.Group
	eg.Go(func() error {
		return rpc.DefaultServer.Start(ctx)
	})

	if cmd.RequiresRoot {
		if syscall.Getuid() != 0 {
			slog.ErrorContext(ctx, "This command requires root")
			os.Exit(1)
		}
	}

	status := cmd.Run(ctx, config, args[2:])

	// Stop background jobs and wait for them to finish
	cancel()
	eg.Wait()

	return status
}

func main() {
	app := App{
		Commands: map[string]Command{
			"clean": {
				Description:  "Cleanup before shutdown",
				Run:          cleanup.Run,
				RequiresRoot: true,
			},
			"cli": {
				Description: "CLI tools",
				Run:         cli.Run,
			},
			"connector": {
				Description:  "Connector",
				Run:          connector.Run,
				RequiresRoot: false,
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
			"meter": {
				Description:  "Start resource usage meter",
				Run:          meter.Run,
				RequiresRoot: false,
			},
			"setup": {
				Description: "Setup server",
				Run: func(ctx context.Context, config *runner.Config, args []string) int {
					serverSetup := serversetup.ServerSetup{}
					serverSetup.Run(ctx, config)
					return 0
				},
				RequiresRoot: true,
			},
			"snapshot-helper": {
				Description:  "Snapshot helper",
				Run:          snapshot.Run,
				RequiresRoot: true,
			},
			"update-packages": {
				Description:  "Update system packages",
				Run:          sysupdate.Run,
				RequiresRoot: true,
			},
			"version": {
				Description: "Print version (in machine-readable way) and exit",
				Run: func(ctx context.Context, config *runner.Config, args []string) int {
					fmt.Println(metadata.Revision)
					return 0
				},
				RequiresRoot: false,
			},
		},
	}

	os.Exit(app.Run(context.Background(), os.Args))
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"syscall"

	potel "github.com/kofuk/premises/internal/otel"
	"github.com/kofuk/premises/runner/internal/commands/cleanup"
	"github.com/kofuk/premises/runner/internal/commands/cli"
	"github.com/kofuk/premises/runner/internal/commands/connector"
	"github.com/kofuk/premises/runner/internal/commands/exteriord"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher"
	"github.com/kofuk/premises/runner/internal/commands/serversetup"
	"github.com/kofuk/premises/runner/internal/commands/snapshot"
	"github.com/kofuk/premises/runner/internal/commands/systemstat"
	"github.com/kofuk/premises/runner/internal/commands/sysupdate"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/metadata"
	"github.com/kofuk/premises/runner/internal/rpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

type Command struct {
	Description  string
	Run          func(ctx context.Context, args []string) int
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

func createContext() context.Context {
	traceContext := os.Getenv("TRACEPARENT")
	os.Unsetenv("TRACEPARENT")
	return potel.ContextFromTraceContext(context.Background(), traceContext)
}

func (app App) Run(ctx context.Context, args []string) int {
	if len(args) < 2 {
		app.printUsage()
		os.Exit(1)
	}

	cmdName := strings.TrimPrefix(args[1], "--")

	cmd, ok := app.Commands[cmdName]
	if !ok {
		slog.Error("Subcommand not found.", slog.String("cmd", cmdName))
		app.printUsage()
		os.Exit(1)
	}

	var tracerProvider *sdktrace.TracerProvider
	if cmdName != "sysstat" {
		var err error
		tracerProvider, err = potel.InitializeTracer(ctx)
		if err != nil {
			slog.Error("Failed to initialize tracer", slog.Any("error", err))
		}
	}
	if tracerProvider != nil {
		defer tracerProvider.Shutdown(ctx)

		tracer := tracerProvider.Tracer("github.com/kofuk/premises/runner/cmd/premises-runner")

		var span trace.Span
		ctx, span = tracer.Start(createContext(), "Runner main")
		defer span.End()
	}

	slog.SetDefault(slog.Default().With(slog.String("runner_command", cmdName)))
	os.Setenv("PREMISES_RUNNER_COMMAND", cmdName)

	rpc.InitializeDefaultServer(env.DataPath("rpc@" + cmdName))

	ctx, cancel := context.WithCancel(ctx)

	var eg errgroup.Group
	eg.Go(func() error {
		return rpc.DefaultServer.Start(ctx)
	})

	if cmd.RequiresRoot {
		if syscall.Getuid() != 0 {
			slog.Error("This command requires root")
			os.Exit(1)
		}
	}

	status := cmd.Run(ctx, args[2:])

	// Stop background jobs and wait for them to finish
	cancel()
	eg.Wait()

	return status
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
				Run:          cleanup.Run,
				RequiresRoot: true,
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
			"cli": {
				Description: "CLI tools",
				Run:         cli.Run,
			},
			"setup": {
				Description: "Setup server",
				Run: func(ctx context.Context, args []string) int {
					serverSetup := serversetup.ServerSetup{}
					serverSetup.Run(ctx)
					return 0
				},
				RequiresRoot: true,
			},
			"snapshot-helper": {
				Description:  "Snapshot helper",
				Run:          snapshot.Run,
				RequiresRoot: true,
			},
			"sysstat": {
				Description:  "Monitor system load",
				Run:          systemstat.Run,
				RequiresRoot: false,
			},
			"update-packages": {
				Description:  "Update system packages",
				Run:          sysupdate.Run,
				RequiresRoot: true,
			},
			"version": {
				Description: "Print version (in machine-readable way) and exit",
				Run: func(ctx context.Context, args []string) int {
					fmt.Println(metadata.Revision)
					return 0
				},
				RequiresRoot: false,
			},
		},
	}

	os.Exit(app.Run(context.Background(), os.Args))
}

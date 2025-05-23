package exteriord

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kofuk/premises/internal"
	"github.com/kofuk/premises/runner/internal/commands/exteriord/exterior"
	"github.com/kofuk/premises/runner/internal/commands/exteriord/outbound"
	"github.com/kofuk/premises/runner/internal/commands/exteriord/proc"
	"github.com/kofuk/premises/runner/internal/config"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/rpc"
)

func Run(ctx context.Context, args []string) int {
	signal.Ignore(syscall.SIGHUP)

	slog.Info("Starting premises-runner...", slog.String("protocol_version", internal.Version))

	config, err := config.Load()
	if err != nil {
		slog.Error("Unable to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, cancelFn := context.WithCancel(ctx)

	msgChan := make(chan outbound.OutboundMessage, 8)

	ob := outbound.NewServer(config.ControlPanel, config.AuthKey, msgChan)
	go ob.Start(ctx)

	stateStore := NewStateStore(NewLocalStorageStateBackend(env.DataPath("states.json")))

	rpcHandler := NewRPCHandler(rpc.DefaultServer, msgChan, stateStore, cancelFn)
	rpcHandler.Bind()

	e := exterior.New()

	setupTask := e.RegisterTask("Initialize Server",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--setup"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		))
	e.RegisterTask("System Statistics",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--sysstat"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask) // We can't use restricted user before setup task finished
	monitoring := e.RegisterTask("Game Monitoring Service",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--launcher"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	systemUpdate := e.RegisterTask("Keep System Up-to-date",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--update-packages"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Connector",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--connector"),
			proc.Restart(proc.RestartOnFailure),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	e.RegisterTask("Snapshot Service",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--snapshot-helper"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Clean Up",
		proc.NewProc(env.DataPath("bin/premises-runner"),
			proc.Args("--clean"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), monitoring, systemUpdate)

	e.Run(ctx)

	return 0
}

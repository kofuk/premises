package exteriord

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kofuk/premises/runner/commands/exteriord/exterior"
	"github.com/kofuk/premises/runner/commands/exteriord/outbound"
	"github.com/kofuk/premises/runner/commands/exteriord/proc"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
)

func Run(args []string) int {
	signal.Ignore(syscall.SIGHUP)

	config, err := config.Load()
	if err != nil {
		slog.Error("Unable to load config", slog.Any("error", err))
		os.Exit(1)
	}

	msgChan := make(chan outbound.OutboundMessage, 8)

	ob := outbound.NewServer(config.ControlPanel, config.AuthKey, msgChan)
	go ob.Start(context.TODO())

	stateStore := NewStateStore(NewLocalStorageStateBackend(fs.DataPath("states.json")))

	rpcHandler := NewRPCHandler(rpc.DefaultServer, msgChan, stateStore)
	rpcHandler.Bind()

	e := exterior.New()

	setupTask := e.RegisterTask("Initialize Server",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--setup"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		))
	e.RegisterTask("Syatem Statistics",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--sysstat"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask) // We can't use restricted user before setup task finished
	monitoring := e.RegisterTask("Game Monitoring Service",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--launcher"),
			proc.Restart(proc.RestartOnFailure),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	systemUpdate := e.RegisterTask("Keep System Up-to-date",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--update-packages"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Connector",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--connector"),
			proc.Restart(proc.RestartAlways),
			proc.UserType(proc.UserRestricted),
		), setupTask)
	e.RegisterTask("Snapshot Service",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--snapshot-helper"),
			proc.Restart(proc.RestartAlways),
			proc.RestartRandomDelay(),
			proc.UserType(proc.UserPrivileged),
		), setupTask)
	e.RegisterTask("Clean Up",
		proc.NewProc(fs.DataPath("bin/premises-runner"),
			proc.Args("--clean"),
			proc.Restart(proc.RestartNever),
			proc.UserType(proc.UserPrivileged),
		), monitoring, systemUpdate)

	e.Run(context.TODO())

	return 1
}

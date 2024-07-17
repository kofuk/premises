package connector

import (
	"context"
	"log/slog"
	"os"

	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/rpc"
)

func Run(args []string) int {
	config, err := config.Load()
	if err != nil {
		slog.Error("Error loading config", slog.Any("error", err))
		return 1
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	rpcHandler := NewRPCHandler(rpc.DefaultServer, config, cancelFn)
	rpcHandler.Bind()

	rpc.ToExteriord.Notify("proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	<-ctx.Done()

	return 0
}

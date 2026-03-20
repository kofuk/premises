package connector

import (
	"context"
	"log/slog"
	"os"

	"github.com/kofuk/premises/backend/runner/config"
	"github.com/kofuk/premises/backend/runner/rpc"
)

func Run(ctx context.Context, args []string) int {
	config, err := config.Load()
	if err != nil {
		slog.ErrorContext(ctx, "Error loading config", slog.Any("error", err))
		return 1
	}

	ctx, cancelFn := context.WithCancel(ctx)

	metrics := NewMetrics()

	rpcHandler := NewRPCHandler(rpc.DefaultServer, config, cancelFn, metrics)
	rpcHandler.Bind()

	rpc.ToExteriord.Notify(ctx, "proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	<-ctx.Done()

	return 0
}

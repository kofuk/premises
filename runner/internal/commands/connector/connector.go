package connector

import (
	"context"
	"log/slog"
	"os"

	"github.com/kofuk/premises/runner/internal/config"
	"github.com/kofuk/premises/runner/internal/rpc"
)

func Run(ctx context.Context, args []string) int {
	config, err := config.Load()
	if err != nil {
		slog.Error("Error loading config", slog.Any("error", err))
		return 1
	}

	ctx, cancelFn := context.WithCancel(ctx)

	rpcHandler := NewRPCHandler(rpc.DefaultServer, config, cancelFn)
	rpcHandler.Bind()

	rpc.ToExteriord.Notify(ctx, "proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	<-ctx.Done()

	return 0
}

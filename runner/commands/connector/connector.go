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

	rpc.ToExteriord.Notify("proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	ctx, cancelFn := context.WithCancel(context.Background())
	rpc.DefaultServer.RegisterNotifyMethod("base/stop", func(req *rpc.AbstractRequest) error {
		cancelFn()
		return nil
	})

	rpcHandler := NewRPCHandler(rpc.DefaultServer, config)
	rpcHandler.Bind()

	<-ctx.Done()

	return 0
}

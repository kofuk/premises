package connector

import (
	"context"
	"os"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/rpc"
)

func Run(ctx context.Context, config *runner.Config, args []string) int {
	ctx, cancelFn := context.WithCancel(ctx)

	metrics := NewMetrics()

	rpcHandler := NewRPCHandler(rpc.DefaultServer, config, cancelFn, metrics)
	rpcHandler.Bind()

	rpc.ToExteriord.Notify(ctx, "proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	<-ctx.Done()

	return 0
}

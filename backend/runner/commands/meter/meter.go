package meter

import (
	"context"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/rpc"
)

func Run(ctx context.Context, config *runner.Config, args []string) int {
	meterService := NewMeterService()
	meterService.Initialize()

	rpcHandler := NewRPCHandler(rpc.DefaultServer, meterService)
	rpcHandler.Bind()

	<-ctx.Done()

	return 0
}

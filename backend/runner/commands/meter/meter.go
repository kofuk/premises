package meter

import (
	"context"
	"log/slog"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/rpc"
)

func Run(ctx context.Context, config *runner.Config, args []string) int {
	meterService := NewMeterService()

	rpcHandler := NewRPCHandler(rpc.DefaultServer, meterService)
	rpcHandler.Bind()

	if err := meterService.Initialize(); err != nil {
		slog.ErrorContext(ctx, "Failed to initialize meter service", slog.Any("error", err))
		return 1
	}

	slog.InfoContext(ctx, "Meter is successfully initialized and ready to accept connections")

	<-ctx.Done()

	return 0
}

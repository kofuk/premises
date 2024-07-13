package connector

import (
	"log/slog"

	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/rpc"
)

func Run(args []string) int {
	config, err := config.Load()
	if err != nil {
		slog.Error("Error loading config", slog.Any("error", err))
		return 1
	}

	rpcHandler := NewRPCHandler(rpc.DefaultServer, config)
	rpcHandler.Bind()

	<-make(chan struct{})

	return 0
}

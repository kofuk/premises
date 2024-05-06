package mclauncher

import (
	"errors"
	"log/slog"

	"github.com/kofuk/premises/runner/commands/mclauncher/game"
	"github.com/kofuk/premises/runner/commands/mclauncher/world"
	"github.com/kofuk/premises/runner/config"
	"github.com/kofuk/premises/runner/metadata"
	"github.com/kofuk/premises/runner/rpc"
)

func Run(args []string) int {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	config, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		return 1
	}

	worldService := world.New(config.AWS.AccessKey, config.AWS.SecretKey, config.S3.Endpoint, config.S3.Bucket)

	launcher := game.NewLauncher(config, worldService)

	rpcHandler := NewRPCHandler(rpc.DefaultServer, launcher)
	rpcHandler.Bind()

	err = launcher.Launch()
	if err != nil {
		slog.Error("Unable to launch server", slog.Any("error", err))
	}

	if errors.Is(err, game.RestartRequested) {
		slog.Info("Restart...")

		return 100
	}

	return 0
}

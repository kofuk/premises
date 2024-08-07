package mclauncher

import (
	"errors"
	"log/slog"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/game"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/world"
	"github.com/kofuk/premises/runner/internal/config"
	"github.com/kofuk/premises/runner/internal/metadata"
	"github.com/kofuk/premises/runner/internal/rpc"
)

func Run(args []string) int {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	config, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		return 1
	}

	worldService := world.New(config.ControlPanel, config.AuthKey)

	launcher := game.NewLauncher(config, worldService)

	rpcHandler := NewRPCHandler(rpc.DefaultServer, launcher)
	rpcHandler.Bind()

	err = launcher.Launch()
	if err != nil {
		slog.Error("Unable to launch server", slog.Any("error", err))
	}

	if errors.Is(err, game.ErrRestartRequested) {
		slog.Info("Restart...")

		return 100
	}

	return 0
}

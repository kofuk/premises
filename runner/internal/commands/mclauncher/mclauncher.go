package mclauncher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/game"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/autoversion"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/eula"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverjar"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverproperties"
	middlewareWorld "github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world"
	worldService "github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world/service"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/repository"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/world"
	"github.com/kofuk/premises/runner/internal/config"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/metadata"
	"github.com/kofuk/premises/runner/internal/rpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const ScopeName = "github.com/kofuk/premises/runner/internal/commands/mclauncher"

func Run(ctx context.Context, args []string) int {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	config, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		return 1
	}

	worldService := world.New(config.ControlPanel, config.AuthKey)

	launcher := game.NewLauncher(ctx, config, worldService)

	rpcHandler := NewRPCHandler(rpc.DefaultServer, launcher)
	rpcHandler.Bind()

	err = launcher.Launch(ctx)
	if err != nil {
		slog.Error("Unable to launch server", slog.Any("error", err))
	}

	if errors.Is(err, game.ErrRestartRequested) {
		slog.Info("Restart...")

		return 100
	}

	return 0
}

func NewRun(ctx context.Context, args []string) int {
	slog.Info("Starting Premises Runner", slog.String("revision", metadata.Revision))

	config, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		return 1
	}

	// TODO: Support RPC endpoints

	settingsRepository := repository.NewConfigJSONSettingsRepository(config)
	stateRepository := repository.NewExteriorStateRepository(rpc.ToExteriord)

	launcher := core.NewLauncherCore(settingsRepository, env.DefaultEnvProvider, stateRepository)

	worldService := worldService.NewWorldService(config.ControlPanel, config.AuthKey, otelhttp.DefaultClient)
	launchermetaClient := launchermeta.NewLauncherMetaClient(launchermeta.WithHTTPClient(otelhttp.DefaultClient))

	launcher.Use(monitoring.NewMonitoringMiddleware())
	launcher.Use(eula.NewEulaMiddleware())
	launcher.Use(serverproperties.NewServerPropertiesMiddleware())
	launcher.Use(serverjar.NewServerJarMiddleware(
		launchermetaClient,
		otelhttp.DefaultClient,
	))
	launcher.Use(autoversion.NewAutoVersionMiddleware())
	launcher.Use(middlewareWorld.NewWorldMiddleware(worldService))

	err = launcher.Start(ctx)
	if errors.Is(err, core.ErrRestart) {
		slog.Info("Restart...")

		return 100
	} else if err != nil {
		slog.Error(fmt.Sprintf("Failed to start launcher: %v", err))
		return 1
	}

	return 0
}

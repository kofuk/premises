package mclauncher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/autoversion"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/eula"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverjar"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverproperties"
	middlewareWorld "github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world"
	worldService "github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world/service"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/quickundo"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/repository"
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

	settingsRepository := repository.NewConfigJSONSettingsRepository(config)
	stateRepository := repository.NewExteriorStateRepository(rpc.ToExteriord)

	launcher := core.NewLauncherCore(settingsRepository, env.DefaultEnvProvider, stateRepository)

	quickUndoService := quickundo.NewQuickUndoService(rpc.ToSnapshotHelper)
	quickUndoService.Register(launcher)

	worldService := worldService.NewWorldService(config.ControlPanel, config.AuthKey, otelhttp.DefaultClient)

	launchermetaOptions := []launchermeta.Option{
		launchermeta.WithHTTPClient(otelhttp.DefaultClient),
	}
	if config.GameConfig.Server.ManifestOverride != "" {
		launchermetaOptions = append(launchermetaOptions, launchermeta.WithManifestURL(config.GameConfig.Server.ManifestOverride))
	}
	launchermetaClient := launchermeta.NewLauncherMetaClient(
		launchermetaOptions...,
	)

	rconClient := rcon.NewRcon(rcon.NewRconExecutor("127.0.0.2:25575", "x"))

	launcher.Use(monitoring.NewMonitoringMiddleware(
		watchdog.NewLivenessWatchdog(),
		watchdog.NewOneTimeInitWatchdog(rconClient, config.GameConfig.Operators, config.GameConfig.Whitelist),
		watchdog.NewActivenessWatchdog(rconClient, config.GameConfig.Server.InactiveTimeout),
	))
	launcher.Use(eula.NewEulaMiddleware())
	launcher.Use(serverproperties.NewServerPropertiesMiddleware())
	launcher.Use(serverjar.NewServerJarMiddleware(
		launchermetaClient,
		otelhttp.DefaultClient,
	))
	launcher.Use(autoversion.NewAutoVersionMiddleware())
	launcher.Use(middlewareWorld.NewWorldMiddleware(worldService))

	rpcHandler := NewRPCHandler(rpc.DefaultServer, quickUndoService, rconClient)
	rpcHandler.Bind()

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

package mclauncher

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/common/mc/launchermeta"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/core"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/autoversion"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/eula"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/monitoring"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/serverjar"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/serverproperties"
	middlewareWorld "github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/world"
	worldService "github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/world/service"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/quickundo"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/rcon"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/repository"
	"github.com/kofuk/premises/backend/runner/env"
	"github.com/kofuk/premises/backend/runner/metadata"
	"github.com/kofuk/premises/backend/runner/rpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const ScopeName = "github.com/kofuk/premises/backend/runner/commands/mclauncher"

func Run(ctx context.Context, config *runner.Config, args []string) int {
	slog.InfoContext(ctx, "Starting Premises Runner", slog.String("revision", metadata.Revision))

	settingsRepository := repository.NewConfigJSONSettingsRepository(config)
	stateRepository := repository.NewExteriorStateRepository(rpc.ToExteriord)

	launcher := core.NewLauncherCore(settingsRepository, env.DefaultEnvProvider, stateRepository)

	quickUndoService := quickundo.NewQuickUndoService(rpc.ToSnapshotHelper)
	quickUndoService.Register(launcher)

	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	worldService := worldService.NewWorldService(config.ControlPlane, config.AuthKey, httpClient)

	launchermetaOptions := []launchermeta.Option{
		launchermeta.WithHTTPClient(httpClient),
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
		httpClient,
	))
	launcher.Use(autoversion.NewAutoVersionMiddleware())
	launcher.Use(middlewareWorld.NewWorldMiddleware(worldService))

	rpcHandler := NewRPCHandler(rpc.DefaultServer, quickUndoService, rconClient)
	rpcHandler.Bind()

	if err := launcher.Start(ctx); errors.Is(err, core.ErrRestart) {
		slog.InfoContext(ctx, "Restart...")

		return 100
	} else if err != nil {
		slog.ErrorContext(ctx, "Failed to start launcher", slog.Any("error", err))
		return 1
	}

	return 0
}

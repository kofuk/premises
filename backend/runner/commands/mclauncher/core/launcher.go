package core

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/kofuk/premises/backend/common/retry"
	coreUtil "github.com/kofuk/premises/backend/runner/commands/mclauncher/core/util"
	"github.com/kofuk/premises/backend/runner/rpc"
	"github.com/kofuk/premises/backend/runner/rpc/types"
	"github.com/kofuk/premises/backend/runner/system"
	"github.com/kofuk/premises/backend/runner/util"
)

func (l *LauncherCore) executeWithBackOff(c LauncherContext, cmdline []string, workDir string) error {
	backOffWaitTime := 2

	for {
		for _, listener := range l.beforeLaunchListeners {
			if err := listener(c); err != nil {
				slog.ErrorContext(c.Context(), "failed to execute before launch listener", slog.Any("error", err))
			}
		}

		slog.DebugContext(c.Context(), "Starting minecraft server...")
		handle, err := l.CommandExecutor.Start(c.Context(), cmdline[0], cmdline[1:], system.WithWorkingDir(workDir))
		if err != nil {
			slog.ErrorContext(c.Context(), "Failed to start Minecraft server", slog.Any("error", err))
		} else {
			retry.Retry(c.Context(), func(ctx context.Context) (retry.Void, error) {
				return retry.V, rpc.ToMeter.Call(c.Context(), "target/register", types.RegisterMeterTargetInput{
					Pid: handle.Pid,
				}, nil)
			}, 30*time.Second)

			err := handle.Wait()

			retry.Retry(c.Context(), func(ctx context.Context) (retry.Void, error) {
				return retry.V, rpc.ToMeter.Call(c.Context(), "target/unregister", types.RegisterMeterTargetInput{
					Pid: handle.Pid,
				}, nil)
			}, 30*time.Second)

			if err != nil {
				slog.ErrorContext(c.Context(), "Minecraft server exited with error", slog.Any("error", err))
			} else {
				slog.InfoContext(c.Context(), "Minecraft server exited")
				return nil
			}
		}

		timer := time.NewTimer(time.Duration(backOffWaitTime)*time.Second + time.Duration(rand.Float64()*500.0)*time.Millisecond)
		select {
		case <-c.Context().Done():
			return err
		case <-timer.C:
		}

		backOffWaitTime <<= 1
	}
}

func (l *LauncherCore) startMinecraft(c LauncherContext) error {
	serverPath := c.Settings().GetServerPath()
	workDir := c.Env().GetDataPath("gamedata")

	var commandLine []string
	if util.IsJar(c.Context(), serverPath) {
		// If this is JAR file, execute it with Java.
		memSize := c.Settings().GetAllowedMemSize(c.Context())
		commandLine = []string{
			coreUtil.FindJavaPath(c.Context()),
			fmt.Sprintf("-Xmx%dM", memSize),
			fmt.Sprintf("-Xms%dM", memSize),
		}

		if otlpEndpoint := c.Settings().GetOtlpEndpoint(); otlpEndpoint != "" {
			commandLine = append(
				commandLine,
				fmt.Sprintf("-javaagent:%s", c.Env().GetDataPath("resources/opentelemetry-javaagent.jar")),
				fmt.Sprintf("-Dotel.javaagent.configuration-file=%s", c.Env().GetDataPath("resources/opentelemetry-javaagent.properties")),
				"-Dotel.service.name=minecraft-server",
				fmt.Sprintf("-Dotel.exporter.otlp.endpoint=%s", otlpEndpoint),
				fmt.Sprintf("-Dotel.metric.export.interval=%d", max(c.Settings().GetMetricExportIntervalMs(), 1000)),
			)
		}

		commandLine = append(
			commandLine,
			"-jar",
			serverPath,
			"nogui",
		)
	} else {
		// If it is wrapper script or something, execute it directly.
		commandLine = []string{
			serverPath,
		}
	}

	return l.executeWithBackOff(c, commandLine, workDir)
}

package core

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	coreUtil "github.com/kofuk/premises/backend/runner/commands/mclauncher/core/util"
	"github.com/kofuk/premises/backend/runner/system"
	"github.com/kofuk/premises/backend/runner/util"
)

func (l *LauncherCore) executeWithBackOff(c LauncherContext, cmdline []string, workDir string) error {
	backOffWaitTime := 2

	var err error
	for {
		for _, listener := range l.beforeLaunchListeners {
			if err := listener(c); err != nil {
				slog.ErrorContext(c.Context(), "failed to execute before launch listener", slog.Any("error", err))
			}
		}

		err = l.CommandExecutor.Run(c.Context(), cmdline[0], cmdline[1:], system.WithWorkingDir(workDir))
		if err == nil {
			return nil
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

package core

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core/util"
	"github.com/kofuk/premises/runner/internal/system"
)

func executeWithBackOff(ctx context.Context, cmdline []string, workDir string) error {
	backOffWaitTime := 2

	var err error
	for {
		select {
		case <-ctx.Done():
			return err
		default:
		}

		err = system.Cmd(ctx, cmdline[0], cmdline[1:], system.WithWorkingDir(workDir))

		timer := time.NewTimer(time.Duration(backOffWaitTime)*time.Second + time.Duration(rand.Float64()*500.0)*time.Millisecond)
		select {
		case <-ctx.Done():
			return err
		case <-timer.C:
		}

		backOffWaitTime *= 2
	}
}

func startMinecraft(c LauncherContext) error {
	serverPath := c.Settings().GetServerPath()
	workDir := c.Settings().GetDataDir()

	var commandLine []string
	if util.IsJar(serverPath) {
		// If this is JAR file, execute it with Java.
		memSize := c.Settings().GetAllowedMemSize()
		desiredJavaVersion := c.Settings().GetDesiredJavaVersion()
		commandLine = []string{
			util.FindJavaPath(c.Context(), desiredJavaVersion),
			fmt.Sprintf("-Xmx%dM", memSize),
			fmt.Sprintf("-Xms%dM", memSize),
			"-jar",
			serverPath,
			"nogui",
		}
	} else {
		// If it is wrapper script or something, execute it directly.
		commandLine = []string{
			serverPath,
		}
	}

	return executeWithBackOff(c.Context(), commandLine, workDir)
}

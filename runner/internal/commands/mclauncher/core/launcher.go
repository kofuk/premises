package core

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core/util"
	"github.com/kofuk/premises/runner/internal/system"
)

func executeWithBackOff(ctx context.Context, commandExecutor system.CommandExecutor, cmdline []string, workDir string) error {
	backOffWaitTime := 2

	var err error
	for {
		err = commandExecutor.Run(ctx, cmdline[0], cmdline[1:], system.WithWorkingDir(workDir))
		if err == nil {
			return nil
		}

		timer := time.NewTimer(time.Duration(backOffWaitTime)*time.Second + time.Duration(rand.Float64()*500.0)*time.Millisecond)
		select {
		case <-ctx.Done():
			return err
		case <-timer.C:
		}

		backOffWaitTime <<= 1
	}
}

func (launcher *LauncherCore) startMinecraft(c *LauncherContext) error {
	serverPath := c.Settings().GetServerPath()
	workDir := c.Settings().GetDataDir()

	var commandLine []string
	if util.IsJar(serverPath) {
		// If this is JAR file, execute it with Java.
		memSize := c.Settings().GetAllowedMemSize()
		commandLine = []string{
			util.FindJavaPath(c.Context()),
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

	return executeWithBackOff(c.Context(), launcher.CommandExecutor, commandLine, workDir)
}

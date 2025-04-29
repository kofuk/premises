package watchdog

import (
	"fmt"
	"log/slog"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
)

type ActivenessWatchdog struct {
	rcon           *rcon.Rcon
	timeoutMinutes int
	lastActive     int
}

func NewActivenessWatchdog(rcon *rcon.Rcon, timeoutMinutes int) *ActivenessWatchdog {
	return &ActivenessWatchdog{
		rcon:           rcon,
		timeoutMinutes: timeoutMinutes,
		lastActive:     -1,
	}
}

var _ Watchdog = (*ActivenessWatchdog)(nil)

func (w *ActivenessWatchdog) Name() string {
	return "ActivenessWatchdog"
}

func (w *ActivenessWatchdog) Check(c core.LauncherContext, watchID int, status *Status) error {
	if w.timeoutMinutes <= 0 {
		// No timeout set, so no need to check
		return nil
	}

	if !status.Online {
		// Server is not started, so no need to check
		return nil
	}

	if w.lastActive < 0 {
		// Start counting when the server comes online
		w.lastActive = watchID
	}

	if watchID%60 != 0 {
		// Only check every 60 seconds
		return nil
	}

	output, err := w.rcon.List()
	if err != nil {
		return err
	}

	active := len(output.Players) != 0

	if active {
		w.lastActive = watchID
	} else {
		minutesSinceLastActive := (watchID - w.lastActive) / 60
		if minutesSinceLastActive > w.timeoutMinutes {
			slog.Debug(fmt.Sprintf("Server is inactive for %d minutes, stopping the server", minutesSinceLastActive))

			return w.rcon.Stop()
		}
	}

	return nil
}

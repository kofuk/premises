package watchdog

import (
	"log/slog"

	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	"github.com/kofuk/premises/runner/internal/exterior"
)

// This is not a real watchdog, but we'll use watchdog mechanism
// to initialize the server after the first start.
type OneTimeInitWatchdog struct {
	rcon       *rcon.Rcon
	prevOnline bool
	ops        []string
	whitelist  []string
}

var _ Watchdog = (*OneTimeInitWatchdog)(nil)

func NewOneTimeInitWatchdog(rcon *rcon.Rcon, ops []string, whitelist []string) *OneTimeInitWatchdog {
	return &OneTimeInitWatchdog{
		rcon:      rcon,
		ops:       ops,
		whitelist: whitelist,
	}
}

func (l *OneTimeInitWatchdog) Name() string {
	return "OneTimeInitWatchdog"
}

func (l *OneTimeInitWatchdog) Check(c core.LauncherContext, watchID int, status *Status) error {
	if !status.Online || l.prevOnline {
		return nil
	}

	l.prevOnline = true

	slog.Debug("Server became online, invoking one-time initialization")

	for _, user := range l.ops {
		if err := l.rcon.AddToOp(user); err != nil {

			return err
		}
	}
	for _, user := range l.whitelist {
		if err := l.rcon.AddToWhiteList(user); err != nil {
			return err
		}
	}

	data := &runner.StartedExtra{}
	data.ServerVersion = c.Settings().GetMinecraftVersion()
	data.World.Name = c.Settings().GetWorldName()
	seed, err := l.rcon.Seed()
	if err != nil {
		slog.Error("Failed to retrieve seed", slog.Any("error", err))
		// We don't want to fail the startup if we can't get the seed
	}
	data.World.Seed = string(seed)

	exterior.SendEvent(c.Context(), runner.Event{
		Type:    runner.EventStarted,
		Started: data,
	})

	return nil
}

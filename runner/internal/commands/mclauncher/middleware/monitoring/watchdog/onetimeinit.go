package watchdog

import (
	"context"
	"log/slog"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
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

func (l *OneTimeInitWatchdog) Check(ctx context.Context, watchID int, status *Status) error {
	if !status.Online || l.prevOnline {
		return nil
	}

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

	exterior.SendEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventRunning,
		},
	})

	l.prevOnline = true

	return nil
}

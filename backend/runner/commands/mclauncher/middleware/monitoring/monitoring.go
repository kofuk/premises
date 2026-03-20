package monitoring

import (
	"context"
	"log/slog"
	"time"

	"github.com/kofuk/premises/backend/common/entity"
	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/core"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/backend/runner/exterior"
)

type MonitoringMiddleware struct {
	watchdogs []watchdog.Watchdog
}

var _ core.Middleware = (*MonitoringMiddleware)(nil)

func NewMonitoringMiddleware(watchdogs ...watchdog.Watchdog) *MonitoringMiddleware {
	return &MonitoringMiddleware{
		watchdogs: watchdogs,
	}
}

func (m *MonitoringMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		exterior.SendEvent(c.Context(), runner.Event{
			Type: runner.EventStatus,
			Status: &runner.StatusExtra{
				EventCode: entity.EventLoading,
			},
		})

		ctx, cancel := context.WithCancel(c.Context())
		defer cancel()

		go func(ctx context.Context) {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			watchID := 0

			prevOnline := false

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				status := &watchdog.Status{}
				for _, w := range m.watchdogs {
					if err := w.Check(c, watchID, status); err != nil {
						slog.ErrorContext(ctx, "Watchdog raised an error", slog.String("watchdog", w.Name()), slog.Any("error", err))
					}
				}

				if prevOnline != status.Online {
					prevOnline = status.Online
					if status.Online {
						exterior.SendEvent(c.Context(), runner.Event{
							Type: runner.EventStatus,
							Status: &runner.StatusExtra{
								EventCode: entity.EventRunning,
							},
						})
					} else {
						exterior.SendEvent(c.Context(), runner.Event{
							Type: runner.EventStatus,
							Status: &runner.StatusExtra{
								EventCode: entity.EventLoading,
							},
						})
					}
				}

				watchID++
			}
		}(ctx)

		return next(c)
	}
}

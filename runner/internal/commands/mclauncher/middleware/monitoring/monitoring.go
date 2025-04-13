package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/runner/internal/exterior"
)

type MonitoringMiddleware struct {
	watchdogs []watchdog.Watchdog
}

var _ core.Middleware = (*MonitoringMiddleware)(nil)

func NewMonitoringMiddleware() *MonitoringMiddleware {
	return &MonitoringMiddleware{}
}

func (m *MonitoringMiddleware) AddWatchdog(w watchdog.Watchdog) {
	m.watchdogs = append(m.watchdogs, w)
}

func (m *MonitoringMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c *core.LauncherContext) error {
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

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				status := &watchdog.Status{}
				for _, w := range m.watchdogs {
					if err := w.Check(ctx, watchID, status); err != nil {
						slog.Error(fmt.Sprintf("Watchdog %s raised an error: %v", w.Name(), err))
					}
				}

				watchID++
			}
		}(ctx)

		return next(c)
	}
}

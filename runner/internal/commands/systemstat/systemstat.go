package systemstat

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/exterior"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/system"
)

func sendSysstat(ctx context.Context) error {
	cpuStat, err := system.NewCPUUsage()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
		}

		usage, err := cpuStat.Percent()
		if err != nil {
			slog.Error("Failed to retrieve CPU usage", slog.Any("error", err))
			continue
		}

		exterior.SendEvent(ctx, runner.Event{
			Type: runner.EventSysstat,
			Sysstat: &runner.SysstatExtra{
				CPUUsage: usage,
				Time:     time.Now().UnixMilli(),
			},
		})
	}
}

func Run(ctx context.Context, args []string) int {
	rpc.ToExteriord.Notify(ctx, "proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	ctx, cancelFn := context.WithCancel(ctx)

	rpc.DefaultServer.RegisterNotifyMethod("base/stop", func(ctx context.Context, req *rpc.AbstractRequest) error {
		cancelFn()
		return nil
	})

	if err := sendSysstat(ctx); err != nil {
		slog.Error("Unable to send sysstat", slog.Any("error", err))
		return 1
	}
	return 0
}

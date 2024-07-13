package systemstat

import (
	"context"
	"log/slog"
	"os"
	"time"

	entity "github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/system"
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

		exterior.SendEvent(entity.Event{
			Type: entity.EventSysstat,
			Sysstat: &entity.SysstatExtra{
				CPUUsage: usage,
				Time:     time.Now().UnixMilli(),
			},
		})
	}
}

func Run(args []string) int {
	rpc.ToExteriord.Notify("proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	ctx, cancelFn := context.WithCancel(context.Background())

	rpc.DefaultServer.RegisterNotifyMethod("base/stop", func(req *rpc.AbstractRequest) error {
		cancelFn()
		return nil
	})

	if err := sendSysstat(ctx); err != nil {
		slog.Error("Unable to send sysstat", slog.Any("error", err))
		return 1
	}
	return 0
}

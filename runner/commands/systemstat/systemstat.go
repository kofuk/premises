package systemstat

import (
	"log/slog"
	"os"
	"time"

	entity "github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/system"
)

func Run(args []string) int {
	cpuStat, err := system.NewCPUUsage()
	if err != nil {
		slog.Error("Failed to initialize CPU usage", slog.Any("error", err))
		os.Exit(1)
	}

	ticker := time.NewTicker(time.Second)

	for range ticker.C {
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

	return 1
}

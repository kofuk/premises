package systemstat

import (
	"log/slog"
	"os"
	"time"

	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
)

func Run() {
	cpuStat, err := systemutil.NewCPUUsage()
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

		data := entity.Event{
			Type: entity.EventSysstat,
			Sysstat: &entity.SysstatExtra{
				CPUUsage: usage,
				Time:     time.Now().UnixMilli(),
			},
		}
		if err := exterior.SendMessage("systemStat", data); err != nil {
			slog.Error("Unable to write system stat", slog.Any("error", err))
		}
	}
}

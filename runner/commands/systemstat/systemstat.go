package systemstat

import (
	"time"

	entity "github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
	log "github.com/sirupsen/logrus"
)

func Run() {
	cpuStat, err := systemutil.NewCPUUsage()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize CPU usage")
	}

	ticker := time.NewTicker(2 * time.Second)

	for range ticker.C {
		usage, err := cpuStat.Percent()
		if err != nil {
			log.WithError(err).Error("Failed to retrieve CPU usage")
			continue
		}

		data := entity.Event{
			Type: entity.EventSysstat,
			Sysstat: &entity.SysstatExtra{
				CPUUsage: usage,
			},
		}
		if err := exterior.SendMessage("systemStat", data); err != nil {
			log.Error(err)
		}
	}
}

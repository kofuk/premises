package systemstat

import (
	"encoding/json"
	"time"

	"github.com/kofuk/premises/runner/config"
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

		statusData := config.StatusData{
			Type:     config.StatusTypeSystemStat,
			Shutdown: false,
			HasError: false,
			CPUUsage: usage,
		}
		statusJson, _ := json.Marshal(statusData)

		if err := exterior.SendMessage(exterior.Message{
			Type:     "systemStat",
			UserData: string(statusJson),
		}); err != nil {
			log.Error(err)
		}
	}
}

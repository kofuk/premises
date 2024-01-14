package config

import (
	"path/filepath"
	"sync"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/config"
)

type PMCMContext struct {
	Cfg            *runner.Config
	StatusChannels []chan string
	ChannelMutex   sync.Mutex
	LastStatus     string
}

type StatusType string

const (
	StatusTypeLegacyEvent StatusType = "legacyEvent"
	StatusTypeSystemStat  StatusType = "systemStat"
)

func (ctx *PMCMContext) LocateWorldData(path string) string {
	return ctx.LocateDataFile(filepath.Join("gamedata", path))
}

func (ctx *PMCMContext) LocateServer(serverName string) string {
	return ctx.LocateDataFile(filepath.Join("servers.d", serverName+".jar"))
}

func (ctx *PMCMContext) LocateDataFile(path string) string {
	return filepath.Join("/opt/premises", path)
}

func LoadConfig(ctx *PMCMContext) error {
	config, err := config.Load()
	if err != nil {
		return err
	}
	ctx.Cfg = config
	return nil
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/kofuk/premises/common/entity/runner"
)

type PMCMContext struct {
	Cfg            runner.GameConfig
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
	data, err := os.ReadFile(ctx.LocateDataFile("config.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &ctx.Cfg); err != nil {
		return err
	}
	return nil
}

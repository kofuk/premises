package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	AllocSize int    `json:"allocSize"`
	AuthKey   string `json:"authKey"`
	Server    struct {
		Version     string `json:"name"`
		DownloadUrl string `json:"downloadUrl"`
	} `json:"server"`
	World struct {
		ShouldGenerate bool   `json:"shouldGenerate"`
		Name           string `json:"name"`
		GenerationId   string `json:"generationId"`
		Seed           string `json:"seed"`
		LevelType      string `json:"levelType"`
		Difficulty     string `json:"difficulty"`
		UseCache       bool   `json:"useCache"`
	} `json:"world"`
	Motd      string   `json:"motd"`
	Operators []string `json:"operators"`
	Whitelist []string `json:"whitelist"`
	AWS       struct {
		AccessKey string
		SecretKey string
	} `json:"aws"`
	S3 struct {
		Endpoint string `json:"endpoint"`
		Bucket   string `json:"bucket"`
	} `json:"s3"`
}

type PMCMContext struct {
	Cfg            Config
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

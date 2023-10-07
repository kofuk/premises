package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	AllocSize int    `json:"allocSize"`
	AuthKey   string `json:"authKey"`
	Locale    string `json:"locale"`
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
	Mega      struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		FolderName string `json:"folderName"`
	} `json:"mega"`
}

type PMCMContext struct {
	Cfg            Config
	StatusChannels []chan string
	ChannelMutex   sync.Mutex
	LastStatus     string
	Localize       *i18n.Bundle
}

func (ctx *PMCMContext) L(msgId string) string {
	if ctx.Localize == nil {
		log.Error("i18n data is not initizlied")
		return msgId
	}
	locale := ctx.Cfg.Locale
	if locale == "" {
		locale = "en"
	}
	localizer := i18n.NewLocalizer(ctx.Localize, locale)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
	if err != nil {
		log.WithError(err).Error("Error loading localized message. Fallback to \"en\"")
		localizer := i18n.NewLocalizer(ctx.Localize, "en")
		msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
		if err != nil {
			log.WithError(err).Error("Error loading localized message (fallback)")
			return msgId
		}
		return msg
	}
	return msg
}

func (ctx *PMCMContext) NotifyStatus(status string) {
	ctx.ChannelMutex.Lock()
	defer ctx.ChannelMutex.Unlock()
	ctx.LastStatus = status
	for _, ch := range ctx.StatusChannels {
		ch <- status
	}
}

func (ctx *PMCMContext) AddStatusChannel(ch chan string) {
	ctx.ChannelMutex.Lock()
	defer ctx.ChannelMutex.Unlock()

	ctx.StatusChannels = append(ctx.StatusChannels, ch)
}

func (ctx *PMCMContext) RemoveStatusChannel(ch chan string) {
	ctx.ChannelMutex.Lock()
	defer ctx.ChannelMutex.Unlock()

	for i, c := range ctx.StatusChannels {
		if c == ch {
			if i != len(ctx.StatusChannels)-1 {
				ctx.StatusChannels[i] = ctx.StatusChannels[len(ctx.StatusChannels)-1]
			}
			ctx.StatusChannels = ctx.StatusChannels[0 : len(ctx.StatusChannels)-1]
			break
		}
	}
}

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

package gameconfig

import (
	"context"
	"encoding/base32"
	"errors"
	"math/rand"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/controlpanel/mcversions"
)

type GameConfig struct {
	RemoveMe  bool   `json:"removeMe"`
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

func New() *GameConfig {
	result := GameConfig{
		RemoveMe: true,
	}
	result.World.LevelType = "default"
	result.World.Difficulty = "normal"

	return &result
}

func (gc *GameConfig) SetServerVersion(version string) error {
	dlUrl, err := mcversions.GetDownloadUrl(context.TODO(), version)
	if err != nil {
		return err
	}

	gc.Server.Version = version
	gc.Server.DownloadUrl = dlUrl

	return nil
}

func (gc *GameConfig) SetAllocFromAvailableMemSize(memSizeMiB int) error {
	if memSizeMiB < 1024 {
		return errors.New("Memory too small")
	}
	if memSizeMiB > 2048 {
		gc.AllocSize = memSizeMiB - 1024
	} else {
		gc.AllocSize = memSizeMiB - 512
	}
	return nil
}

const authKeySymbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-=\\`!@#$%^&*()_+|~[]{};:'\",./<>?"

var randSource = rand.NewSource(time.Now().UnixNano())

func (gc *GameConfig) GenerateAuthKey() string {
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	result := encoder.EncodeToString(securecookie.GenerateRandomKey(30))
	gc.AuthKey = result
	return result
}

func (gc *GameConfig) SetWorld(worldName string, generationId string) {
	gc.World.Name = worldName
	gc.World.GenerationId = generationId
}

func (gc *GameConfig) GenerateWorld(worldName, seed string) {
	gc.World.ShouldGenerate = true
	gc.World.Name = worldName
	gc.World.Seed = seed
}

func (gc *GameConfig) SetMotd(motd string) {
	gc.Motd = motd
}

func (gc *GameConfig) SetLevelType(levelType string) error {
	if levelType != "default" && levelType != "flat" && levelType != "largeBiomes" && levelType != "amplified" && levelType != "buffet" {
		return errors.New("Unknown level type")
	}
	gc.World.LevelType = levelType
	return nil
}

func (gc *GameConfig) SetDifficulty(difficulty string) error {
	if difficulty != "peaceful" && difficulty != "easy" && difficulty != "normal" && difficulty != "hard" {
		return errors.New("Unknown difficulty")
	}
	gc.World.Difficulty = difficulty
	return nil
}

func (gc *GameConfig) UseCache(useCache bool) {
	gc.World.UseCache = useCache
}

func appendIfNotIncluded(to []string, elm string) ([]string, bool) {
	for _, v := range to {
		if v == elm {
			return to, false
		}
	}
	return append(to, elm), true
}

func (gc *GameConfig) SetOperators(ops []string) {
	for _, op := range ops {
		gc.Operators, _ = appendIfNotIncluded(gc.Operators, op)
		gc.Whitelist, _ = appendIfNotIncluded(gc.Whitelist, op)
	}
}

func (gc *GameConfig) SetWhitelist(wlist []string) {
	for _, wl := range wlist {
		gc.Whitelist, _ = appendIfNotIncluded(gc.Whitelist, wl)
	}
}

func (gc *GameConfig) SetMegaCredential(email, password string) {
	gc.Mega.Email = email
	gc.Mega.Password = password
}

func (gc *GameConfig) SetLocale(locale string) {
	gc.Locale = locale
}

func (gc *GameConfig) SetFolderName(name string) {
	gc.Mega.FolderName = name
}

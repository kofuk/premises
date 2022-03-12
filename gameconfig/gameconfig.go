package gameconfig

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/kofuk/premises/mcversions"
)

type GameConfig struct {
	RemoveMe  bool   `json:"removeMe"`
	AllocSize int    `json:"allocSize"`
	AuthKey   string `json:"authKey"`
	Server    struct {
		Version     string `json:"name"`
		DownloadUrl string `json:"downloadUrl"`
	} `json:"server"`
	World struct {
		ShouldGenerate bool   `json:"shouldGenerate"`
		Name           string `json:"name"`
		Generation     string `json:"generation"`
		Seed           string `json:"seed"`
	} `json:"world"`
	Motd       string   `json:"motd"`
	LevelType  string   `json:"levelType"`
	Operators  []string `json:"operators"`
	Whitelist  []string `json:"whitelist"`
	Difficulty string   `json:"difficulty"`
	Mega       struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	} `json:"mega"`
}

func New() *GameConfig {
	return &GameConfig{
		RemoveMe:   true,
		LevelType:  "default",
		Difficulty: "normal",
	}
}

func (gc *GameConfig) SetServerVersion(version string) error {
	dlUrl, err := mcversions.GetDownloadUrl(version)
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
	length := int(randSource.Int63()%17 + 11)
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		builder.WriteByte(authKeySymbols[int(randSource.Int63())%len(authKeySymbols)])
	}
	result := builder.String()
	gc.AuthKey = result
	return result
}

func (gc *GameConfig) SetWorld(worldName string, generation string) {
	gc.World.Name = worldName
	gc.World.Generation = generation
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
	gc.LevelType = levelType
	return nil
}

func (gc *GameConfig) SetDifficulty(difficulty string) error {
	if difficulty != "peaceful" && difficulty != "easy" && difficulty != "normal" && difficulty != "hard" {
		return errors.New("Unknown difficulty")
	}
	gc.Difficulty = difficulty
	return nil
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

package gameconfig

import (
	"errors"
	"fmt"

	"github.com/kofuk/premises/controlpanel/internal/launcher"
	"golang.org/x/exp/slices"
)

type Config struct {
	C launcher.LaunchConfig
}

func New() *Config {
	result := new(Config)
	result.C.World.LevelType = "default"
	result.C.World.Difficulty = "normal"

	return result
}

func (c *Config) SetServer(version, downloadURL string) {
	c.C.Server.Version = version
	c.C.Server.DownloadUrl = downloadURL
}

var (
	ErrMemoryTooSmall = errors.New("memory too small")
)

func calculateMemSizeForGame(availableSizeMiB int) (int, error) {
	if availableSizeMiB < 2048 {
		return 0, ErrMemoryTooSmall
	}
	return availableSizeMiB - 1024, nil
}

func (c *Config) SetDetectServerVersion(detect bool) {
	c.C.Server.PreferDetected = detect
}

func (c *Config) SetWorld(worldName string, generationId string) error {
	if worldName == "" || generationId == "" {
		return fmt.Errorf("either worldName or generationId is empty")
	}
	c.C.World.Name = worldName
	c.C.World.GenerationID = generationId
	return nil
}

func (c *Config) GenerateWorld(worldName, seed string) {
	c.C.World.ShouldGenerate = true
	c.C.World.Name = worldName
	c.C.World.Seed = seed
}

func (c *Config) SetMotd(motd string) {
	c.C.Server.Motd = motd
}

func isValidLevelType(levelType string) bool {
	return slices.Contains([]string{"default", "flat", "largeBiomes", "amplified", "buffet"}, levelType)
}

func (c *Config) SetLevelType(levelType string) error {
	if !isValidLevelType(levelType) {
		return errors.New("unknown level type")
	}
	c.C.World.LevelType = levelType
	return nil
}

func isValidDifficulty(difficulty string) bool {
	return slices.Contains([]string{"peaceful", "easy", "normal", "hard"}, difficulty)
}

func (c *Config) SetDifficulty(difficulty string) error {
	if !isValidDifficulty(difficulty) {
		return errors.New("unknown difficulty")
	}
	c.C.World.Difficulty = difficulty
	return nil
}

func addToSlice[T comparable](to []T, elm T) []T {
	for _, v := range to {
		if v == elm {
			return to
		}
	}
	return append(to, elm)
}

func (c *Config) SetOperators(ops []string) {
	for _, op := range ops {
		c.C.Server.Operators = addToSlice(c.C.Server.Operators, op)
		c.C.Server.Whitelist = addToSlice(c.C.Server.Whitelist, op)
	}
}

func (c *Config) SetWhitelist(wlist []string) {
	for _, wl := range wlist {
		c.C.Server.Whitelist = addToSlice(c.C.Server.Whitelist, wl)
	}
}

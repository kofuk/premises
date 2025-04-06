package launcher

import (
	"encoding/base32"

	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/internal/entity/runner"
)

type LaunchServerConfig struct {
	PreferDetected     bool
	Version            string
	DownloadUrl        string
	ManifestOverride   string
	CustomCommand      []string
	ServerPropOverride map[string]string
	JavaVersion        int
	InactiveTimeout    int
	// TODO: Move this to world config
	Motd      string
	Operators []string
	Whitelist []string
}

type LaunchWorldConfig struct {
	ShouldGenerate bool
	Name           string
	GenerationID   string
	Seed           string
	LevelType      string
	Difficulty     string
}

type LaunchConfig struct {
	MachineType string
	Server      LaunchServerConfig
	World       LaunchWorldConfig
}

func generateAuthKey() string {
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	result := encoder.EncodeToString(securecookie.GenerateRandomKey(30))
	return result
}

func (c *LaunchConfig) Validate() error {
	return nil
}

func (c *LaunchConfig) ToRunnerConfig(cfg *config.Config) (*runner.Config, error) {
	result := &runner.Config{}

	// server config
	result.GameConfig.Server.PreferDetected = c.Server.PreferDetected
	result.GameConfig.Server.Version = c.Server.Version
	result.GameConfig.Server.DownloadUrl = c.Server.DownloadUrl
	result.GameConfig.Server.ManifestOverride = c.Server.ManifestOverride
	result.GameConfig.Server.CustomCommand = c.Server.CustomCommand
	result.GameConfig.Server.ServerPropOverride = c.Server.ServerPropOverride
	result.GameConfig.Server.JavaVersion = c.Server.JavaVersion
	result.GameConfig.Server.InactiveTimeout = c.Server.InactiveTimeout
	result.GameConfig.Motd = c.Server.Motd

	// world config
	result.GameConfig.World.ShouldGenerate = c.World.ShouldGenerate
	result.GameConfig.World.Name = c.World.Name
	result.GameConfig.World.GenerationId = c.World.GenerationID
	result.GameConfig.World.Seed = c.World.Seed
	result.GameConfig.World.LevelType = c.World.LevelType
	result.GameConfig.World.Difficulty = c.World.Difficulty

	// misc config
	result.GameConfig.Operators = c.Server.Operators
	result.GameConfig.Whitelist = c.Server.Whitelist

	result.AuthKey = generateAuthKey()
	result.ControlPanel = cfg.Origin

	return result, nil
}

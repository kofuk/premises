package repository

import (
	"fmt"
	"maps"

	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/system"
	"golang.org/x/exp/slog"
)

type ConfigJSONSettingsRepository struct {
	serverPath                string
	minecraftVersion          string
	autoVersionEnabled        bool
	worldName                 string
	worldResourceID           string
	newWorld                  bool
	motd                      string
	difficulty                string
	levelType                 string
	seed                      string
	serverPropertiesOverrides map[string]string
}

var _ core.SettingsRepository = (*ConfigJSONSettingsRepository)(nil)

func NewConfigJSONSettingsRepository(config *runner.Config) *ConfigJSONSettingsRepository {
	result := &ConfigJSONSettingsRepository{}
	result.initialize(config)
	return result
}

func (r *ConfigJSONSettingsRepository) initialize(config *runner.Config) {
	r.minecraftVersion = config.GameConfig.Server.Version
	r.autoVersionEnabled = config.GameConfig.Server.PreferDetected
	r.worldName = config.GameConfig.World.Name
	r.worldResourceID = config.GameConfig.World.GenerationId
	r.newWorld = config.GameConfig.World.ShouldGenerate
	r.motd = config.GameConfig.Motd
	r.difficulty = config.GameConfig.World.Difficulty
	r.levelType = config.GameConfig.World.LevelType
	r.seed = config.GameConfig.World.Seed
	r.serverPropertiesOverrides = make(map[string]string)
	maps.Copy(r.serverPropertiesOverrides, config.GameConfig.Server.ServerPropOverride)
}

func getAllowedSizeMiB() int {
	if env.IsDevEnv() {
		return 1024
	}

	totalMem, err := system.GetTotalMemory()
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to get total memory: %v", err))
		return 1024
	}
	return totalMem/1024/1024 - 1024
}

func (r *ConfigJSONSettingsRepository) GetAllowedMemSize() int {
	return getAllowedSizeMiB()
}

func (r *ConfigJSONSettingsRepository) GetServerPath() string {
	return r.serverPath
}

func (r *ConfigJSONSettingsRepository) SetServerPath(path string) {
	r.serverPath = path
}

func (r *ConfigJSONSettingsRepository) GetMinecraftVersion() string {
	return r.minecraftVersion
}

func (r *ConfigJSONSettingsRepository) SetMinecraftVersion(version string) {
	r.minecraftVersion = version
}

func (r *ConfigJSONSettingsRepository) AutoVersionEnabled() bool {
	return r.autoVersionEnabled
}

func (r *ConfigJSONSettingsRepository) GetWorldName() string {
	return r.worldName
}

func (r *ConfigJSONSettingsRepository) GetWorldResourceID() string {
	return r.worldResourceID
}

func (r *ConfigJSONSettingsRepository) SetWorldResourceID(resourceID string) {
	r.worldResourceID = resourceID
}

func (r *ConfigJSONSettingsRepository) IsNewWorld() bool {
	return r.newWorld
}

func (r *ConfigJSONSettingsRepository) GetMotd() string {
	return r.motd
}

func (r *ConfigJSONSettingsRepository) GetDifficulty() string {
	return r.difficulty
}

func (r *ConfigJSONSettingsRepository) GetLevelType() string {
	return r.levelType
}

func (r *ConfigJSONSettingsRepository) GetSeed() string {
	return r.seed
}

func (r *ConfigJSONSettingsRepository) ServerPropertiesOverrides() map[string]string {
	return r.serverPropertiesOverrides
}

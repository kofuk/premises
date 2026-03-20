package repository

import (
	"context"
	"fmt"
	"maps"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/core"
	"github.com/kofuk/premises/backend/runner/env"
	"github.com/kofuk/premises/backend/runner/system"
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
	otlpEndpoint              string
	metricExportIntervalMs    int
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
	r.otlpEndpoint = config.Observability.OtlpEndpoint
	r.metricExportIntervalMs = config.Observability.MetricExportIntervalMs
}

func getAllowedSizeMiB(ctx context.Context) int {
	if env.IsDevEnv() {
		return 1024
	}

	totalMem, err := system.GetTotalMemory()
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Unable to get total memory: %v", err))
		return 1024
	}
	return totalMem/1024/1024 - 1024
}

func (r *ConfigJSONSettingsRepository) GetAllowedMemSize(ctx context.Context) int {
	return getAllowedSizeMiB(ctx)
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

func (r *ConfigJSONSettingsRepository) GetOtlpEndpoint() string {
	return r.otlpEndpoint
}

func (r *ConfigJSONSettingsRepository) GetMetricExportIntervalMs() int {
	return r.metricExportIntervalMs
}

package core

import "context"

//go:generate go tool mockgen -destination settings_mock.go -package core . SettingsRepository

type SettingsRepository interface {
	GetAllowedMemSize(ctx context.Context) int
	GetServerPath() string
	SetServerPath(path string)
	GetMinecraftVersion() string
	SetMinecraftVersion(version string)
	AutoVersionEnabled() bool
	GetWorldName() string
	GetWorldResourceID() string
	SetWorldResourceID(resourceID string)
	IsNewWorld() bool
	GetMotd() string
	GetDifficulty() string
	GetLevelType() string
	GetSeed() string
	ServerPropertiesOverrides() map[string]string
	GetOtlpEndpoint() string
	GetMetricExportIntervalMs() int
}

package core

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination settings_mock.go -package core . SettingsRepository

type SettingsRepository interface {
	GetAllowedMemSize() int
	GetServerPath() string
	SetServerPath(path string)
	GetMinecraftVersion() string
	SetMinecraftVersion(version string)
	AutoVersionEnabled() bool
	GetWorldName() string
	GetWorldResourceID() string
	SetWorldResourceID(resourceID string)
	IsNewWorld() bool
}

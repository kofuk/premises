package core

type SettingsRepository interface {
	GetAllowedMemSize() int
	GetServerPath() string
	GetDesiredJavaVersion() int
	GetDataDir() string
}

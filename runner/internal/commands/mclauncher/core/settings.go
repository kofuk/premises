package core

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination settings_mock.go -package core . SettingsRepository

type SettingsRepository interface {
	GetAllowedMemSize() int
	GetServerPath() string
	GetDesiredJavaVersion() int
	GetDataDir() string
}

package env

//go:generate go tool mockgen -destination env_mock.go -package env . EnvProvider

import (
	"os"
	"path/filepath"
)

func IsDevEnv() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

type EnvProvider interface {
	GetDataPath(path ...string) string
	GetTempDir() string
}

type defaultEnvProvider struct{}

var DefaultEnvProvider EnvProvider = &defaultEnvProvider{}

func (p *defaultEnvProvider) GetDataPath(path ...string) string {
	return filepath.Join(BaseDir, filepath.Join(path...))
}

func (p *defaultEnvProvider) GetTempDir() string {
	return p.GetDataPath("tmp")
}

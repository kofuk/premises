package env

import (
	"os"
	"path/filepath"
)

const (
	BaseDir = "/opt/premises"
)

func DataPath(path ...string) string {
	return filepath.Join(BaseDir, filepath.Join(path...))
}

func LocateServer(serverName string) string {
	return DataPath(filepath.Join("servers.d", serverName+".jar"))
}

func GetTempDir() string {
	return DataPath("tmp")
}

func MkdirTemp(envProvider EnvProvider) (string, error) {
	return os.MkdirTemp(envProvider.GetTempDir(), "premises-temp")
}

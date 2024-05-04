package fs

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

func RemoveIfExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}

func GetTempDir() string {
	return DataPath("tmp")
}

func MkdirTemp() (string, error) {
	return os.MkdirTemp(GetTempDir(), "premises-temp")
}

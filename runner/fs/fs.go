package fs

import (
	"os"
	"path/filepath"
)

const (
	baseDir = "/opt/premises"
)

func LocateWorldData(path string) string {
	return LocateDataFile(filepath.Join("gamedata", path))
}

func LocateServer(serverName string) string {
	return LocateDataFile(filepath.Join("servers.d", serverName+".jar"))
}

func LocateDataFile(path string) string {
	return filepath.Join(baseDir, path)
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

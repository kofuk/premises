package fs

import (
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

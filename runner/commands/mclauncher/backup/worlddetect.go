package backup

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kofuk/premises/runner/fs"
)

var (
	worldNotFound = errors.New("world not found")
)

func levelDatExists(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	levelDatFound := false
	for _, ent := range entries {
		if ent.Name() == "level.dat" {
			levelDatFound = true
			break
		}
	}

	return levelDatFound, nil
}

func isLikelyWorldsDir(dir string) (bool, error) {
	if _, err := os.Stat(filepath.Join(dir, "level.dat")); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func locateWorldsDir(tmpDir string) (string, error) {
	retryCount := 0

retry:
	if retryCount > 3 {
		return "", worldNotFound
	}

	isWorldDir, err := isLikelyWorldsDir(tmpDir)
	if err != nil {
		return "", err
	}
	if isWorldDir {
		return tmpDir, nil
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", worldNotFound
	}

	if len(entries) == 1 {
		dir := filepath.Join(tmpDir, entries[0].Name())
		isWorldDir, err := isLikelyWorldsDir(dir)
		if err != nil {
			return "", err
		}
		if isWorldDir {
			return dir, nil
		} else {
			tmpDir = dir
			retryCount++
			goto retry
		}
	}

	return "", worldNotFound
}

func moveWorldDataToGameDir(tmpDir string) error {
	dir, err := locateWorldsDir(tmpDir)
	if err != nil {
		return err
	}

	worldDir := fs.LocateWorldData("world")
	if _, err := os.Stat(worldDir); err != nil {
		if err := os.RemoveAll(worldDir); err != nil {
			return err
		}
	}
	netherDir := fs.LocateWorldData("world_nether")
	if _, err := os.Stat(netherDir); err != nil {
		if err := os.RemoveAll(netherDir); err != nil {
			return err
		}
	}
	endDir := fs.LocateWorldData("world_the_end")
	if _, err := os.Stat(endDir); err != nil {
		if err := os.RemoveAll(endDir); err != nil {
			return err
		}
	}

	slog.Info("Copying overworld data", slog.String("from", dir), slog.String("to", worldDir))
	cmd := exec.Command("mv", "--", dir, worldDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

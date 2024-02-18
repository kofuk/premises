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

type worldDir struct {
	defWorld string
	nether   string
	theEnd   string
}

func newVanillaWorld(defWorld string) *worldDir {
	return &worldDir{
		defWorld: defWorld,
	}
}

func newModWorld(defWorld, nether, theEnd string) *worldDir {
	return &worldDir{
		defWorld: defWorld,
		nether:   nether,
		theEnd:   theEnd,
	}
}

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

func locateWorldsDir(tmpDir string) (*worldDir, error) {
	retryCount := 0

retry:
	if retryCount > 3 {
		return nil, worldNotFound
	}

	isWorldDir, err := isLikelyWorldsDir(tmpDir)
	if err != nil {
		return nil, err
	}
	if isWorldDir {
		return newVanillaWorld(tmpDir), nil
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, worldNotFound
	}

	if len(entries) == 1 {
		dir := filepath.Join(tmpDir, entries[0].Name())
		isWorldDir, err := isLikelyWorldsDir(dir)
		if err != nil {
			return nil, err
		}
		if isWorldDir {
			return newVanillaWorld(dir), nil
		} else {
			tmpDir = dir
			retryCount++
			goto retry
		}
	}

	if len(entries) >= 3 {
		haveWorld := false
		haveWorldNether := false
		haveWorldEnd := false

		for _, ent := range entries {
			if ent.Name() == "world" {
				haveWorld = true
			} else if ent.Name() == "world_nether" {
				haveWorldNether = true
			} else if ent.Name() == "world_the_end" {
				haveWorldEnd = true
			}
		}

		if haveWorld && (!haveWorldEnd || !haveWorldNether) {
			dir := filepath.Join(tmpDir, "world")
			isWorldDir, err := isLikelyWorldsDir(dir)
			if err != nil {
				return nil, err
			}
			if isWorldDir {
				return newVanillaWorld(dir), nil
			}
		} else if haveWorld && haveWorldEnd && haveWorldNether {
			worldIsValid, err := isLikelyWorldsDir(filepath.Join(tmpDir, "world"))
			if err != nil {
				return nil, err
			}
			netherIsValid, err := isLikelyWorldsDir(filepath.Join(tmpDir, "world_nether"))
			if err != nil {
				return nil, err
			}
			endIsValid, err := isLikelyWorldsDir(filepath.Join(tmpDir, "world_the_end"))
			if err != nil {
				return nil, err
			}

			if !worldIsValid {
				return nil, err
			}
			if netherIsValid && endIsValid {
				return newModWorld(filepath.Join(tmpDir, "world"), filepath.Join(tmpDir, "world_nether"), filepath.Join(tmpDir, "world_the_end")), nil
			}
			return newVanillaWorld(filepath.Join(tmpDir, "world")), nil
		}
	}

	return nil, worldNotFound
}

func moveWorldDataToGameDir(tmpDir string) error {
	dirs, err := locateWorldsDir(tmpDir)
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

	slog.Info("Copying overworld data", slog.String("from", dirs.defWorld), slog.String("to", worldDir))
	cmd := exec.Command("mv", "--", dirs.defWorld, worldDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if dirs.nether != "" {
		slog.Info("Copying nether data", slog.String("from", dirs.nether), slog.String("to", netherDir))
		cmd := exec.Command("mv", "--", dirs.nether, netherDir)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		slog.Info("Copying the end data", slog.String("from", dirs.theEnd), slog.String("to", endDir))
		cmd = exec.Command("mv", "--", dirs.theEnd, endDir)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

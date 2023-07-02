package backup

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kofuk/premises/mcmanager/config"
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

func moveWorldDataToGameDir(ctx *config.PMCMContext, tmpDir string) error {
	dirs, err := locateWorldsDir(tmpDir)
	if err != nil {
		return err
	}

	worldDir := ctx.LocateWorldData("world")
	if _, err := os.Stat(worldDir); err != nil {
		if err := os.RemoveAll(worldDir); err != nil {
			return err
		}
	}
	netherDir := ctx.LocateWorldData("world_nether")
	if _, err := os.Stat(netherDir); err != nil {
		if err := os.RemoveAll(netherDir); err != nil {
			return err
		}
	}
	endDir := ctx.LocateWorldData("world_the_end")
	if _, err := os.Stat(endDir); err != nil {
		if err := os.RemoveAll(endDir); err != nil {
			return err
		}
	}

	log.Println("mv", "--", dirs.defWorld, worldDir)
	cmd := exec.Command("mv", "--", dirs.defWorld, worldDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if dirs.nether != "" {
		log.Println("mv", "--", dirs.nether, netherDir)
		cmd := exec.Command("mv", "--", dirs.nether, netherDir)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		log.Println("mv", "--", dirs.theEnd, endDir)
		cmd = exec.Command("mv", "--", dirs.theEnd, endDir)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

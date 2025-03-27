package eula

import (
	"os"
	"path/filepath"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
)

func createEulaFile(dataDir string) error {
	eulaFile, err := os.Create(filepath.Join(dataDir, "eula.txt"))
	if err != nil {
		return err
	}
	defer eulaFile.Close()
	_, err = eulaFile.WriteString("eula=true")
	return err
}

func EulaMiddleware(next core.HandlerFunc) core.HandlerFunc {
	return func(c *core.LauncherContext) error {
		if err := createEulaFile(c.Settings().GetDataDir()); err != nil {
			return err
		}

		return next(c)
	}
}

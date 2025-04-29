package eula

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
)

type EulaMiddleware struct {
}

func NewEulaMiddleware() *EulaMiddleware {
	return &EulaMiddleware{}
}

func createEulaFile(dataDir string) error {
	eulaFile, err := os.Create(filepath.Join(dataDir, "eula.txt"))
	if err != nil {
		return err
	}
	defer eulaFile.Close()
	_, err = eulaFile.WriteString("eula=true")
	return err
}

func (m *EulaMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		slog.Info("Assume EULA is accepted")
		if err := createEulaFile(c.Env().GetDataPath("gamedata")); err != nil {
			return err
		}

		return next(c)
	}
}

var _ core.Middleware = (*EulaMiddleware)(nil)

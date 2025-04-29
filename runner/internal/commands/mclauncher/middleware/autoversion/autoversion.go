package autoversion

import (
	"fmt"
	"log/slog"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/autoversion/leveldat"
)

type AutoVersionMiddleware struct {
}

var _ core.Middleware = (*AutoVersionMiddleware)(nil)

func NewAutoVersionMiddleware() *AutoVersionMiddleware {
	return &AutoVersionMiddleware{}
}

func (m *AutoVersionMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		if c.Settings().AutoVersionEnabled() {
			if version, err := leveldat.GetCanonicalServerVersion(c.Env().GetDataPath("gamedata/world/level.dat")); err != nil {
				// Don't exit here, just log the error
				slog.Error(fmt.Sprintf("failed to detect server version: %v", err))
			} else {
				c.Settings().SetMinecraftVersion(version)
			}
		}

		return next(c)
	}
}

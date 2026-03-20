package autoversion

import (
	"log/slog"

	"github.com/kofuk/premises/backend/runner/commands/mclauncher/core"
	"github.com/kofuk/premises/backend/runner/commands/mclauncher/middleware/autoversion/leveldat"
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
			slog.InfoContext(c.Context(), "Detecting server version from existing level.dat")
			if version, err := leveldat.GetCanonicalServerVersion(c.Env().GetDataPath("gamedata/world/level.dat")); err != nil {
				// Don't exit here, just log the error
				slog.ErrorContext(c.Context(), "failed to detect server version", slog.Any("error", err))
			} else {
				slog.InfoContext(c.Context(), "Detected server version", slog.String("version", version))
				c.Settings().SetMinecraftVersion(version)
			}
		}

		return next(c)
	}
}

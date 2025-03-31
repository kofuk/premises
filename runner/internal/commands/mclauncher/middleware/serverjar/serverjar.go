package serverjar

import (
	"os"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/env"
)

type ServerJarMiddleware struct {
	pathProvider env.PathProvider
}

var _ core.Middleware = (*ServerJarMiddleware)(nil)

func NewServerJarMiddleware(pathProvider env.PathProvider) *ServerJarMiddleware {
	return &ServerJarMiddleware{
		pathProvider: pathProvider,
	}
}

func (m *ServerJarMiddleware) downloadIfNotExists(c *core.LauncherContext) error {
	version := c.Settings().GetMinecraftVersion()
	serverPath := m.pathProvider.GetDataPath("servers.d", version+".jar")

	if _, err := os.Stat(serverPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// TODO: Download the server jar file from the official source.

	return nil
}

func (m *ServerJarMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c *core.LauncherContext) error {
		if err := m.downloadIfNotExists(c); err != nil {
			return err
		}
		c.Settings().SetServerPath(m.pathProvider.GetDataPath("servers.d", c.Settings().GetMinecraftVersion()+".jar"))

		return next(c)
	}
}

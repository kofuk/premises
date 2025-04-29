package serverproperties

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
)

type ServerPropertiesMiddleware struct{}

var _ core.Middleware = (*ServerPropertiesMiddleware)(nil)

func NewServerPropertiesMiddleware() *ServerPropertiesMiddleware {
	return &ServerPropertiesMiddleware{}
}

func (m *ServerPropertiesMiddleware) createServerPropertiesFile(c core.LauncherContext) error {
	serverProperties := NewServerPropertiesGenerator()
	serverProperties.SetMotd(c.Settings().GetMotd())
	serverProperties.SetDifficulty(c.Settings().GetDifficulty())
	serverProperties.SetLevelType(c.Settings().GetLevelType())
	serverProperties.SetSeed(c.Settings().GetSeed())
	for key, value := range c.Settings().ServerPropertiesOverrides() {
		if err := serverProperties.Set(key, value); err != nil {
			// We don't want to stop the server, so we log the error and continue.
			slog.Error(fmt.Sprintf("Failed to set server property: %v", err), "key", key, "value", value)
		}
	}

	outFile, err := os.Create(c.Env().GetDataPath("gamedata/server.properties"))
	if err != nil {
		return err
	}
	defer outFile.Close()
	if err := serverProperties.Write(outFile); err != nil {
		return err
	}

	return nil
}

func (m *ServerPropertiesMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		if err := m.createServerPropertiesFile(c); err != nil {
			return fmt.Errorf("failed to create server.properties file: %w", err)
		}
		return next(c)
	}
}

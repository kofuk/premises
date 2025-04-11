package world

import (
	"context"
	"errors"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world/service"
	"github.com/kofuk/premises/runner/internal/fs"
)

const (
	StateKeyWorldKey = "github.com/kofuk/premises/runner/mclauncher/middleware/world.WorldKey"
	LatestResourceID = "@/latest"
)

type WorldMiddleware struct {
	worldService *service.WorldService
}

func NewWorldMiddleware(worldService *service.WorldService) *WorldMiddleware {
	return &WorldMiddleware{
		worldService: worldService,
	}
}

var _ core.Middleware = (*WorldMiddleware)(nil)

func (m *WorldMiddleware) getRealResourceID(ctx context.Context, worldName string, resourceID string) (string, error) {
	if resourceID != LatestResourceID {
		return resourceID, nil
	}

	return m.worldService.GetLatestResourceID(ctx, worldName)
}

func (m *WorldMiddleware) cleanupOldWorldFiles(c *core.LauncherContext) error {
	return fs.RemoveIfExists(c.Env().GetDataPath("gamedata/world"))
}

func (m *WorldMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c *core.LauncherContext) error {
		worldName := c.Settings().GetWorldName()
		worldResourceID, err := m.getRealResourceID(c.Context(), worldName, c.Settings().GetWorldGeneration())
		if err != nil {
			return err
		}
		isNewWorld := c.Settings().IsNewWorld()
		oldResourceID, ok := c.State().GetState(StateKeyWorldKey).(string)

		if isNewWorld || !ok || worldResourceID != oldResourceID {
			// In this case, we don't need data of previously launched world.

			// just in case
			c.State().SetState(StateKeyWorldKey, nil)

			if err := m.cleanupOldWorldFiles(c); err != nil {
				return err
			}

			if !isNewWorld {
				// Download world if we are not creating a new world
				err := m.worldService.DownloadWorld(c.Context(), c.Settings().GetWorldGeneration(), c.Env())
				if err != nil {
					return err
				}
			}
		}

		innerError := next(c)

		if innerError == nil || errors.Is(innerError, core.ErrRestart) {
			worldKey, err := m.worldService.UploadWorld(c.Context(), worldName, c.Env())
			if err != nil {
				return errors.Join(err, innerError)
			} else {
				c.State().SetState(StateKeyWorldKey, worldKey)
			}
		}

		// Return the inner error if any
		return innerError
	}
}

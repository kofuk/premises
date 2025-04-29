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
	worldService service.WorldServiceInterface
}

func NewWorldMiddleware(worldService service.WorldServiceInterface) *WorldMiddleware {
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

func (m *WorldMiddleware) cleanupOldWorldFiles(c core.LauncherContext) error {
	return fs.RemoveIfExists(c.Env().GetDataPath("gamedata/world"))
}

func (m *WorldMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		isNewWorld := c.Settings().IsNewWorld()
		worldName := c.Settings().GetWorldName()

		var worldResourceID, oldResourceID string
		if !isNewWorld {
			var err error
			worldResourceID, err = m.getRealResourceID(c.Context(), worldName, c.Settings().GetWorldResourceID())
			if err != nil {
				return err
			}
			c.Settings().SetWorldResourceID(worldResourceID)
			oldResourceID, _ = c.State().GetState(c.Context(), StateKeyWorldKey)
		}

		// just in case
		c.State().RemoveState(c.Context(), StateKeyWorldKey)

		if isNewWorld || worldResourceID != oldResourceID {
			// In this case, we don't need data of previously launched world.

			if err := m.cleanupOldWorldFiles(c); err != nil {
				return err
			}

			if !isNewWorld {
				// Download world if we are not creating a new world
				err := m.worldService.DownloadWorld(c.Context(), worldResourceID, c.Env())
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
				c.State().SetState(c.Context(), StateKeyWorldKey, worldKey)
			}
		}

		// Return the inner error if any
		return innerError
	}
}

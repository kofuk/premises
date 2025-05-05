package watchdog

//go:generate go tool mockgen -destination watchdog_mock.go -package watchdog . Watchdog

import (
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
)

type Status struct {
	Online bool
}

type Watchdog interface {
	Name() string
	Check(c core.LauncherContext, watchID int, status *Status) error
}

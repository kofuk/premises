package watchdog

//go:generate go run go.uber.org/mock/mockgen@v0.5.2 -destination watchdog_mock.go -package watchdog . Watchdog

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

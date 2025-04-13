package watchdog

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination watchdog_mock.go -package watchdog . Watchdog

import "context"

type Status struct {
	Online bool
}

type Watchdog interface {
	Name() string
	Check(ctx context.Context, watchID int, status *Status) error
}

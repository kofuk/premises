package server

import (
	"context"
	"fmt"

	"github.com/kofuk/premises/internal/entity/runner"
)

type ServerCookie interface {
	fmt.Stringer
}

type GameServer interface {
	IsAvailable() bool
	Start(ctx context.Context, gameConfig *runner.Config, machineType string) (ServerCookie, error)
	Find(ctx context.Context) (ServerCookie, error)
	IsRunning(ctx context.Context, cookie ServerCookie) bool
	Stop(ctx context.Context, cookie ServerCookie) bool
	Delete(ctx context.Context, cookie ServerCookie) bool
}

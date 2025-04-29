package core

//go:generate go run go.uber.org/mock/mockgen@v0.5.2 -destination context_mock.go -package core . LauncherContext

import (
	"context"

	"github.com/kofuk/premises/runner/internal/env"
)

type LauncherContext interface {
	Context() context.Context
	Settings() SettingsRepository
	Env() env.EnvProvider
	State() StateRepository
}

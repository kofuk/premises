package core

import (
	"context"
	"errors"

	"github.com/kofuk/premises/runner/internal/system"
)

var ErrRestart = errors.New("restart required")

type LauncherContext struct {
	ctx      context.Context
	settings SettingsRepository
}

func (c *LauncherContext) Context() context.Context {
	return c.ctx
}

func (c *LauncherContext) Settings() SettingsRepository {
	return c.settings
}

type HandlerFunc func(c *LauncherContext) error

type Middleware interface {
	Wrap(next HandlerFunc) HandlerFunc
}

type stopMiddleware struct{}

func (m *stopMiddleware) Wrap(next HandlerFunc) HandlerFunc {
	return func(c *LauncherContext) error {
		return nil
	}
}

var StopMiddleware = &stopMiddleware{}

type LauncherCore struct {
	handler         HandlerFunc
	settings        SettingsRepository
	CommandExecutor system.CommandExecutor
}

func New(settings SettingsRepository) *LauncherCore {
	launcher := &LauncherCore{
		settings: settings,
	}

	launcher.handler = launcher.startMinecraft

	return launcher
}

func (l *LauncherCore) Middleware(m Middleware) {
	l.handler = m.Wrap(l.handler)
}

func (l *LauncherCore) createContext(ctx context.Context) *LauncherContext {
	return &LauncherContext{
		ctx:      ctx,
		settings: l.settings,
	}
}

func (l *LauncherCore) Start(ctx context.Context) error {
	return l.handler(l.createContext(ctx))
}

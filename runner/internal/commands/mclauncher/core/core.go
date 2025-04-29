package core

import (
	"context"
	"errors"

	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/system"
)

var ErrRestart = errors.New("restart required")

type launcherContext struct {
	ctx      context.Context
	settings SettingsRepository
	env      env.EnvProvider
	state    StateRepository
}

func (c *launcherContext) Context() context.Context {
	return c.ctx
}

func (c *launcherContext) Settings() SettingsRepository {
	return c.settings
}

func (c *launcherContext) Env() env.EnvProvider {
	return c.env
}

func (c *launcherContext) State() StateRepository {
	return c.state
}

type HandlerFunc func(c LauncherContext) error

type Middleware interface {
	Wrap(next HandlerFunc) HandlerFunc
}

type stopMiddleware struct{}

func (m *stopMiddleware) Wrap(next HandlerFunc) HandlerFunc {
	return func(c LauncherContext) error {
		return nil
	}
}

var StopMiddleware Middleware = &stopMiddleware{}

type LauncherCore struct {
	handler               HandlerFunc
	settings              SettingsRepository
	env                   env.EnvProvider
	CommandExecutor       system.CommandExecutor
	state                 StateRepository
	beforeLaunchListeners []BeforeLaunchListener
}

func NewLauncherCore(settings SettingsRepository, env env.EnvProvider, state StateRepository) *LauncherCore {
	launcher := &LauncherCore{
		settings:        settings,
		env:             env,
		state:           state,
		CommandExecutor: system.DefaultExecutor,
	}

	launcher.handler = launcher.startMinecraft

	return launcher
}

func (l *LauncherCore) Use(m Middleware) {
	l.handler = m.Wrap(l.handler)
}

type BeforeLaunchListener func(c LauncherContext) error

func (l *LauncherCore) AddBeforeLaunchListener(listener BeforeLaunchListener) {
	l.beforeLaunchListeners = append(l.beforeLaunchListeners, listener)
}

func (l *LauncherCore) createContext(ctx context.Context) LauncherContext {
	return &launcherContext{
		ctx:      ctx,
		settings: l.settings,
		env:      l.env,
		state:    l.state,
	}
}

func (l *LauncherCore) Start(ctx context.Context) error {
	return l.handler(l.createContext(ctx))
}

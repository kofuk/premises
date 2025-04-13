package core

import (
	"context"
	"errors"

	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/system"
)

var ErrRestart = errors.New("restart required")

type LauncherContext struct {
	ctx      context.Context
	settings SettingsRepository
	env      env.EnvProvider
	state    StateRepository
}

func (c *LauncherContext) Context() context.Context {
	return c.ctx
}

func (c *LauncherContext) Settings() SettingsRepository {
	return c.settings
}

func (c *LauncherContext) Env() env.EnvProvider {
	return c.env
}

func (c *LauncherContext) State() StateRepository {
	return c.state
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

var StopMiddleware Middleware = &stopMiddleware{}

type LauncherCore struct {
	handler               HandlerFunc
	settings              SettingsRepository
	env                   env.EnvProvider
	CommandExecutor       system.CommandExecutor
	state                 StateRepository
	BeforeLaunchListeners []BeforeLaunchListener
}

func NewLauncherCore(settings SettingsRepository, env env.EnvProvider, state StateRepository) *LauncherCore {
	launcher := &LauncherCore{
		settings: settings,
		env:      env,
		state:    state,
	}

	launcher.handler = launcher.startMinecraft

	return launcher
}

func (l *LauncherCore) Middleware(m Middleware) {
	l.handler = m.Wrap(l.handler)
}

type BeforeLaunchListener func(c *LauncherContext)

func (l *LauncherCore) AddBeforeLaunchListener(listener BeforeLaunchListener) {
	l.BeforeLaunchListeners = append(l.BeforeLaunchListeners, listener)
}

func (l *LauncherCore) createContext(ctx context.Context) *LauncherContext {
	return &LauncherContext{
		ctx:      ctx,
		settings: l.settings,
		env:      l.env,
		state:    l.state,
	}
}

func (l *LauncherCore) Start(ctx context.Context) error {
	return l.handler(l.createContext(ctx))
}

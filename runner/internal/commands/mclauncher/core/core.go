package core

import (
	"context"
	"errors"
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

type HandlerFunc func(c LauncherContext) error

type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type LauncherCore struct {
	handler HandlerFunc
}

func New(settings SettingsRepository) *LauncherCore {
	return &LauncherCore{
		handler: startMinecraft,
	}
}

func (l *LauncherCore) Middlware(m MiddlewareFunc) {
	l.handler = m(l.handler)
}

func (l *LauncherCore) Start(c LauncherContext) error {
	return l.handler(c)
}

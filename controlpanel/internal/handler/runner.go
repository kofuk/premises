package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/monitor"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/labstack/echo/v4"
)

func (h *Handler) handleRunnerPollAction(c echo.Context) error {
	runnerId, ok := c.Get("runner-id").(string)
	if !ok || runnerId == "" {
		slog.Error("Server ID is not set")
		return c.String(http.StatusInternalServerError, "")
	}

	action, err := h.runnerAction.Wait(c.Request().Context(), runnerId)
	if err != nil {
		if err == longpoll.ErrCancelled {
			return nil
		}
		slog.Error("Error waiting action", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "")
	}

	return c.JSONBlob(http.StatusOK, []byte(action))
}

func (h *Handler) handlePushStatus(c echo.Context) error {
	runnerId, ok := c.Get("runner-id").(string)
	if !ok || runnerId == "" {
		slog.Error("Runner ID is not set")
		return c.String(http.StatusInternalServerError, "")
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		slog.Error("Error reading status", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "")
	}

	events := bytes.Split(body, []byte{0})

	for _, eventData := range events {
		if len(eventData) == 0 {
			continue
		}

		var event runner.Event
		if err := json.Unmarshal(eventData, &event); err != nil {
			slog.Error("Unable to unmarshal status data", slog.Any("error", err))
			return c.String(http.StatusBadRequest, "")
		}

		if event.Type == runner.EventStatus && event.Status.EventCode == entity.EventShutdown {
			go h.shutdownServer(context.Background(), h.GameServer, c.Request().Header.Get("Authorization"))

			url, _ := url.Parse(h.cfg.ProxyAPIEndpoint)
			url.Path = "/clear"
			q := url.Query()
			q.Add("name", h.cfg.GameDomain)
			url.RawQuery = q.Encode()

			resp, err := http.Post(url.String(), "text/plain", nil)
			if err != nil {
				slog.Error("Error updating proxy", slog.Any("error", err))
			} else {
				io.Copy(io.Discard, resp.Body)
			}

			return c.String(http.StatusOK, "")
		}

		slog.Debug("Event from runner", slog.Any("payload", event))

		if err := monitor.HandleEvent(context.Background(), runnerId, h.Streaming, h.cfg, &h.KVS, &event); err != nil {
			slog.Error("Unable to handle event", slog.Any("error", err))
			return c.String(http.StatusInternalServerError, "")
		}
	}

	return nil
}

func (h *Handler) handleGetInstallScript(c echo.Context) error {
	var protocol string
	if c.QueryParam("s") == "0" {
		protocol = "http"
	} else {
		protocol = "https"
	}

	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
function run() {

if [ "$(whoami)" != root ]; then
    echo 'This script must be run as root.' >&2
    exit 1
fi

# Explicitly opening /dev/tty because stdin will be curl.
exec 3</dev/tty

read -sp 'Enter auth code: ' -u3 auth
echo
exec 3<&-

echo 'Launching Premises...'
curl -H "Authorization: Setup-Code ${auth}" '%s://%s/_runner/startup' | bash

echo 'Success! Premises should be started shortly!'

exit
} && run
`, protocol, c.Request().Host)

	return c.String(http.StatusOK, script)
}

func (h *Handler) handleGetStartupScript(c echo.Context) error {
	authKey := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(authKey, "Setup-Code ") {
		c.Response().Status = http.StatusBadRequest
		return nil
	}
	authKey = strings.TrimPrefix(authKey, "Setup-Code ")

	var script string
	if err := h.KVS.Get(c.Request().Context(), fmt.Sprintf("startup:%s", authKey), &script); err != nil {
		slog.Error("Invalid auth code", slog.Any("error", err))
		c.Response().Status = http.StatusBadRequest
		return nil
	}

	return c.String(http.StatusOK, script)
}

func (h *Handler) authKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authKey := c.Request().Header.Get("Authorization")

		var runnerId string
		if err := h.KVS.Get(c.Request().Context(), fmt.Sprintf("runner:%s", authKey), &runnerId); err != nil {
			slog.Error("Invalid auth key", slog.Any("error", err))
			return c.String(http.StatusBadRequest, "")
		}

		c.Set("runner-id", runnerId)

		return next(c)
	}
}

func (h *Handler) setupRunnerRoutes(group *echo.Group) {
	group.GET("/install", h.handleGetInstallScript)
	group.GET("/startup", h.handleGetStartupScript)

	privates := group.Group("", h.authKeyMiddleware)
	privates.GET("/poll-action", h.handleRunnerPollAction)
	privates.POST("/push-status", h.handlePushStatus)
}

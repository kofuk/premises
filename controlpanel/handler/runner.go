package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/controlpanel/monitor"
	"github.com/kofuk/premises/controlpanel/pollable"
)

func (h *Handler) handleRunnerPollAction(c *gin.Context) {
	runnerId := c.GetString("runner-id")
	if runnerId == "" {
		slog.Error("Server ID is not set")
		c.Status(http.StatusInternalServerError)
		return
	}

	action, err := h.runnerAction.Wait(c.Request.Context(), runnerId)
	if err != nil {
		if err == pollable.Cancelled {
			return
		}
		slog.Error("Error waiting action", slog.Any("error", err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "application/json")
	c.Writer.Write([]byte(action))
}

func (h *Handler) handlePushStatus(c *gin.Context) {
	runnerId := c.GetString("runner-id")
	if runnerId == "" {
		slog.Error("Runner ID is not set")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("Error reading status", slog.Any("error", err))
		c.Status(http.StatusInternalServerError)
		return
	}

	events := bytes.Split(body, []byte{0})

	slog.Debug("Event from runner", slog.Int("estimated_count", len(events)-1))

	for _, eventData := range events {
		if len(eventData) == 0 {
			continue
		}

		var event runner.Event
		if err := json.Unmarshal(eventData, &event); err != nil {
			slog.Error("Unable to unmarshal status data", slog.Any("error", err))
			c.Status(http.StatusBadRequest)
			return
		}

		if event.Type == runner.EventStatus && event.Status.EventCode == runner.EventShutdown {
			go h.shutdownServer(h.GameServer, c.GetHeader("Authorization"))
			return
		}

		if err := monitor.HandleEvent(runnerId, h.Streaming, h.cfg, &h.Cacher, h.dnsService, &event); err != nil {
			slog.Error("Unable to handle event", slog.Any("error", err))
			c.Status(http.StatusInternalServerError)
			return
		}
	}
}

func (h *Handler) handleGetInstallScript(c *gin.Context) {
	var protocol string
	if c.Query("s") == "0" {
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

name="/premises-userdata-${RANDOM}"

curl -H "Authorization: Setup-Code ${auth}" '%s://%s/_runner/startup' | base64 -d >"${name}"
chmod +x "${name}"

echo 'Launching Premises...'
bash "${name}"

rm -f "${name}"

echo 'Success! Premises should be started shortly!'


exit
} && run
`, protocol, c.Request.Host)

	c.Writer.WriteString(script)
}

func (h *Handler) handleGetStartupScript(c *gin.Context) {
	authKey := c.GetHeader("Authorization")
	if !strings.HasPrefix(authKey, "Setup-Code ") {
		c.Status(http.StatusBadRequest)
		return
	}
	authKey = strings.TrimPrefix(authKey, "Setup-Code ")

	var script string
	if err := h.Cacher.Get(c.Request.Context(), fmt.Sprintf("startup:%s", authKey), &script); err != nil {
		slog.Error("Invalid auth code", slog.Any("error", err))
		c.Status(http.StatusBadRequest)
		return
	}

	c.Writer.WriteString(script)
}

func (h *Handler) authKeyMiddleware(c *gin.Context) {
	authKey := c.GetHeader("Authorization")

	var runnerId string
	if err := h.Cacher.Get(c.Request.Context(), fmt.Sprintf("runner:%s", authKey), &runnerId); err != nil {
		slog.Error("Invalid auth key", slog.Any("error", err))
		c.Status(http.StatusBadRequest)
		c.Abort()
		return
	}

	c.Set("runner-id", runnerId)
}

func (h *Handler) setupRunnerRoutes(group *gin.RouterGroup) {
	group.GET("/install", h.handleGetInstallScript)
	group.GET("/startup", h.handleGetStartupScript)

	privates := group.Group("", h.authKeyMiddleware)
	privates.GET("/poll-action", h.handleRunnerPollAction)
	privates.POST("/push-status", h.handlePushStatus)
}

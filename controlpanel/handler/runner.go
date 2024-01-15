package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/controlpanel/dns"
	"github.com/kofuk/premises/controlpanel/monitor"
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

	var event runner.Event
	if err := json.Unmarshal(body, &event); err != nil {
		slog.Error("Unable to unmarshal status data", slog.Any("error", err))
		c.Status(http.StatusBadRequest)
		return
	}

	if event.Type == runner.EventStatus && event.Status.EventCode == runner.EventShutdown {
		var dnsProvider *dns.DNSProvider
		if h.cfg.Cloudflare.Token != "" {
			cloudflareDNS, err := dns.NewCloudflareDNS(h.cfg.Cloudflare.Token, h.cfg.Cloudflare.ZoneID)
			if err != nil {
				slog.Error("Failed to initialize DNS provider", slog.Any("error", err))
			} else {
				dnsProvider = dns.New(cloudflareDNS, h.cfg.Cloudflare.GameDomainName)
			}
		}
		go h.shutdownServer(h.serverImpl, h.redis, dnsProvider, c.GetHeader("Authorization"))

		return
	}

	if err := monitor.HandleEvent(runnerId, h.Streaming, h.cfg, &h.Cacher, &event); err != nil {
		slog.Error("Unable to handle event", slog.Any("error", err))
		c.Status(http.StatusInternalServerError)
		return
	}
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
	group.Use(h.authKeyMiddleware)
	group.GET("/poll-action", h.handleRunnerPollAction)
	group.POST("/push-status", h.handlePushStatus)
}

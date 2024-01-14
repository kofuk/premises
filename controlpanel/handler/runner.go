package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *Handler) authKeyMiddleware(c *gin.Context) {
	authKey := c.GetHeader("Authorization")

	var runnerId string
	if err := h.Cacher.Get(c.Request.Context(), fmt.Sprintf("runner:%s", authKey), &runnerId); err != nil {
		c.Status(http.StatusBadRequest)
		c.Abort()
		return
	}

	c.Set("runner-id", runnerId)
}

func (h *Handler) setupRunnerRoutes(group *gin.RouterGroup) {
	group.Use(h.authKeyMiddleware)
	group.GET("/poll-action", h.handleRunnerPollAction)
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/monitor"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/entity/web"
	potel "github.com/kofuk/premises/internal/otel"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (h *Handler) handleRunnerPoll(c echo.Context) error {
	runnerId, ok := c.Get("runner-id").(string)
	if !ok || runnerId == "" {
		slog.Error("Server ID is not set")
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	action, err := h.runnerAction.Wait(c.Request().Context(), runnerId)
	if err != nil {
		if err == longpoll.ErrCancelled {
			return nil
		}
		slog.Error("Error waiting action", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[json.RawMessage]{
		Success: true,
		Data:    json.RawMessage(action),
	})
}

func (h *Handler) handlePostStatus(c echo.Context) error {
	runnerId, ok := c.Get("runner-id").(string)
	if !ok || runnerId == "" {
		slog.Error("Runner ID is not set")
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		slog.Error("Error reading status", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	events := bytes.Split(body, []byte{0})

	for _, eventData := range events {
		if len(eventData) == 0 {
			continue
		}

		var event runner.Event
		if err := json.Unmarshal(eventData, &event); err != nil {
			slog.Error("Unable to unmarshal status data", slog.Any("error", err))
			return c.JSON(http.StatusOK, web.SuccessfulResponse[interface{}]{
				Success: true,
				Data:    nil,
			})
		}

		span := potel.NoopSpan
		ctx := context.Background()
		if event.Type != runner.EventSysstat {
			tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer(ScopeName)
			ctx, span = tracer.Start(
				potel.ContextFromTraceContext(ctx, event.Metadata.Traceparent),
				fmt.Sprintf("Event (%s)", event.Type),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(attribute.String("premises.event_type", event.Type.String())),
			)
		}

		if event.Type == runner.EventStatus && event.Status.EventCode == entity.EventShutdown {
			go h.shutdownServer(ctx, c.Request().Header.Get("Authorization"))

			span.End()

			return c.JSON(http.StatusOK, web.SuccessfulResponse[interface{}]{
				Success: true,
				Data:    nil,
			})
		}

		slog.Debug("Event from runner", slog.Any("payload", event))

		if err := monitor.HandleEvent(ctx, runnerId, h.Streaming, h.cfg, &h.KVS, &event); err != nil {
			slog.Error("Unable to handle event", slog.Any("error", err))

			span.SetStatus(codes.Error, err.Error())
			span.End()

			return c.JSON(http.StatusOK, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrInternal,
			})
		}

		span.End()
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[interface{}]{
		Success: true,
		Data:    nil,
	})
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
curl -H "Authorization: Setup-Code ${auth}" '%s://%s/_/startup' | bash

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

func (h *Handler) handleGetLatestWorldID(c echo.Context) error {
	worldName := c.Param("worldName")
	if worldName == "" {
		slog.Error("World name is not set")
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	key, err := h.world.GetLatestWorldKey(c.Request().Context(), worldName)
	if err != nil {
		slog.Error("Unable to get latest world key", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.GetLatestWorldIDResponse]{
		Success: true,
		Data:    web.GetLatestWorldIDResponse{WorldID: key},
	})
}

func (h *Handler) handleCreateWorldDownloadURL(c echo.Context) error {
	var req web.CreateWorldDownloadURLRequest
	if err := c.Bind(&req); err != nil {
		slog.Error("Unable to bind request", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	url, err := h.world.GetPresignedGetURL(c.Request().Context(), req.WorldID)
	if err != nil {
		slog.Error("Unable to get presigned URL", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.CreateWorldDownloadURLResponse]{
		Success: true,
		Data:    web.CreateWorldDownloadURLResponse{URL: url},
	})
}

func (h *Handler) handleCreateWorldUploadURL(c echo.Context) error {
	var req web.CreateWorldUploadURLRequest
	if err := c.Bind(&req); err != nil {
		slog.Error("Unable to bind request", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if strings.ContainsRune(req.WorldName, '/') {
		slog.Error("Invalid world name", slog.Any("worldName", req.WorldName))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	key := fmt.Sprintf("%s/%s.tar.zst", req.WorldName, time.Now().Format(time.DateTime))

	url, err := h.world.GetPresignedPutURL(c.Request().Context(), key)
	if err != nil {
		slog.Error("Unable to get presigned URL", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.CreateWorldUploadURLResponse]{
		Success: true,
		Data: web.CreateWorldUploadURLResponse{
			URL:     url,
			WorldID: key,
		},
	})
}

func (h *Handler) authKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authKey := c.Request().Header.Get("Authorization")

		var runnerId string
		if err := h.KVS.Get(c.Request().Context(), fmt.Sprintf("runner:%s", authKey), &runnerId); err != nil {
			slog.Error("Invalid auth key", slog.Any("error", err))
			return c.JSON(http.StatusOK, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrBadRequest,
			})
		}

		c.Set("runner-id", runnerId)

		return next(c)
	}
}

func (h *Handler) setupRunnerRoutes(group *echo.Group) {
	group.GET("/install", h.handleGetInstallScript)
	group.GET("/startup", h.handleGetStartupScript)

	privates := group.Group("", h.authKeyMiddleware)
	privates.GET("/poll", h.handleRunnerPoll)
	privates.POST("/status", h.handlePostStatus)
	privates.GET("/world/latest-id/:worldName", h.handleGetLatestWorldID)
	privates.POST("/world/download-url", h.handleCreateWorldDownloadURL)
	privates.POST("/world/upload-url", h.handleCreateWorldUploadURL)
}

package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/conoha"
	"github.com/kofuk/premises/controlpanel/gameconfig"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/kofuk/premises/controlpanel/monitor"
	"github.com/kofuk/premises/controlpanel/streaming"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	CacheKeyBackups          = "backups"
	CacheKeyMCVersions       = "mcversions"
	CacheKeySystemInfoPrefix = "system-info"
)

func (h *Handler) handleApiSessionData(c echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	userID, ok := session.Values["user_id"].(uint)

	sessionData := web.SessionData{
		LoggedIn: ok,
	}

	if ok {
		var userName string
		if err := h.db.NewSelect().Model((*model.User)(nil)).Column("name").Where("id = ? AND deleted_at IS NULL", userID).Scan(c.Request().Context(), &userName); err != nil {
			slog.Error("User not found", slog.Any("error", err))
			return c.JSON(http.StatusOK, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrInternal,
			})
		}
		sessionData.UserName = userName
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.SessionData]{
		Success: true,
		Data:    sessionData,
	})
}

func (h *Handler) createStreamingEndpoint(stream *streaming.Stream, eventName string) func(c echo.Context) error {
	return func(c echo.Context) error {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		jsonRequested := c.Request().Header.Get("Accept") == "application/ld+json"

		writeEvent := func(message []byte) error {
			writer := bufio.NewWriter(c.Response().Writer)

			if jsonRequested {
				writer.Write(message)
				writer.WriteRune('\n')
			} else {
				writer.WriteString("event: " + eventName + "\n")
				writer.WriteString("data: ")
				writer.Write(message)
				writer.WriteString("\n\n")
			}
			writer.Flush()

			if flusher, ok := c.Response().Writer.(http.Flusher); ok {
				flusher.Flush()
			}
			return nil
		}

		subscription, statusHistory, err := h.Streaming.SubscribeEvent(c.Request().Context(), stream)
		if err != nil {
			slog.Error("Failed to connect to stream", slog.Any("error", err))
			return c.String(http.StatusInternalServerError, "")
		}
		defer subscription.Close()

		if !jsonRequested {
			c.Response().Header().Set("Content-Type", "text/event-stream")
			c.Response().Header().Set("X-Accel-Buffering", "no")
		}
		c.Response().Header().Set("Cache-Control", "no-store")

		for _, entry := range statusHistory {
			if err := writeEvent(entry); err != nil {
				slog.Error("Failed to write data", slog.Any("error", err))
				return err
			}
		}

		if jsonRequested {
			return nil
		}

		eventChannel := subscription.Channel()

	out:
		for {
			select {
			case status := <-eventChannel:
				if err := writeEvent([]byte(status.Payload)); err != nil {
					slog.Error("Failed to write server-sent event", slog.Any("error", err))
					break out
				}

			case <-ticker.C:
				if _, err := c.Response().Writer.Write([]byte(": uhaha\n")); err != nil {
					slog.Error("Failed to write keep-alive message", slog.Any("error", err))
					break out
				}
				if flusher, ok := c.Response().Writer.(http.Flusher); ok {
					flusher.Flush()
				}

			case <-c.Request().Context().Done():
				break out
			}
		}

		return nil
	}
}

func isValidMemSize(memSize int) bool {
	return memSize == 1 || memSize == 2 || memSize == 4 || memSize == 8 || memSize == 16 || memSize == 32 || memSize == 64
}

func (h *Handler) createConfigFromPostData(ctx context.Context, values url.Values, cfg *config.Config) (*runner.Config, error) {
	if !values.Has("server-version") {
		return nil, errors.New("Server version is not set")
	}
	result := gameconfig.New()

	result.C.ControlPanel = h.cfg.ControlPanel.Origin
	if strings.HasPrefix(h.cfg.ControlPanel.Origin, "http://localhost:") {
		result.C.ControlPanel = strings.Replace(h.cfg.ControlPanel.Origin, "http://localhost", "http://host.docker.internal", 1)
	}

	serverDownloadURL, launchCommand, err := h.MCVersions.GetServerInfo(ctx, values.Get("server-version"))
	if err != nil {
		return nil, err
	}
	result.SetServer(values.Get("server-version"), serverDownloadURL)
	result.SetDetectServerVersion(values.Get("prefer-detect") == "true")
	result.C.Server.ManifestOverride = h.MCVersions.GetOverridenManifestUrl()
	result.C.Server.CustomCommand = launchCommand

	if !values.Has("machine-type") {
		return nil, errors.New("Machine type is not set")
	}
	memSizeGB, err := strconv.Atoi(strings.Replace(values.Get("machine-type"), "g", "", 1))
	if err != nil {
		return nil, err
	}
	if !isValidMemSize(memSizeGB) {
		return nil, errors.New("Invalid machine type")
	}
	result.GenerateAuthKey()

	if values.Get("world-source") == "backups" {
		if err := result.SetWorld(values.Get("world-name"), values.Get("backup-generation")); err != nil {
			return nil, err
		}
	} else {
		if !values.Has("world-name") {
			return nil, errors.New("World name is not set")
		}
		result.GenerateWorld(values.Get("world-name"), values.Get("seed"))
		if err := result.SetLevelType(values.Get("level-type")); err != nil {
			return nil, err
		}
	}

	result.SetOperators(cfg.Game.Operators)
	result.SetWhitelist(cfg.Game.Whitelist)
	result.C.AWS.AccessKey = cfg.AWS.AccessKey
	result.C.AWS.SecretKey = cfg.AWS.SecretKey
	result.C.S3.Endpoint = cfg.S3.Endpoint
	result.C.S3.Bucket = cfg.S3.Bucket
	result.SetMotd(cfg.Game.Motd)

	return &result.C, nil
}

func (h *Handler) notifyNonRecoverableFailure(cfg *config.Config, detail string) {
	if cfg.ControlPanel.AlertWebhookUrl != "" {
		payload := struct {
			Text     string `json:"text"`
			Markdown bool   `json:"mrkdwn"`
		}{
			Text:     "Unrecoverable error occurred: " + detail,
			Markdown: false,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPost, cfg.ControlPanel.AlertWebhookUrl, bytes.NewBuffer(body))
		if err != nil {
			slog.Error("Failed to create new request", slog.Any("error", err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Failed to send request", slog.Any("error", err))
			return
		}
		d, _ := io.ReadAll(resp.Body)
		slog.Info("Webhook response", slog.String("body", string(d)))
	}
}

func (h *Handler) shutdownServer(ctx context.Context, gameServer *GameServer, authKey string) {
	defer func() {
		h.serverMutex.Lock()
		defer h.serverMutex.Unlock()
		h.serverRunning = false
	}()

	stdStream := h.Streaming.GetStream(streaming.StandardStream)
	infoStream := h.Streaming.GetStream(streaming.InfoStream)

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessage(entity.EventStopRunner, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	var id string
	if err := h.KVS.Get(ctx, "runner-id:default", &id); err != nil {
		if err := h.Streaming.PublishEvent(
			ctx,
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		); err != nil {
			slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
		}
		h.notifyNonRecoverableFailure(h.cfg, "Runner ID is not set")
		return
	}

	if !gameServer.StopVM(ctx, id) {
		if err := h.Streaming.PublishEvent(
			ctx,
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		); err != nil {
			slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
		}
		h.notifyNonRecoverableFailure(h.cfg, "Failed to stop VM")
		return
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EventStopRunner, 40, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EventStopRunner, 80, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	if !gameServer.DeleteVM(ctx, id) {
		if err := h.Streaming.PublishEvent(
			ctx,
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		); err != nil {
			slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
		}
		h.notifyNonRecoverableFailure(h.cfg, "Failed to delete VM")
		return
	}

	if h.dnsService != nil {
		h.dnsService.UpdateV4(ctx, net.ParseIP("127.0.0.1"))
	}

	if err := h.KVS.Del(ctx, "runner-id:default", "runner-info:default", "world-info:default", fmt.Sprintf("runner:%s", authKey)); err != nil {
		slog.Error("Failed to unset runner information", slog.Any("error", err))
		return
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessage(entity.EventStopped, web.PageLaunch),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	if err := h.Streaming.ClearHistory(ctx, h.Streaming.GetStream(streaming.SysstatStream)); err != nil {
		slog.Error("Unable to clear sysstat history", slog.Any("error", err))
	}
}

func (h *Handler) LaunchServer(ctx context.Context, gameConfig *runner.Config, gameServer *GameServer, memSizeGB int) {
	stdStream := h.Streaming.GetStream(streaming.StandardStream)
	infoStream := h.Streaming.GetStream(streaming.InfoStream)

	if err := h.KVS.Set(ctx, fmt.Sprintf("runner:%s", gameConfig.AuthKey), "default", -1); err != nil {
		slog.Error("Failed to save runner id", slog.Any("error", err))

		if err := h.Streaming.PublishEvent(
			ctx,
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		); err != nil {
			slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
		}

		h.serverRunning = false
		return
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessage(entity.EventCreateRunner, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EventCreateRunner, 10, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	slog.Info("Generating startup script...")
	startupScript, err := conoha.GenerateStartupScript(gameConfig)
	if err != nil {
		slog.Error("Failed to generate startup script", slog.Any("error", err))

		if err := h.Streaming.PublishEvent(
			ctx,
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		); err != nil {
			slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
		}

		h.serverRunning = false
		return
	}
	slog.Info("Generating startup script...Done")

	id := gameServer.SetUp(ctx, gameConfig, memSizeGB, startupScript)
	if id == "" {
		// Startup failed. Manual setup can recover this status.

		encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
		authCode := encoder.EncodeToString(securecookie.GenerateRandomKey(10))

		if err := h.Streaming.PublishEvent(
			ctx,
			stdStream,
			streaming.NewStandardMessageWithTextData(entity.EventManualSetup, authCode, web.PageManualSetup),
		); err != nil {
			slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
		}

		if err := h.KVS.Set(ctx, fmt.Sprintf("startup:%s", authCode), string(startupScript), 30*time.Minute); err != nil {
			slog.Error("Failed to set startup script", slog.Any("error", err))
			return
		}

		return
	}

	if err := h.KVS.Set(ctx, "runner-id:default", id, -1); err != nil {
		slog.Error("Failed to set runner ID", slog.Any("error", err))
		return
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EventCreateRunner, 50, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}

	if err := h.Streaming.PublishEvent(
		ctx,
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EventCreateRunner, 80, web.PageLoading),
	); err != nil {
		slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
	}
}

func (h *Handler) handleApiLaunch(c echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		slog.Error("Failed to parse form", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	h.serverMutex.Lock()
	defer h.serverMutex.Unlock()

	if h.serverRunning {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrServerRunning,
		})
	}

	gameConfig, err := h.createConfigFromPostData(c.Request().Context(), c.Request().Form, h.cfg)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInvalidConfig,
		})
	}

	h.serverRunning = true

	machineType := c.Request().PostFormValue("machine-type")
	h.serverState.machineType = machineType
	memSizeGB, _ := strconv.Atoi(strings.Replace(machineType, "g", "", 1))

	go h.LaunchServer(context.Background(), gameConfig, h.GameServer, memSizeGB)

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiReconfigure(c echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		slog.Error("Failed to parse form", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	formValues := c.Request().Form
	formValues.Set("machine-type", h.serverState.machineType)

	gameConfig, err := h.createConfigFromPostData(c.Request().Context(), formValues, h.cfg)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInvalidConfig,
		})
	}

	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type:   runner.ActionReconfigure,
		Config: gameConfig,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiStop(c echo.Context) error {
	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionStop,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiBackups(c echo.Context) error {
	if val, err := h.redis.Get(c.Request().Context(), CacheKeyBackups).Result(); err == nil {
		return c.JSONBlob(http.StatusOK, []byte(val))
	} else if err != redis.Nil {
		slog.Error("Error retrieving backups from cache", slog.Any("error", err))
	}

	slog.Info("cache miss", slog.String("cache_key", CacheKeyBackups))

	backups, err := h.backup.GetWorlds(c.Request().Context())
	if err != nil {
		slog.Error("Failed to retrieve backup list", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBackup,
		})
	}

	resp := web.SuccessfulResponse[[]web.WorldBackup]{
		Success: true,
		Data:    backups,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		slog.Error("Failed to marshal backpu list", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if _, err := h.redis.Set(c.Request().Context(), CacheKeyBackups, jsonResp, 3*time.Minute).Result(); err != nil {
		slog.Error("Failed to store backup list", slog.Any("error", err))
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) handleApiMcversions(c echo.Context) error {
	versions, err := h.MCVersions.GetVersions(c.Request().Context())
	if err != nil {
		slog.Error("Failed to retrieve Minecraft versions", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	versionsEntity := make([]web.MCVersion, 0)
	for _, ver := range versions {
		channel := ""
		if ver.Type == "release" {
			channel = "stable"
		} else if ver.Type == "snapshot" {
			channel = "snapshot"
		} else if ver.Type == "old_beta" {
			channel = "beta"
		} else if ver.Type == "old_alpha" {
			channel = "alpha"
		} else {
			channel = "unknown"
		}

		versionsEntity = append(versionsEntity, web.MCVersion{
			Name:        ver.ID,
			IsStable:    ver.Type == "release",
			Channel:     channel,
			ReleaseDate: ver.ReleaseTime,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[[]web.MCVersion]{
		Success: true,
		Data:    versionsEntity,
	})
}

func (h *Handler) handleApiSystemInfo(c echo.Context) error {
	data, err := monitor.GetSystemInfo(c.Request().Context(), h.cfg, &h.KVS)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.SystemInfo]{
		Success: true,
		Data:    *data,
	})
}

func (h *Handler) handleApiWorldInfo(c echo.Context) error {
	data, err := monitor.GetWorldInfo(c.Request().Context(), h.cfg, &h.KVS)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.WorldInfo]{
		Success: true,
		Data:    *data,
	})
}

func (h *Handler) handleApiQuickUndoSnapshot(c echo.Context) error {
	var config web.SnapshotConfiguration
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if config.Slot < 0 || 10 <= config.Slot {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionSnapshot,
		Snapshot: runner.SnapshotConfig{
			Slot: config.Slot,
		},
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiQuickUndoUndo(c echo.Context) error {
	var config web.SnapshotConfiguration
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if config.Slot < 0 || 10 <= config.Slot {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionUndo,
		Snapshot: runner.SnapshotConfig{
			Slot: config.Slot,
		},
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func setupApiQuickUndoRoutes(h *Handler, group *echo.Group) {
	group.POST("/snapshot", h.handleApiQuickUndoSnapshot)
	group.POST("/undo", h.handleApiQuickUndoUndo)
}

func (h *Handler) handleApiUsersChangePassword(c echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	var req web.UpdatePassword
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if !isAllowedPassword(req.NewPassword) {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
	}

	var password string
	if err := h.db.NewSelect().Model((*model.User)(nil)).Column("password").Where("id = ? AND deleted_at IS NULL", userID).Scan(c.Request().Context(), &password); err != nil {
		slog.Error("User not found", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if _, err := h.db.NewUpdate().Model((*model.User)(nil)).Set("password = ?", string(hashedPassword)).Set("initialized = ?", true).Where("id = ? AND deleted_at IS NULL", userID).Exec(c.Request().Context()); err != nil {
		slog.Error("error updating password", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiUsersAdd(c echo.Context) error {
	session, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	var req web.PasswordCredential
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if len(req.UserName) == 0 || len(req.UserName) > 32 {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}
	if !isAllowedPassword(req.Password) {
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	user := &model.User{
		Name:          req.UserName,
		Password:      string(hashedPassword),
		AddedByUserID: &userID,
		Initialized:   false,
	}

	if _, err := h.db.NewInsert().Model(user).Exec(c.Request().Context()); err != nil {
		slog.Error("error registering user", slog.Any("error", err))
		return c.JSON(http.StatusOK, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrDupUserName,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func setupApiUsersRoutes(h *Handler, group *echo.Group) {
	group.POST("/change-password", h.handleApiUsersChangePassword)
	group.POST("/add", h.handleApiUsersAdd)
}

func (h *Handler) middlewareSessionCheck(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 1. Verify that the request is sent from allowed origin (if needed).
		if c.Request().Method == http.MethodPost || (c.Request().Method == http.MethodGet && c.Request().Header.Get("Upgrade") == "WebSocket") {
			if c.Request().Header.Get("Origin") != h.cfg.ControlPanel.Origin {
				slog.Error("origin not allowed", slog.String("origin", c.Request().Header.Get("Origin")))
				return c.JSON(http.StatusOK, web.ErrorResponse{
					Success:   false,
					ErrorCode: entity.ErrBadRequest,
				})
			}
		}

		// 2. Verify that the client is logged in.
		session, err := session.Get("session", c)
		if err != nil {
			return c.JSON(http.StatusOK, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrInternal,
			})
		}

		if _, ok := session.Values["user_id"]; !ok {
			return c.JSON(http.StatusOK, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrRequiresAuth,
			})
		}
		return next(c)
	}
}

func (h *Handler) setupApiRoutes(group *echo.Group) {
	group.GET("/session-data", h.handleApiSessionData)
	needsAuth := group.Group("")
	needsAuth.Use(h.middlewareSessionCheck)
	needsAuth.GET("/streaming/events", h.createStreamingEndpoint(h.Streaming.GetStream(streaming.StandardStream), "statuschanged"))
	needsAuth.GET("/streaming/info", h.createStreamingEndpoint(h.Streaming.GetStream(streaming.InfoStream), "notify"))
	needsAuth.GET("/streaming/sysstat", h.createStreamingEndpoint(h.Streaming.GetStream(streaming.SysstatStream), "systemstat"))
	needsAuth.POST("/launch", h.handleApiLaunch)
	needsAuth.POST("/reconfigure", h.handleApiReconfigure)
	needsAuth.POST("/stop", h.handleApiStop)
	needsAuth.GET("/backups", h.handleApiBackups)
	needsAuth.GET("/mcversions", h.handleApiMcversions)
	needsAuth.GET("/systeminfo", h.handleApiSystemInfo)
	needsAuth.GET("/worldinfo", h.handleApiWorldInfo)
	setupApiQuickUndoRoutes(h, needsAuth.Group("/quickundo"))
	setupApiUsersRoutes(h, needsAuth.Group("/users"))
}

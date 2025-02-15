package handler

import (
	"bufio"
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/controlpanel/internal/auth"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/db/model"
	"github.com/kofuk/premises/controlpanel/internal/gameconfig"
	"github.com/kofuk/premises/controlpanel/internal/monitor"
	"github.com/kofuk/premises/controlpanel/internal/startup"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/entity/web"
	potel "github.com/kofuk/premises/internal/otel"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

const (
	CacheKeyWorlds           = "worlds"
	CacheKeyMCVersions       = "mcversions"
	CacheKeySystemInfoPrefix = "system-info"
)

func (h *Handler) handleStream(c echo.Context) error {
	jsonMode := c.Request().Header.Get("Accept") == "application/json"

	userID := c.Get("access_token").(*auth.Token).UserID

	writeEvent := func(eventName string, message []byte) error {
		if jsonMode {
			c.JSONBlob(http.StatusOK, message)
			return nil
		}

		var partial struct {
			Actor int `json:"actor"`
		}
		json.Unmarshal(message, &partial)

		if partial.Actor != 0 && partial.Actor != int(userID) {
			// Skip delivering messages that were triggered by other users.
			return nil
		}

		writer := bufio.NewWriter(c.Response().Writer)

		writer.WriteString("event: " + eventName + "\n")
		writer.WriteString("data: ")
		writer.Write(message)
		writer.WriteString("\n\n")
		writer.Flush()

		if flusher, ok := c.Response().Writer.(http.Flusher); ok {
			flusher.Flush()
		}
		return nil
	}

	subscription, err := h.Streaming.SubscribeEvent(c.Request().Context())
	if err != nil {
		slog.Error("Failed to connect to stream", slog.Any("error", err))
		return c.String(http.StatusInternalServerError, "")
	}
	defer subscription.Close()

	if !jsonMode {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("X-Accel-Buffering", "no")
	}
	c.Response().Header().Set("Cache-Control", "no-store")

	if err := writeEvent(streaming.EventMessage.String(), subscription.CurrentState); err != nil {
		slog.Error("Failed to write data", slog.Any("error", err))
		return err
	}

	if jsonMode {
		// If it is a JSON mode, we only send the current state and exit.
		return nil
	}

	for _, entry := range subscription.SysstatHistory {
		if err := writeEvent(streaming.SysstatMessage.String(), entry); err != nil {
			slog.Error("Failed to write data", slog.Any("error", err))
			return err
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	eventChannel := subscription.Channel()

out:
	for {
		select {
		case status := <-eventChannel:
			body, _ := json.Marshal(status.Body)
			if err := writeEvent(status.Type.String(), body); err != nil {
				slog.Error("Failed to write server-sent event", slog.Any("error", err))
				break out
			}

		case <-c.Request().Context().Done():
			break out
		}
	}

	return nil
}

func isValidMemSize(memSize int) bool {
	return memSize == 1 || memSize == 2 || memSize == 4 || memSize == 8 || memSize == 16 || memSize == 32 || memSize == 64
}

func (h *Handler) createConfigFromPostData(ctx context.Context, config web.PendingConfig, cfg *config.Config) (*runner.GameConfig, error) {
	if config.ServerVersion == nil || *config.ServerVersion == "" {
		return nil, errors.New("server version is not set")
	}
	result := gameconfig.New()

	serverInfo, err := h.MCVersions.GetServerInfo(ctx, *config.ServerVersion)
	if err != nil {
		return nil, err
	}
	result.SetServer(*config.ServerVersion, serverInfo.DownloadURL)
	result.SetDetectServerVersion(*config.GuessVersion)
	result.C.Server.ManifestOverride = h.MCVersions.GetOverridenManifestUrl()
	result.C.Server.CustomCommand = serverInfo.LaunchCommand
	result.C.Server.JavaVersion = serverInfo.JavaVersion
	if config.InactiveTimeout != nil {
		result.C.Server.InactiveTimeout = *config.InactiveTimeout
	} else {
		result.C.Server.InactiveTimeout = -1
	}

	if config.WorldSource != nil && *config.WorldSource == "backups" {
		if config.WorldName == nil || config.BackupGen == nil {
			return nil, errors.New("both worldName and backupGen must be set if worldSource is backups")
		}

		if err := result.SetWorld(*config.WorldName, *config.BackupGen); err != nil {
			return nil, err
		}
	} else {
		if config.WorldName == nil || *config.WorldName == "" {
			return nil, errors.New("world name is not set")
		}
		seed := ""
		if config.Seed != nil {
			seed = *config.Seed
		}
		levelType := "default"
		if config.LevelType != nil {
			levelType = *config.LevelType
		}
		result.GenerateWorld(*config.WorldName, seed)
		if err := result.SetLevelType(levelType); err != nil {
			return nil, err
		}
	}
	if config.ServerPropOverride != nil {
		result.C.Server.ServerPropOverride = *config.ServerPropOverride
	}
	if config.Motd != nil {
		result.SetMotd(*config.Motd)
	}

	result.SetOperators(cfg.Operators)
	result.SetWhitelist(cfg.Whitelist)

	return &result.C, nil
}

func (h *Handler) shutdownServer(ctx context.Context, gameServer *GameServer, authKey string) {
	defer h.releaseServerLock(context.TODO())

	h.Streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessage(entity.EventStopRunner, web.PageLoading),
	)

	var id string
	if err := h.KVS.Get(ctx, "runner-id:default", &id); err != nil || !gameServer.IsAvailable() {
		if err == redis.Nil {
			goto out
		}

		h.Streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		)
		return
	}

	if !gameServer.StopVM(ctx, id) {
		h.Streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		)
		return
	}

	h.Streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessageWithProgress(entity.EventStopRunner, 40, web.PageLoading),
	)

	h.Streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessageWithProgress(entity.EventStopRunner, 80, web.PageLoading),
	)

	if !gameServer.DeleteVM(ctx, id) {
		h.Streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		)
		return
	}

out:
	if err := h.world.Prune(ctx, 3); err != nil {
		slog.Error("Failed to prune worlds", slog.Any("error", err))
	}

	if err := h.KVS.Del(ctx, "runner-id:default", "runner-info:default", "world-info:default", fmt.Sprintf("runner:%s", authKey)); err != nil {
		slog.Error("Failed to unset runner information", slog.Any("error", err))
		return
	}

	h.Streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessage(entity.EventStopped, web.PageLaunch),
	)

	if err := h.Streaming.ClearSysstat(ctx); err != nil {
		slog.Error("Unable to clear sysstat history", slog.Any("error", err))
	}
}

func (h *Handler) LaunchServer(ctx context.Context, serverConfig *runner.Config, gameServer *GameServer, memSizeGB int) {
	if err := h.KVS.Set(ctx, fmt.Sprintf("runner:%s", serverConfig.AuthKey), "default", -1); err != nil {
		slog.Error("Failed to save runner id", slog.Any("error", err))

		h.Streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		)

		h.releaseServerLock(context.TODO())
		return
	}

	h.Streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessage(entity.EventCreateRunner, web.PageLoading),
	)

	slog.Info("Generating startup script...")
	startupScript, err := startup.GenerateStartupScript(serverConfig)
	if err != nil {
		slog.Error("Failed to generate startup script", slog.Any("error", err))

		h.Streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		)

		h.releaseServerLock(context.TODO())
		return
	}
	slog.Info("Generating startup script...Done")

	if gameServer.IsAvailable() {
		if id := gameServer.SetUp(ctx, serverConfig, memSizeGB, startupScript); id != "" {
			if err := h.KVS.Set(ctx, "runner-id:default", id, -1); err != nil {
				slog.Error("Failed to set runner ID", slog.Any("error", err))
				return
			}

			h.Streaming.PublishEvent(
				ctx,
				streaming.NewStandardMessageWithProgress(entity.EventCreateRunner, 50, web.PageLoading),
			)

			return
		}
	}

	// Startup failed. Manual setup required.

	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	authCode := encoder.EncodeToString(securecookie.GenerateRandomKey(10))

	h.Streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessageWithTextData(entity.EventManualSetup, authCode, web.PageManualSetup),
	)

	if err := h.KVS.Set(ctx, fmt.Sprintf("startup:%s", authCode), string(startupScript), time.Hour); err != nil {
		slog.Error("Failed to set startup script", slog.Any("error", err))
		return
	}
}

func (h *Handler) aquireServerLock(ctx context.Context) (bool, error) {
	var running bool
	if err := h.KVS.GetSet(ctx, "running", true, -1, &running); err != nil {
		if errors.Is(err, redis.Nil) {
			return true, nil
		}
		return false, err
	}
	return !running, nil
}

func (h *Handler) releaseServerLock(ctx context.Context) error {
	return h.KVS.Del(ctx, "running")
}

func generateAuthKey() string {
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	result := encoder.EncodeToString(securecookie.GenerateRandomKey(30))
	return result
}

func (h *Handler) handleApiLaunch(c echo.Context) error {
	var config web.PendingConfig
	if err := h.KVS.Get(c.Request().Context(), "pending-config", &config); err != nil {
		slog.Error("Failed to get pending config", slog.Any("error", err))
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	gameConfig, err := h.createConfigFromPostData(c.Request().Context(), config, h.cfg)
	if err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInvalidConfig,
		})
	}

	if config.MachineType == nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}
	memSizeGB, err := strconv.Atoi(strings.Replace(*config.MachineType, "g", "", 1))
	if err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}
	if !isValidMemSize(memSizeGB) {
		slog.Error("Invalid mem size")
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	canLaunch, err := h.aquireServerLock(c.Request().Context())
	if err != nil {
		slog.Error("Failed to aquire server lock", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if !canLaunch {
		return c.JSON(http.StatusConflict, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrServerRunning,
		})
	}

	serverConfig := &runner.Config{
		AuthKey:      generateAuthKey(),
		ControlPanel: h.cfg.Origin,
		GameConfig:   *gameConfig,
	}

	go h.LaunchServer(
		trace.ContextWithSpan(context.Background(), trace.SpanFromContext(c.Request().Context())),
		serverConfig,
		h.GameServer,
		memSizeGB,
	)

	return c.JSON(http.StatusAccepted, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiReconfigure(c echo.Context) error {
	var config web.PendingConfig
	if err := h.KVS.Get(c.Request().Context(), "pending-config", &config); err != nil {
		slog.Error("Failed to get pending config", slog.Any("error", err))
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	gameConfig, err := h.createConfigFromPostData(c.Request().Context(), config, h.cfg)
	if err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInvalidConfig,
		})
	}

	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionReconfigure,
		Metadata: runner.RequestMeta{
			Traceparent: potel.TraceContextFromContext(c.Request().Context()),
		},
		Config: gameConfig,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusAccepted, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiStop(c echo.Context) error {
	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionStop,
		Metadata: runner.RequestMeta{
			Traceparent: potel.TraceContextFromContext(c.Request().Context()),
		},
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusAccepted, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiListWorlds(c echo.Context) error {
	if val, err := h.redis.Get(c.Request().Context(), CacheKeyWorlds).Result(); err == nil {
		return c.JSONBlob(http.StatusOK, []byte(val))
	} else if err != redis.Nil {
		slog.Error("Error retrieving backups from cache", slog.Any("error", err))
	}

	slog.Info("cache miss", slog.String("cache_key", CacheKeyWorlds))

	worlds, err := h.world.GetWorlds(c.Request().Context())
	if err != nil {
		slog.Error("Failed to retrieve backup list", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBackup,
		})
	}

	resp := web.SuccessfulResponse[[]web.World]{
		Success: true,
		Data:    worlds,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		slog.Error("Failed to marshal backpu list", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if _, err := h.redis.Set(c.Request().Context(), CacheKeyWorlds, jsonResp, 5*time.Second).Result(); err != nil {
		slog.Error("Failed to store backup list", slog.Any("error", err))
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) handleApiDeleteWorld(c echo.Context) error {
	var req web.DeleteWorldReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if err := h.world.DeleteWorld(c.Request().Context(), req.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusNoContent, web.SuccessfulResponse[any]{
		Success: true,
	})
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
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
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
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.WorldInfo]{
		Success: true,
		Data:    *data,
	})
}

func (h *Handler) validateAndNormalizeConfig(config *web.PendingConfig) bool {
	if config.MachineType == nil || !slices.Contains([]string{"2g", "4g", "8g", "16g", "32g", "64g"}, *config.MachineType) {
		config.MachineType = nil
		return false
	}
	if config.ServerVersion == nil || *config.ServerVersion == "" {
		config.ServerVersion = nil
		return false
	}
	if config.GuessVersion == nil {
		config.GuessVersion = web.BoolP(false)
	}
	if config.WorldSource == nil {
		return false
	} else if !slices.Contains([]string{"backups", "new-world"}, *config.WorldSource) {
		config.WorldSource = nil
		return false
	}
	if config.WorldName == nil {
		return false
	} else if *config.WorldName == "" {
		config.WorldName = nil
		return false
	}
	if config.BackupGen == nil || *config.BackupGen == "" {
		if *config.WorldSource == "new-world" {
			config.BackupGen = nil
		} else {
			return false
		}
	}
	if *config.WorldSource == "new-world" {
		if config.LevelType != nil && !slices.Contains([]string{"default", "flat", "largeBiomes", "amplified", "buffet"}, *config.LevelType) {
			config.LevelType = nil
			return false
		}
		if config.Seed != nil && *config.Seed == "" {
			config.Seed = nil
		}
	} else {
		config.LevelType = nil
		config.Seed = nil
	}

	return true
}

func (h *Handler) handleApiGetConfig(c echo.Context) error {
	var config web.PendingConfig

	if err := h.KVS.Get(c.Request().Context(), "pending-config", &config); err != nil {
		config = web.PendingConfig{
			MachineType:     web.StringP("4g"),
			GuessVersion:    web.BoolP(true),
			InactiveTimeout: web.IntP(30),
		}
	}

	isValid := h.validateAndNormalizeConfig(&config)

	if err := h.KVS.Set(c.Request().Context(), "pending-config", config, 30*24*time.Hour); err != nil {
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.ConfigAndValidity]{
		Success: true,
		Data: web.ConfigAndValidity{
			IsValid: isValid,
			Config:  config,
		},
	})
}

func (h *Handler) handleApiUpdateConfig(c echo.Context) error {
	var newConfig web.PendingConfig
	if err := c.Bind(&newConfig); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	var config web.PendingConfig
	if err := h.KVS.Get(c.Request().Context(), "pending-config", &config); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if err := mergo.Merge(&config, &newConfig, mergo.WithOverride, mergo.WithoutDereference); err != nil {
		slog.Error("Error merging config", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	isValid := h.validateAndNormalizeConfig(&config)

	if err := h.KVS.Set(c.Request().Context(), "pending-config", config, 30*24*time.Hour); err != nil {
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.ConfigAndValidity]{
		Success: true,
		Data: web.ConfigAndValidity{
			IsValid: isValid,
			Config:  config,
		},
	})
}

func (h *Handler) handleApiCreateWorldDownloadLink(c echo.Context) error {
	var req web.CreateWorldLinkReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	url, err := h.world.GetPresignedGetURLWithLifetime(c.Request().Context(), req.ID, time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusCreated, web.SuccessfulResponse[web.DelegatedURL]{
		Success: true,
		Data: web.DelegatedURL{
			URL: url,
		},
	})
}

func (h *Handler) handleApiCreateWorldUploadLink(c echo.Context) error {
	var req web.CreateWorldUploadLinkReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if strings.ContainsAny(req.WorldName, "@/\\") {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	ext := ""
	switch req.MimeType {
	case "application/zip":
		ext = ".zip"
	case "application/x-gzip":
		ext = ".tar.gz"
	case "application/zstd":
		ext = ".tar.zst"
	default:
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	fileName := fmt.Sprintf("%s/user_uploaded_world%s", req.WorldName, ext)

	url, err := h.world.GetPresignedPutURLWithLifetime(c.Request().Context(), fileName, time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[web.DelegatedURL]{
		Success: true,
		Data: web.DelegatedURL{
			URL: url,
		},
	})
}

func (h *Handler) handleApiQuickUndoSnapshot(c echo.Context) error {
	userID := c.Get("access_token").(*auth.Token).UserID

	var config web.SnapshotConfiguration
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if config.Slot < 0 || 10 <= config.Slot {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionSnapshot,
		Metadata: runner.RequestMeta{
			Traceparent: potel.TraceContextFromContext(c.Request().Context()),
		},
		Actor: int(userID),
		Snapshot: &runner.SnapshotConfig{
			Slot: config.Slot,
		},
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusAccepted, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiQuickUndoUndo(c echo.Context) error {
	userID := c.Get("access_token").(*auth.Token).UserID

	var config web.SnapshotConfiguration
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if config.Slot < 0 || 10 <= config.Slot {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if err := h.runnerAction.Push(c.Request().Context(), "default", runner.Action{
		Type: runner.ActionUndo,
		Metadata: runner.RequestMeta{
			Traceparent: potel.TraceContextFromContext(c.Request().Context()),
		},
		Actor: int(userID),
		Snapshot: &runner.SnapshotConfig{
			Slot: config.Slot,
		},
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
	}

	return c.JSON(http.StatusAccepted, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func setupApiQuickUndoRoutes(h *Handler, group *echo.Group) {
	group.POST("/snapshot", h.handleApiQuickUndoSnapshot, scope(auth.ScopeAdmin))
	group.POST("/undo", h.handleApiQuickUndoUndo, scope(auth.ScopeAdmin))
}

func (h *Handler) handleApiUsersChangePassword(c echo.Context) error {
	userID := c.Get("access_token").(*auth.Token).UserID

	var req web.UpdatePassword
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if !isAllowedPassword(req.NewPassword) {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
	}

	var password string
	if err := h.db.NewSelect().Model((*model.User)(nil)).Column("password").Where("id = ? AND deleted_at IS NULL", userID).Scan(c.Request().Context(), &password); err != nil {
		slog.Error("User not found", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	if _, err := h.db.NewUpdate().Model((*model.User)(nil)).Set("password = ?", string(hashedPassword)).Set("initialized = ?", true).Where("id = ? AND deleted_at IS NULL", userID).Exec(c.Request().Context()); err != nil {
		slog.Error("error updating password", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
	}

	return c.JSON(http.StatusOK, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiUsersAdd(c echo.Context) error {
	userID := c.Get("access_token").(*auth.Token).UserID

	var req web.PasswordCredential
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}

	if len(req.UserName) == 0 || len(req.UserName) > 32 {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
	}
	if !isAllowedPassword(req.Password) {
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("error hashing password", slog.Any("error", err))
		return c.JSON(http.StatusInternalServerError, web.ErrorResponse{
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
		return c.JSON(http.StatusBadRequest, web.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrDupUserName,
		})
	}

	return c.JSON(http.StatusCreated, web.SuccessfulResponse[any]{
		Success: true,
	})
}

func setupApiUsersRoutes(h *Handler, group *echo.Group) {
	group.POST("/change-password", h.handleApiUsersChangePassword, scope(auth.ScopeAdmin))
	group.POST("/add", h.handleApiUsersAdd, scope(auth.ScopeAdmin))
}

func (h *Handler) accessTokenMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authorization := c.Request().Header.Get("Authorization")
		if authorization == "" {
			authorization = c.Request().URL.Query().Get("x-auth")
		}

		if !strings.HasPrefix(authorization, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrRequiresAuth,
			})
		}

		accessToken := strings.TrimPrefix(authorization, "Bearer ")

		if token, err := h.authService.Get(c.Request().Context(), accessToken); err != nil {
			return c.JSON(http.StatusUnauthorized, web.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrRequiresAuth,
			})
		} else {
			c.Set("access_token", token)
		}

		return next(c)
	}
}

func scope(scopes ...auth.Scope) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Get("access_token").(*auth.Token)

			for _, scope := range scopes {
				if !token.HasScope(scope) {
					return c.JSON(http.StatusForbidden, web.ErrorResponse{
						Success:   false,
						ErrorCode: entity.ErrCredential,
					})
				}
			}

			return next(c)
		}
	}
}

func (h *Handler) setupApiRoutes(group *echo.Group) {
	needsAuth := group.Group("")
	needsAuth.Use(h.accessTokenMiddleware)
	needsAuth.GET("/streaming", h.handleStream, scope(auth.ScopeAdmin))
	needsAuth.POST("/launch", h.handleApiLaunch, scope(auth.ScopeAdmin))
	needsAuth.POST("/reconfigure", h.handleApiReconfigure, scope(auth.ScopeAdmin))
	needsAuth.POST("/stop", h.handleApiStop, scope(auth.ScopeAdmin))
	needsAuth.GET("/worlds", h.handleApiListWorlds, scope(auth.ScopeAdmin))
	needsAuth.DELETE("/worlds", h.handleApiDeleteWorld, scope(auth.ScopeAdmin))
	needsAuth.GET("/mcversions", h.handleApiMcversions, scope(auth.ScopeAdmin))
	needsAuth.GET("/systeminfo", h.handleApiSystemInfo, scope(auth.ScopeAdmin))
	needsAuth.GET("/worldinfo", h.handleApiWorldInfo, scope(auth.ScopeAdmin))
	needsAuth.GET("/config", h.handleApiGetConfig, scope(auth.ScopeAdmin))
	needsAuth.PUT("/config", h.handleApiUpdateConfig, scope(auth.ScopeAdmin))
	needsAuth.POST("/world-link/download", h.handleApiCreateWorldDownloadLink, scope(auth.ScopeAdmin))
	needsAuth.POST("/world-link/upload", h.handleApiCreateWorldUploadLink, scope(auth.ScopeAdmin))
	setupApiQuickUndoRoutes(h, needsAuth.Group("/quickundo"))
	setupApiUsersRoutes(h, needsAuth.Group("/users"))
}

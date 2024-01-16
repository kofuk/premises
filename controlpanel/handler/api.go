package handler

import (
	"bytes"
	"context"
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

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/dns"
	"github.com/kofuk/premises/controlpanel/gameconfig"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/kofuk/premises/controlpanel/monitor"
	"github.com/kofuk/premises/controlpanel/streaming"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	CacheKeyBackups          = "backups"
	CacheKeyMCVersions       = "mcversions"
	CacheKeySystemInfoPrefix = "system-info"
)

func (h *Handler) handleApiSessionData(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(uint)

	sessionData := entity.SessionData{
		LoggedIn: ok,
	}

	if ok {
		user := model.User{}
		if err := h.db.WithContext(c.Request.Context()).Find(&user, userID).Error; err != nil {
			log.WithError(err).Error("User not found")
			c.JSON(http.StatusOK, entity.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrInternal,
			})
			return
		}
		sessionData.UserName = user.Name
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SessionData]{
		Success: true,
		Data:    sessionData,
	})
}

func (h *Handler) createStreamingEndpoint(stream *streaming.Stream, eventName string) func(c *gin.Context) {
	return func(c *gin.Context) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		writeEvent := func(message []byte) error {
			if _, err := c.Writer.WriteString("event: " + eventName + "\n"); err != nil {
				return err
			}

			data := []byte("data: ")
			data = append(data, message...)
			data = append(data, []byte("\n\n")...)

			if _, err := c.Writer.Write(data); err != nil {
				return err
			}
			c.Writer.Flush()
			return nil
		}

		subscription, statusHistory, err := h.Streaming.SubscribeEvent(c.Request.Context(), stream)
		if err != nil {
			log.WithError(err).Error("Failed to connect to stream")
			c.Status(http.StatusInternalServerError)
			return
		}
		defer subscription.Close()

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-store")
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		for _, entry := range statusHistory {
			if err := writeEvent(entry); err != nil {
				log.WithError(err).Error("Failed to write data")
				return
			}
		}

		eventChannel := subscription.Channel()

	out:
		for {
			select {
			case status := <-eventChannel:
				if err := writeEvent([]byte(status.Payload)); err != nil {
					log.WithError(err).Error("Failed to write server-sent event")
					break out
				}

			case <-ticker.C:
				if _, err := c.Writer.WriteString(": uhaha\n"); err != nil {
					log.WithError(err).Error("Failed to write keep-alive message")
					break out
				}
				c.Writer.Flush()

			case <-c.Request.Context().Done():
				break out
			}
		}
	}
}

func (h *Handler) handleApiSystemStat(c *gin.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-store")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	writeEvent := func(status []byte) error {
		if _, err := c.Writer.WriteString("event: systemstat\n"); err != nil {
			return err
		}

		data := []byte("data: ")
		data = append(data, status...)
		data = append(data, []byte("\n\n")...)

		if _, err := c.Writer.Write(data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	stream := h.Streaming.GetStream(streaming.SysstatStream)

	subscription, _, err := h.Streaming.SubscribeEvent(c.Request.Context(), stream)
	if err != nil {
		log.WithError(err).Error("Failed to connect to stream")
		c.Status(http.StatusInternalServerError)
		return
	}
	defer subscription.Close()

	defer subscription.Close()
	eventChannel := subscription.Channel()

end:
	for {
		select {
		case status := <-eventChannel:
			if err := writeEvent([]byte(status.Payload)); err != nil {
				log.WithError(err).Error("Failed to write server-sent event")
				break end
			}

		case <-ticker.C:
			if _, err := c.Writer.WriteString(": uhaha\n"); err != nil {
				log.WithError(err).Error("Failed to write keep-alive message")
				break end
			}
			c.Writer.Flush()

		case <-c.Request.Context().Done():
			break end
		}
	}
}

func isValidMemSize(memSize int) bool {
	return memSize == 1 || memSize == 2 || memSize == 4 || memSize == 8 || memSize == 16 || memSize == 32 || memSize == 64
}

func (h *Handler) createConfigFromPostData(ctx context.Context, values url.Values, cfg *config.Config) (*runnerEntity.Config, error) {
	if !values.Has("server-version") {
		return nil, errors.New("Server version is not set")
	}
	result := gameconfig.New()

	result.C.ControlPanel = h.cfg.ControlPanel.Origin
	if strings.HasPrefix(h.cfg.ControlPanel.Origin, "http://localhost:") {
		result.C.ControlPanel = strings.Replace(h.cfg.ControlPanel.Origin, "http://localhost", "http://host.docker.internal", 1)
	}

	serverDownloadURL, err := h.MCVersions.GetDownloadURL(ctx, values.Get("server-version"))
	if err != nil {
		return nil, err
	}
	result.SetServer(values.Get("server-version"), serverDownloadURL)

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
	result.SetAllocFromAvailableMemSize(memSizeGB * 1024)
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
			log.WithError(err).Error("Failed to create new request")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.WithError(err).Error("Failed to send request")
			return
		}
		d, _ := io.ReadAll(resp.Body)
		log.Println(string(d))
	}
}

func (h *Handler) shutdownServer(gameServer GameServer, rdb *redis.Client, dnsProvider *dns.DNSProvider, authKey string) {
	defer func() {
		h.serverMutex.Lock()
		defer h.serverMutex.Unlock()
		h.serverRunning = false
	}()

	stdStream := h.Streaming.GetStream(streaming.StandardStream)
	infoStream := h.Streaming.GetStream(streaming.InfoStream)

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessage(entity.EvStopRunner, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if !gameServer.StopVM() {
		if err := h.Streaming.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		h.notifyNonRecoverableFailure(h.cfg, "Failed to stop VM")
		return
	}

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EvStopRunner, 40, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if !gameServer.SaveImage() {
		if err := h.Streaming.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		h.notifyNonRecoverableFailure(h.cfg, "Failed to save image")
		return
	}

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EvStopRunner, 80, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if !gameServer.DeleteVM() {
		if err := h.Streaming.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		h.notifyNonRecoverableFailure(h.cfg, "Failed to delete VM")
		return
	}

	rdb.Del(context.Background(), "monitor-key").Result()

	if dnsProvider != nil {
		dnsProvider.UpdateV4(context.Background(), net.ParseIP("127.0.0.1"))
		dnsProvider.UpdateV6(context.Background(), net.ParseIP("::1"))
	}

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessage(entity.EvStopped, entity.PageLaunch),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if err := h.Cacher.Del(context.Background(), fmt.Sprintf("runner:%s", authKey)); err != nil {
		slog.Error("Failed to delete runner id", slog.Any("error", err))
	}

	if err := h.Streaming.ClearHistory(context.Background(), h.Streaming.GetStream(streaming.SysstatStream)); err != nil {
		log.WithError(err).Error("Unable to clear sysstat history")
	}
}

func (h *Handler) LaunchServer(gameConfig *runnerEntity.Config, gameServer GameServer, memSizeGB int, rdb *redis.Client) {
	stdStream := h.Streaming.GetStream(streaming.StandardStream)
	infoStream := h.Streaming.GetStream(streaming.InfoStream)

	if err := h.Cacher.Set(context.Background(), fmt.Sprintf("runner:%s", gameConfig.AuthKey), "default", -1); err != nil {
		slog.Error("Failed to save runner id", slog.Any("error", err))

		if err := h.Streaming.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}

		h.serverRunning = false
		return
	}

	var dnsProvider *dns.DNSProvider
	if h.cfg.Cloudflare.Token != "" {
		cloudflareDNS, err := dns.NewCloudflareDNS(h.cfg.Cloudflare.Token, h.cfg.Cloudflare.ZoneID)
		if err != nil {
			log.WithError(err).Error("Failed to initialize DNS provider")
		} else {
			dnsProvider = dns.New(cloudflareDNS, h.cfg.Cloudflare.GameDomainName)
		}
	}

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessage(entity.EvCreateRunner, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	h.cfg.MonitorKey = gameConfig.AuthKey
	rdb.Set(context.Background(), "monitor-key", gameConfig.AuthKey, 0).Result()

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EvCreateRunner, 10, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if !gameServer.SetUp(gameConfig, rdb, memSizeGB) {
		if err := h.Streaming.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		h.serverRunning = false
		return
	}

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EvCreateRunner, 50, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if dnsProvider != nil {
		ipAddresses := gameServer.GetIPAddresses()
		if ipAddresses != nil {
			if err := dnsProvider.UpdateV4(context.Background(), ipAddresses.V4); err != nil {
				log.WithError(err).Error("Failed to update IPv4 address")

				if err := h.Streaming.PublishEvent(
					context.Background(),
					infoStream,
					streaming.NewInfoMessage(entity.InfoErrDNS, true),
				); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}
			}
			if err := dnsProvider.UpdateV6(context.Background(), ipAddresses.V6); err != nil {
				log.WithError(err).Error("Failed to update IPv6 address")

				if err := h.Streaming.PublishEvent(
					context.Background(),
					infoStream,
					streaming.NewInfoMessage(entity.InfoErrDNS, true),
				); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}
			}
		}
	}

	if !gameServer.DeleteImage() {
		if err := h.Streaming.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}

		h.serverRunning = false
		return
	}

	if err := h.Streaming.PublishEvent(
		context.Background(),
		stdStream,
		streaming.NewStandardMessageWithProgress(entity.EvCreateRunner, 80, entity.PageLoading),
	); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}
}

func (h *Handler) handleApiLaunch(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.WithError(err).Error("Failed to parse form")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}

	h.serverMutex.Lock()
	defer h.serverMutex.Unlock()

	if h.serverRunning {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrServerRunning,
		})
		return
	}

	gameConfig, err := h.createConfigFromPostData(c.Request.Context(), c.Request.Form, h.cfg)
	if err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInvalidConfig,
		})
		return
	}

	h.serverRunning = true

	machineType := c.PostForm("machine-type")
	h.serverState.machineType = machineType
	memSizeGB, _ := strconv.Atoi(strings.Replace(machineType, "g", "", 1))

	go h.LaunchServer(gameConfig, h.serverImpl, memSizeGB, h.redis)

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiReconfigure(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.WithError(err).Error("Failed to parse form")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}

	formValues := c.Request.Form
	formValues.Set("machine-type", h.serverState.machineType)

	gameConfig, err := h.createConfigFromPostData(c.Request.Context(), formValues, h.cfg)
	if err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInvalidConfig,
		})
		return
	}
	// Use previously generated key.
	gameConfig.AuthKey = h.cfg.MonitorKey

	if err := h.runnerAction.Push(c.Request.Context(), "default", runnerEntity.Action{
		Type:   runnerEntity.ActionReconfigure,
		Config: gameConfig,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
		return
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiStop(c *gin.Context) {
	if err := h.runnerAction.Push(c.Request.Context(), "default", runnerEntity.Action{
		Type: runnerEntity.ActionStop,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
		return
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiBackups(c *gin.Context) {
	if val, err := h.redis.Get(c.Request.Context(), CacheKeyBackups).Result(); err == nil {
		c.Header("Content-Type", "application/json")
		c.Writer.Write([]byte(val))
		return
	} else if err != redis.Nil {
		log.WithError(err).Error("Error retrieving backups from cache")
	}

	log.WithField("cache_key", CacheKeyBackups).Info("cache miss")

	backups, err := h.backup.GetWorlds(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("Failed to retrieve backup list")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBackup,
		})
		return
	}

	resp := entity.SuccessfulResponse[[]entity.WorldBackup]{
		Success: true,
		Data:    backups,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.WithError(err).Error("Failed to marshal backpu list")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	if _, err := h.redis.Set(c.Request.Context(), CacheKeyBackups, jsonResp, 3*time.Minute).Result(); err != nil {
		log.WithError(err).Error("Failed to store backup list")
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/json")
	c.Writer.Write(jsonResp)
}

func (h *Handler) handleApiMcversions(c *gin.Context) {
	versions, err := h.MCVersions.GetVersions(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("Failed to retrieve Minecraft versions")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	versionsEntity := make([]entity.MCVersion, 0)
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

		versionsEntity = append(versionsEntity, entity.MCVersion{
			Name:        ver.ID,
			IsStable:    ver.Type == "release",
			Channel:     channel,
			ReleaseDate: ver.ReleaseTime,
		})
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[[]entity.MCVersion]{
		Success: true,
		Data:    versionsEntity,
	})
}

func (h *Handler) handleApiSystemInfo(c *gin.Context) {
	data, err := monitor.GetSystemInfo(c.Request.Context(), h.cfg, h.cfg.ServerAddr, &h.Cacher)
	if err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}
	c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.SystemInfo]{
		Success: true,
		Data:    *data,
	})
}

func (h *Handler) handleApiWorldInfo(c *gin.Context) {
	data, err := monitor.GetWorldInfo(c.Request.Context(), h.cfg, h.cfg.ServerAddr, &h.Cacher)
	if err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}
	c.JSON(http.StatusOK, entity.SuccessfulResponse[entity.WorldInfo]{
		Success: true,
		Data:    *data,
	})
}

func (h *Handler) handleApiQuickUndoSnapshot(c *gin.Context) {
	if err := h.runnerAction.Push(c.Request.Context(), "default", runnerEntity.Action{
		Type: runnerEntity.ActionSnapshot,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
		return
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiQuickUndoUndo(c *gin.Context) {
	if err := h.runnerAction.Push(c.Request.Context(), "default", runnerEntity.Action{
		Type: runnerEntity.ActionUndo,
	}); err != nil {
		slog.Error("Unable to write action", slog.Any("error", err))
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRemote,
		})
		return
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func setupApiQuickUndoRoutes(h *Handler, group *gin.RouterGroup) {
	group.POST("/snapshot", h.handleApiQuickUndoSnapshot)
	group.POST("/undo", h.handleApiQuickUndoUndo)
}

func (h *Handler) handleApiUsersChangePassword(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	var req entity.UpdatePassword
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}

	if !isAllowedPassword(req.NewPassword) {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
		return
	}

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Find(&user, userID).Error; err != nil {
		log.WithError(err).Error("User not found")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrCredential,
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error hashing password")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}
	user.Password = string(hashedPassword)
	user.Initialized = true

	if err := h.db.WithContext(c.Request.Context()).Save(user).Error; err != nil {
		log.WithError(err).Error("error updating password")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func (h *Handler) handleApiUsersAdd(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	var req entity.PasswordCredential
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}

	if len(req.UserName) == 0 || len(req.UserName) > 32 {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrBadRequest,
		})
		return
	}
	if !isAllowedPassword(req.Password) {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrPasswordRule,
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error hashing password")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrInternal,
		})
		return
	}

	user := &model.User{
		Name:          req.UserName,
		Password:      string(hashedPassword),
		AddedByUserID: &userID,
		Initialized:   false,
	}

	if err := h.db.WithContext(c.Request.Context()).Create(user).Error; err != nil {
		log.WithError(err).Error("error registering user")
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrDupUserName,
		})
		return
	}

	c.JSON(http.StatusOK, entity.SuccessfulResponse[any]{
		Success: true,
	})
}

func setupApiUsersRoutes(h *Handler, group *gin.RouterGroup) {
	group.POST("/change-password", h.handleApiUsersChangePassword)
	group.POST("/add", h.handleApiUsersAdd)
}

func (h *Handler) middlewareSessionCheck(c *gin.Context) {
	// 1. Verify that the request is sent from allowed origin (if needed).
	if c.Request.Method == http.MethodPost || (c.Request.Method == http.MethodGet && c.GetHeader("Upgrade") == "WebSocket") {
		if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
			log.WithField("origin", c.GetHeader("Origin")).Error("origin not allowed")
			c.JSON(http.StatusOK, entity.ErrorResponse{
				Success:   false,
				ErrorCode: entity.ErrBadRequest,
			})
			c.Abort()
			return
		}
	}

	// 2. Verify that the client is logged in.
	session := sessions.Default(c)
	if session.Get("user_id") == nil {
		c.JSON(http.StatusOK, entity.ErrorResponse{
			Success:   false,
			ErrorCode: entity.ErrRequiresAuth,
		})
		c.Abort()
		return
	}
}

func (h *Handler) setupApiRoutes(group *gin.RouterGroup) {
	group.GET("/session-data", h.handleApiSessionData)
	needsAuth := group.Group("")
	needsAuth.Use(h.middlewareSessionCheck)
	needsAuth.GET("/streaming/events", h.createStreamingEndpoint(h.Streaming.GetStream(streaming.StandardStream), "statuschanged"))
	needsAuth.GET("/streaming/info", h.createStreamingEndpoint(h.Streaming.GetStream(streaming.InfoStream), "notify"))
	needsAuth.GET("/streaming/sysstat", h.createStreamingEndpoint(h.Streaming.GetStream(streaming.SysstatStream), "systemstat"))
	needsAuth.GET("/systemstat", h.handleApiSystemStat)
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

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/kofuk/premises/controlpanel/backup"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/gameconfig"
	"github.com/kofuk/premises/controlpanel/mcversions"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/kofuk/premises/controlpanel/monitor"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	CacheKeyBackups          = "backups"
	CacheKeyMCVersions       = "mcversions"
	CacheKeySystemInfoPrefix = "system-info"
)

func (h *Handler) handleApiCurrentUser(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Find(&user, userID).Error; err != nil {
		log.WithError(err).Error("User not found")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "user_name": user.Name})
}

func (h *Handler) handleApiStatus(c *gin.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-store")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	writeEvent := func(status string) error {
		if _, err := c.Writer.WriteString("event: statuschanged\n"); err != nil {
			return err
		}

		if _, err := c.Writer.Write([]byte("data: " + status + "\n\n")); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	lastStatus, err := h.redis.Get(c.Request.Context(), "last-status:default").Result()
	if err != nil && err != redis.Nil {
		log.WithError(err).Error("Failed to read from Redis")
	}
	if err != redis.Nil {
		if err := writeEvent(lastStatus); err != nil {
			log.WithError(err).Error("Failed to write data")
			return
		}
	}

	subscription := h.redis.Subscribe(c.Request.Context(), "status:default")
	defer subscription.Close()
	eventChannel := subscription.Channel()

end:
	for {
		select {
		case status := <-eventChannel:
			if err := writeEvent(status.Payload); err != nil {
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

func (h *Handler) handleApiSystemStat(c *gin.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-store")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	writeEvent := func(status string) error {
		if _, err := c.Writer.WriteString("event: systemstat\n"); err != nil {
			return err
		}

		if _, err := c.Writer.Write([]byte("data: " + status + "\n\n")); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	subscription := h.redis.Subscribe(c.Request.Context(), "systemstat:default")
	defer subscription.Close()
	eventChannel := subscription.Channel()

end:
	for {
		select {
		case status := <-eventChannel:
			if err := writeEvent(status.Payload); err != nil {
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

func createConfigFromPostData(values url.Values, cfg *config.Config) (*gameconfig.GameConfig, error) {
	if !values.Has("server-version") {
		return nil, errors.New("Server version is not set")
	}
	result := gameconfig.New()
	if err := result.SetServerVersion(values.Get("server-version")); err != nil {
		return nil, err
	}

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
		if !values.Has("world-name") {
			return nil, errors.New("World name is not set")
		} else if !values.Has("backup-generation") {
			return nil, errors.New("Backup generation is not set")
		}
		result.SetWorld(values.Get("world-name"), values.Get("backup-generation"))
		result.UseCache(values.Get("use-cache") == "true")
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
	result.SetMegaCredential(cfg.Mega.Email, cfg.Mega.Password)
	result.SetMotd(cfg.Game.Motd)
	result.SetLocale(cfg.ControlPanel.Locale)
	result.SetFolderName(cfg.Mega.FolderName)

	return result, nil
}

func (h *Handler) notifyNonRecoverableFailure(cfg *config.Config, rdb *redis.Client, detail string) {
	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status:   h.L(cfg.ControlPanel.Locale, "monitor.unrecoverable"),
		HasError: true,
		Shutdown: true,
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

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

func (h *Handler) monitorServer(cfg *config.Config, gameServer GameServer, rdb *redis.Client) {
	defer func() {
		h.serverMutex.Lock()
		defer h.serverMutex.Unlock()
		h.serverRunning = false
	}()

	locale := cfg.ControlPanel.Locale

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: h.L(locale, "monitor.connecting"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if err := monitor.MonitorServer(cfg, cfg.ServerAddr, rdb); err != nil {
		log.WithError(err).Error("Failed to monitor server")
	}

	if !gameServer.StopVM(rdb) {
		h.notifyNonRecoverableFailure(cfg, rdb, "Failed to stop VM")
		return
	}
	if !gameServer.SaveImage(rdb) {
		h.notifyNonRecoverableFailure(cfg, rdb, "Failed to save image")
		return
	}
	if !gameServer.DeleteVM() {
		h.notifyNonRecoverableFailure(cfg, rdb, "Failed to delete VM")
		return
	}

	rdb.Del(context.Background(), "monitor-key").Result()

	gameServer.RevertDNS(rdb)

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status:   h.L(locale, "monitor.stopped"),
		Shutdown: true,
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}
}

func (h *Handler) LaunchServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer, memSizeGB int, rdb *redis.Client) {
	locale := cfg.ControlPanel.Locale

	if err := monitor.GenerateTLSKey(cfg, rdb); err != nil {
		log.WithError(err).Error("Failed to generate TLS key")
		if err := monitor.PublishEvent(rdb, monitor.StatusData{
			Status:   h.L(locale, "monitor.tls_keygen.error"),
			HasError: true,
			Shutdown: true,
		}); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		return
	}

	cfg.MonitorKey = gameConfig.AuthKey
	rdb.Set(context.Background(), "monitor-key", gameConfig.AuthKey, 0).Result()

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status:   h.L(locale, "monitor.waiting"),
		HasError: false,
		Shutdown: false,
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	if !gameServer.SetUp(gameConfig, rdb, memSizeGB) {
		if err := monitor.PublishEvent(rdb, monitor.StatusData{
			Status:   h.L(locale, "vm.start.error"),
			HasError: true,
			Shutdown: false,
		}); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		return
	}

	if !gameServer.UpdateDNS(rdb) {
		if err := monitor.PublishEvent(rdb, monitor.StatusData{
			Status:   h.L(locale, "vm.dns.error"),
			HasError: true,
			Shutdown: false,
		}); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		return
	}

	if !gameServer.DeleteImage(rdb) {
		if err := monitor.PublishEvent(rdb, monitor.StatusData{
			Status:   h.L(locale, "vm.image.delete.error"),
			HasError: true,
			Shutdown: false,
		}); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}

		return
	}

	h.monitorServer(cfg, gameServer, rdb)
}

func StopServer(cfg *config.Config, gameServer GameServer, rdb *redis.Client) {
	if err := monitor.StopServer(cfg, cfg.ServerAddr, rdb); err != nil {
		log.WithError(err).Error("Failed to request stopping server")
	}
}

func ReconfigureServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer, rdb *redis.Client) {
	if err := monitor.ReconfigureServer(gameConfig, cfg, cfg.ServerAddr, rdb); err != nil {
		log.WithError(err).Error("Failed to reconfigure server")
	}
}

func (h *Handler) handleApiLaunch(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.WithError(err).Error("Failed to parse form")
		c.JSON(400, gin.H{"success": false, "message": "Form parse error"})
		return
	}

	h.serverMutex.Lock()
	defer h.serverMutex.Unlock()

	if h.serverRunning {
		c.JSON(400, gin.H{"success": false, "message": "The server has already running"})
		return
	}
	h.serverRunning = true

	gameConfig, err := createConfigFromPostData(c.Request.Form, h.cfg)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}

	machineType := c.PostForm("machine-type")
	h.serverState.machineType = machineType
	memSizeGB, _ := strconv.Atoi(strings.Replace(machineType, "g", "", 1))

	go h.LaunchServer(gameConfig, h.cfg, h.serverImpl, memSizeGB, h.redis)

	c.JSON(200, gin.H{"success": true})
}

func (h *Handler) handleApiReconfigure(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		log.WithError(err).Error("Failed to parse form")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Form parse error"})
		return
	}

	formValues := c.Request.Form
	formValues.Set("machine-type", h.serverState.machineType)

	gameConfig, err := createConfigFromPostData(formValues, h.cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	// Use previously generated key.
	gameConfig.AuthKey = h.cfg.MonitorKey

	go ReconfigureServer(gameConfig, h.cfg, h.serverImpl, h.redis)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) handleApiStop(c *gin.Context) {
	go StopServer(h.cfg, h.serverImpl, h.redis)

	c.JSON(200, gin.H{"success": true})
}

func (h *Handler) handleApiBackups(c *gin.Context) {
	if _, ok := c.GetQuery("reload"); ok {
		if _, err := h.redis.Del(c.Request.Context(), CacheKeyBackups).Result(); err != nil {
			log.WithError(err).Error("Failed to delete backup list cache")
		}
	}

	if val, err := h.redis.Get(c.Request.Context(), CacheKeyBackups).Result(); err == nil {
		c.Header("Content-Type", "application/json")
		c.Writer.Write([]byte(val))
		return
	} else if err != redis.Nil {
		log.WithError(err).Error("Error retrieving mcversions cache")
	}

	log.WithField("cache_key", CacheKeyBackups).Info("cache miss")

	backups, err := backup.GetBackupList(&h.cfg.Mega, h.cfg.Mega.FolderName)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve backup list")
		c.Status(http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(backups)
	if err != nil {
		log.WithError(err).Error("Failed to marshal backpu list")
		c.Status(http.StatusInternalServerError)
		return
	}

	if _, err := h.redis.Set(c.Request.Context(), CacheKeyBackups, jsonData, 24*time.Hour).Result(); err != nil {
		log.WithError(err).Error("Failed to store backup list")
	}

	c.Header("Content-Type", "application/json")
	c.Writer.Write(jsonData)
}

func (h *Handler) handleApiMcversions(c *gin.Context) {
	if _, ok := c.GetQuery("reload"); ok {
		if _, err := h.redis.Del(c.Request.Context(), CacheKeyMCVersions).Result(); err != nil {
			log.WithError(err).Error("Failed to delete mcversions cache")
		}
	}

	if val, err := h.redis.Get(c.Request.Context(), CacheKeyMCVersions).Result(); err == nil {
		c.Header("Content-Type", "application/json")
		c.Writer.Write([]byte(val))
		return
	} else if err != redis.Nil {
		log.WithError(err).Error("Error retrieving mcversions cache")
	}

	log.WithField("cache_key", CacheKeyMCVersions).Info("cache miss")

	versions, err := mcversions.GetVersions(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("Failed to retrieve Minecraft versions")
		c.Status(http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(versions)
	if err != nil {
		log.WithError(err).Error("Failed to marshal mcversions")
		c.Status(http.StatusInternalServerError)
		return
	}

	if _, err := h.redis.Set(c.Request.Context(), CacheKeyMCVersions, jsonData, 7*24*time.Hour).Result(); err != nil {
		log.WithError(err).Error("Failed to cache mcversions")
	}

	c.Header("Content-Type", "application/json")
	c.Writer.Write(jsonData)
}

func (h *Handler) handleApiSystemInfo(c *gin.Context) {
	if h.cfg.ServerAddr == "" {
		c.Status(http.StatusTooEarly)
		return
	}

	cacheKey := fmt.Sprintf("%s:%s", CacheKeySystemInfoPrefix, h.cfg.ServerAddr)

	if _, ok := c.GetQuery("reload"); ok {
		if _, err := h.redis.Del(c.Request.Context(), cacheKey).Result(); err != nil {
			log.WithError(err).WithField("server_addr", h.cfg.ServerAddr).Error("Failed to delete system info cache")
		}
	}

	if val, err := h.redis.Get(c.Request.Context(), cacheKey).Result(); err == nil {
		c.Header("Content-Type", "application/json")
		c.Writer.Write([]byte(val))
		return
	} else if err != redis.Nil {
		log.WithError(err).WithField("server_addr", h.cfg.ServerAddr).Error("Error retrieving system info cache")
	}

	log.WithField("cache_key", cacheKey).Info("cache miss")

	data, err := monitor.GetSystemInfoData(c.Request.Context(), h.cfg, h.cfg.ServerAddr, h.redis)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if _, err := h.redis.Set(c.Request.Context(), cacheKey, data, 24*time.Hour).Result(); err != nil {
		log.WithError(err).WithField("server_addr", h.cfg.ServerAddr).Error("Failed to cache mcversions")
	}

	c.Header("Content-Type", "application/json")
	c.Writer.Write(data)
}

func (h *Handler) handleApiWorldInfo(c *gin.Context) {
	if h.cfg.ServerAddr == "" {
		c.Status(http.StatusTooEarly)
		return
	}

	data, err := monitor.GetWorldInfoData(c.Request.Context(), h.cfg, h.cfg.ServerAddr, h.redis)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "application/json")
	c.Writer.Write(data)
}

func (h *Handler) handleApiSnapshot(c *gin.Context) {
	if h.cfg.ServerAddr == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
		return
	}

	if err := monitor.TakeSnapshot(h.cfg, h.cfg.ServerAddr, h.redis); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) handleApiQuickUndoSnapshot(c *gin.Context) {
	if h.cfg.ServerAddr == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
		return
	}

	if err := monitor.QuickSnapshot(h.cfg, h.cfg.ServerAddr, h.redis); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) handleApiQuickUndoUndo(c *gin.Context) {
	if h.cfg.ServerAddr == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
		return
	}

	if err := monitor.QuickUndo(h.cfg, h.cfg.ServerAddr, h.redis); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func setupApiQuickUndoRoutes(h *Handler, group *gin.RouterGroup) {
	group.POST("/snapshot", h.handleApiQuickUndoSnapshot)
	group.POST("/undo", h.handleApiQuickUndoUndo)
}

func (h *Handler) handleApiUsersChangePassword(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "reason": "Invalid form data"})
		return
	}

	password := c.Request.Form.Get("password")
	newPassword := c.Request.Form.Get("new-password")

	if !isAllowedPassword(newPassword) {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "account.password.disallowed")})
		return
	}

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Find(&user, userID).Error; err != nil {
		log.WithError(err).Error("User not found")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "login.error")})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error registering user")
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": "Error registering user"})
		return
	}
	user.Password = string(hashedPassword)
	user.Initialized = true

	if err := h.db.WithContext(c.Request.Context()).Save(user).Error; err != nil {
		log.WithError(err).Error("error updating password")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) handleApiUsersAdd(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id").(uint)

	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "reason": "Invalid form data"})
		return
	}

	newUsername := c.Request.Form.Get("username")
	password := c.Request.Form.Get("password")

	if len(newUsername) == 0 && len(password) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "reason": "username or password is empty"})
		return
	}
	if !isAllowedPassword(password) {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "account.password.disallowed")})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("error registering user")
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": "Error registering user"})
		return
	}

	user := &model.User{
		Name:          newUsername,
		Password:      string(hashedPassword),
		AddedByUserID: &userID,
		Initialized:   false,
	}

	if err := h.db.WithContext(c.Request.Context()).Create(user).Error; err != nil {
		log.WithError(err).Error("error registering user")
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "account.user.exists")})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func setupApiUsersRoutes(h *Handler, group *gin.RouterGroup) {
	group.POST("/change-password", h.handleApiUsersChangePassword)
	group.POST("/add", h.handleApiUsersAdd)
}

func (h *Handler) handleApiWebauthnRoot(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	var credentials []model.Credential
	if err := h.db.WithContext(c.Request.Context()).Where("owner_id = ?", userID).Find(&credentials).Error; err != nil {
		log.WithError(err).Error("Error fetching credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}
	type credentialRespItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	credentialResp := make([]credentialRespItem, 0)
	for _, c := range credentials {
		credentialResp = append(credentialResp, credentialRespItem{
			ID:   c.UUID,
			Name: c.KeyName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": credentialResp})
}

func (h *Handler) handleApiWebauthnUuid(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	keyUuid := c.Param("uuid")

	var credential model.Credential
	if err := h.db.WithContext(c.Request.Context()).Where("owner_id = ? AND uuid = ?", userID, keyUuid).First(&credential).Error; err != nil {
		log.WithError(err).Error("Error fetching credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	if err := h.db.WithContext(c.Request.Context()).Delete(&credential).Error; err != nil {
		log.WithError(err).Error("Error fetching credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"success": true})
}

func (h *Handler) handleApiWebauthnBegin(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Find(&user, userID).Error; err != nil {
		log.WithError(err).Error("User expected to be found, but didn't")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	waUser := webAuthnUser{
		user: user,
	}

	var credentials []model.Credential
	if err := h.db.WithContext(c.Request.Context()).Where("owner_id = ?", userID).Find(&credentials).Error; err != nil {
		log.WithError(err).Error("Error fetching credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}
	for _, c := range credentials {
		waUser.registerCredential(c)
	}

	registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
		credCreationOpts.CredentialExcludeList = waUser.getCredentialExcludeList()
	}

	options, sessionData, err := h.webauthn.BeginRegistration(&waUser, registerOptions)
	if err != nil {
		log.WithError(err).Error("Error registration")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	marshaled, err := json.Marshal(sessionData)
	if err != nil {
		log.WithError(err).Error("")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	session.Set("hwkey_registration", string(marshaled))
	session.Save()

	c.JSON(http.StatusOK, gin.H{"success": true, "options": options})
}

func (h *Handler) handleApiWebauthnFinish(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	sessionDataMarshaled := session.Get("hwkey_registration")
	session.Delete("hwkey_registration")
	session.Save()

	keyName := c.Query("name")
	if keyName == "" {
		keyName = "Key"
	}

	sessionData := webauthn.SessionData{}
	if err := json.Unmarshal([]byte(sessionDataMarshaled.(string)), &sessionData); err != nil {
		log.WithError(err).Error("Can't unmarshal session data")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	user := model.User{}
	if err := h.db.WithContext(c.Request.Context()).Find(&user, userID).Error; err != nil {
		log.WithError(err).Error("User expected to be found, but didn't")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	waUser := webAuthnUser{
		user: user,
	}

	var credentials []model.Credential
	if err := h.db.WithContext(c.Request.Context()).Where("owner_id = ?", userID).Find(&credentials).Error; err != nil {
		log.WithError(err).Error("Error fetching credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}
	for _, c := range credentials {
		waUser.registerCredential(c)
	}

	credData, err := h.webauthn.FinishRegistration(&waUser, sessionData, c.Request)
	if err != nil {
		log.WithError(err).WithField("info", err.(*protocol.Error).DevInfo).Error("Error registration")
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "hardwarekey.verify.error")})
		return
	}

	keyUuid := uuid.New().String()
	credential := model.Credential{
		OwnerID:                user.ID,
		UUID:                   keyUuid,
		KeyName:                keyName,
		CredentialID:           credData.ID,
		PublicKey:              credData.PublicKey,
		AttestationType:        credData.AttestationType,
		AuthenticatorAAGUID:    credData.Authenticator.AAGUID,
		AuthenticatorSignCount: credData.Authenticator.SignCount,
	}

	var exists bool
	if err := h.db.WithContext(c.Request.Context()).Model(credential).Select("count(*) > 0").Where("owner_id = ? AND credential_id = ?", userID, credential.CredentialID).Find(&exists).Error; err != nil {
		log.WithError(err).Error("Error fetching public key count")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	if exists {
		c.JSON(http.StatusOK, gin.H{"success": false, "reason": h.L(h.cfg.ControlPanel.Locale, "hardwarekey.already_registered")})
		return
	}

	if err := h.db.WithContext(c.Request.Context()).Create(&credential).Error; err != nil {
		log.WithError(err).Error("Error saving credential")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func setupApiWebauthnRoutes(h *Handler, group *gin.RouterGroup) {
	group.GET("", h.handleApiWebauthnRoot)
	group.POST("/:uuid", h.handleApiWebauthnUuid)
	group.POST("/begin", h.handleApiWebauthnBegin)
	group.POST("/finish", h.handleApiWebauthnFinish)
}

func (h *Handler) middlewareSessionCheck(c *gin.Context) {
	// 1. Verify that the client is logged in.
	session := sessions.Default(c)
	if session.Get("user_id") == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Not logged in"})
		c.Abort()
		return
	}

	// 2. Verify that the request is sent from allowed origin (if needed).
	if c.Request.Method == http.MethodPost || (c.Request.Method == http.MethodGet && c.GetHeader("Upgrade") == "WebSocket") {
		if c.GetHeader("Origin") != h.cfg.ControlPanel.Origin {
			log.WithField("origin", c.GetHeader("Origin")).Error("origin not allowed")
			c.JSON(400, gin.H{"success": false, "message": "Invalid request (origin not allowed)"})
			c.Abort()
			return
		}
	}
}

func (h *Handler) setupApiRoutes(group *gin.RouterGroup) {
	group.Use(h.middlewareSessionCheck)
	group.GET("/current-user", h.handleApiCurrentUser)
	group.GET("/status", h.handleApiStatus)
	group.GET("/systemstat", h.handleApiSystemStat)
	group.POST("/launch", h.handleApiLaunch)
	group.POST("/reconfigure", h.handleApiReconfigure)
	group.POST("/stop", h.handleApiStop)
	group.GET("/backups", h.handleApiBackups)
	group.GET("/mcversions", h.handleApiMcversions)
	group.GET("/systeminfo", h.handleApiSystemInfo)
	group.GET("/worldinfo", h.handleApiWorldInfo)
	group.POST("/snapshot", h.handleApiSnapshot)
	setupApiQuickUndoRoutes(h, group.Group("/quickundo"))
	setupApiUsersRoutes(h, group.Group("/users"))
	setupApiWebauthnRoutes(h, group.Group("/hardwarekey"))
}

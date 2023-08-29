package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-contrib/sessions"
	redisSess "github.com/gin-contrib/sessions/redis"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/language"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/kofuk/premises/backup"
	"github.com/kofuk/premises/home/config"
	"github.com/kofuk/premises/home/gameconfig"
	"github.com/kofuk/premises/home/mcversions"
	"github.com/kofuk/premises/home/monitor"
)

//go:embed i18n/*.json
var i18nData embed.FS

//go:embed etc/robots.txt
var robotsTxt []byte

var localizeBundle *i18n.Bundle

var isServerSetUp bool

type User struct {
	gorm.Model
	Name          string `gorm:"type:varchar(32);not null;uniqueIndex"`
	Password      string `gorm:"type:varchar(64);not null"`
	AddedByUserID *uint
	AddedBy       *User        `gorm:"foreignKey:AddedByUserID"`
	Credentials   []Credential `gorm:"foreignKey:OwnerID"`
	Initialized   bool         `gorm:"not null"`
}

type Credential struct {
	gorm.Model
	OwnerID                uint   `gorm:"not null"`
	UUID                   string `gorm:"type:varchar(36);not null;unique"`
	KeyName                string `gorm:"type:varchar(128);not null"`
	CredentialID           []byte `gorm:"type:bytea;not null"`
	PublicKey              []byte `gorm:"type:bytea;not null"`
	AttestationType        string `gorm:"type:varchar(16);not null"`
	AuthenticatorAAGUID    []byte `gorm:"type:bytea;not null"`
	AuthenticatorSignCount uint32 `gorm:"not null"`
}

type webAuthnUser struct {
	user        User
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, uint64(u.user.ID))
	return result
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.user.Name
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.user.Name
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webAuthnUser) registerCredential(cred Credential) {
	u.credentials = append(u.credentials, webauthn.Credential{
		ID:              cred.CredentialID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Authenticator: webauthn.Authenticator{
			AAGUID:    cred.AuthenticatorAAGUID,
			SignCount: cred.AuthenticatorSignCount,
		},
	})
}

func (u *webAuthnUser) getCredentialExcludeList() []protocol.CredentialDescriptor {
	var result []protocol.CredentialDescriptor
	for _, c := range u.credentials {
		result = append(result, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.ID,
		})
	}
	return result
}

func L(locale, msgId string) string {
	if localizeBundle == nil {
		return msgId
	}

	localizer := i18n.NewLocalizer(localizeBundle, locale)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
	if err != nil {
		log.WithError(err).Error("Error loading localized message. Fallback to \"en\"")
		localizer := i18n.NewLocalizer(localizeBundle, "en")
		msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgId})
		if err != nil {
			log.WithError(err).Error("Error loading localized message (fallback)")
			return msgId
		}
		return msg
	}
	return msg
}

func loadI18nData() error {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	ents, err := i18nData.ReadDir("i18n")
	if err != nil {
		return err
	}
	for _, ent := range ents {
		if _, err := bundle.LoadMessageFileFS(i18nData, "i18n/"+ent.Name()); err != nil {
			return err
		}
	}
	localizeBundle = bundle
	return nil
}

type serverState struct {
	statusMu         sync.Mutex
	status           monitor.StatusData
	selectedWorld    string
	monitorChan      chan *monitor.StatusData
	monitorClients   []chan *monitor.StatusData
	monitorClientsMu sync.Mutex
	machineType      string
}

var server serverState

const (
	CacheKeyBackups          = "backups"
	CacheKeyMCVersions       = "mcversions"
	CacheKeySystemInfoPrefix = "system-info"
)

func (s *serverState) addMonitorClient(ch chan *monitor.StatusData) {
	s.monitorClientsMu.Lock()
	defer s.monitorClientsMu.Unlock()

	s.monitorClients = append(s.monitorClients, ch)
}

func (s *serverState) removeMonitorClient(ch chan *monitor.StatusData) {
	s.monitorClientsMu.Lock()
	defer s.monitorClientsMu.Unlock()

	for i, c := range s.monitorClients {
		if c == ch {
			if i != len(s.monitorClients)-1 {
				s.monitorClients[i] = s.monitorClients[len(s.monitorClients)-1]
			}
			s.monitorClients = s.monitorClients[:len(s.monitorClients)-1]
			break
		}
	}
}

func (s *serverState) dispatchMonitorEvent(rdb *redis.Client) {
	for {
		status := <-s.monitorChan

		s.statusMu.Lock()
		s.status = *status
		s.statusMu.Unlock()

		if status.Shutdown {
			if _, err := rdb.Del(context.Background(), CacheKeyBackups).Result(); err != nil {
				log.WithError(err).Error("Failed to delete backup list cache")
			}
		}

		s.monitorClientsMu.Lock()
		for _, ch := range s.monitorClients {
			go func(ch chan *monitor.StatusData) {
				defer func() {
					if err := recover(); err != nil {
						log.WithField("error", err).Error("Recovering previous error")
					}
				}()

				ch <- status
			}(ch)
		}
		s.monitorClientsMu.Unlock()
	}
}

func notifyNonRecoverableFailure(cfg *config.Config, detail string) {
	server.monitorChan <- &monitor.StatusData{
		Status:   L(cfg.ControlPanel.Locale, "monitor.unrecoverable"),
		HasError: true,
		Shutdown: true,
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

func monitorServer(cfg *config.Config, gameServer GameServer, rdb *redis.Client) {
	locale := cfg.ControlPanel.Locale

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "monitor.connecting"),
	}

	if err := monitor.MonitorServer(cfg, cfg.ServerAddr, server.monitorChan, rdb); err != nil {
		log.WithError(err).Error("Failed to monitor server")
	}

	if !gameServer.StopVM() {
		notifyNonRecoverableFailure(cfg, "Failed to stop VM")
		return
	}
	if !gameServer.SaveImage() {
		notifyNonRecoverableFailure(cfg, "Failed to save image")
		return
	}
	if !gameServer.DeleteVM() {
		notifyNonRecoverableFailure(cfg, "Failed to delete VM")
		return
	}

	rdb.Del(context.Background(), "monitor-key").Result()

	gameServer.RevertDNS()

	server.monitorChan <- &monitor.StatusData{
		Status:   L(locale, "monitor.stopped"),
		Shutdown: true,
	}
}

func LaunchServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, gameServer GameServer, memSizeGB int, rdb *redis.Client) {
	locale := cfg.ControlPanel.Locale

	if err := monitor.GenerateTLSKey(cfg, rdb); err != nil {
		log.WithError(err).Error("Failed to generate TLS key")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "monitor.tls_keygen.error"),
			HasError: true,
			Shutdown: true,
		}
		return
	}

	cfg.MonitorKey = gameConfig.AuthKey
	rdb.Set(context.Background(), "monitor-key", gameConfig.AuthKey, 0).Result()

	server.monitorChan <- &monitor.StatusData{
		Status:   L(locale, "monitor.waiting"),
		HasError: false,
		Shutdown: false,
	}

	if !gameServer.SetUp(gameConfig, rdb, memSizeGB) {
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.start.error"),
			HasError: true,
			Shutdown: false,
		}
		return
	}

	if !gameServer.UpdateDNS() {
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
			Shutdown: false,
		}
		return
	}

	if !gameServer.DeleteImage() {
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.image.delete.error"),
			HasError: true,
			Shutdown: false,
		}

		return
	}

	go monitorServer(cfg, gameServer, rdb)
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

//go:embed templates/*.html
var templates embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// already checked by middleware
	CheckOrigin: func(*http.Request) bool { return true },
}

func guessAndHandleCurrentVMState(cfg *config.Config, gameServer GameServer, rdb *redis.Client) {
	if gameServer.VMExists() {
		if gameServer.VMRunning() {
			monitorKey, err := rdb.Get(context.Background(), "monitor-key").Result()
			if err != nil {
				log.WithError(err).Info("Failed to read previous monitor key")
				return
			}
			cfg.MonitorKey = string(monitorKey)

			if gameServer.ImageExists() {
				log.Info("Server seems to be running, but remote image exists")
				gameServer.DeleteImage()
			}

			gameServer.UpdateDNS()

			log.Info("Start monitoring server")
			go monitorServer(cfg, gameServer, rdb)
		} else {
			if !gameServer.ImageExists() && !gameServer.SaveImage() {
				notifyNonRecoverableFailure(cfg, "Invalid state")
				return
			}
			if !gameServer.DeleteVM() {
				notifyNonRecoverableFailure(cfg, "Failed to delete VM")
				return
			}
		}
	}
}

func isAllowedPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	if strings.IndexAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") < 0 {
		return false
	}
	if strings.IndexAny(password, "0123456789") < 0 {
		return false
	}
	return true
}

func main() {
	log.SetReportCaller(true)
	if err := loadI18nData(); err != nil {
		log.Fatal(err)
	}

	if err := godotenv.Load(); err != nil {
		log.WithError(err).Info("Failed to load .env file. If you want to use real envvars, you can ignore this diag safely.")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load config")
	}

	if cfg.Debug.Env {
		if err := os.MkdirAll("/tmp/premises/gamedata", 0755); err != nil {
			log.WithError(err).Info("Cannot create directory for debug environment")
		}
	}

	if cfg.Debug.Web {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	origin, err := url.Parse(cfg.ControlPanel.Origin)
	if err != nil {
		log.WithError(err).Fatal("Failed to parse origin URL")
	}
	wauthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Premises",
		RPID:          origin.Hostname(),
		RPOrigin:      cfg.ControlPanel.Origin,
	})

	sqlDB := postgres.Open(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Etc/UTC", cfg.ControlPanel.Postgres.Address, cfg.ControlPanel.Postgres.Port, cfg.ControlPanel.Postgres.User, cfg.ControlPanel.Postgres.Password, cfg.ControlPanel.Postgres.DBName))
	db, err := gorm.Open(sqlDB, &gorm.Config{})
	if err != nil {
		log.WithError(err).Fatal("Error opening database")
	}
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Credential{})

	if err := db.Model(&User{}).Select("COUNT(id) > 0").Find(&isServerSetUp).Error; err != nil {
		log.WithError(err).Fatal("Failed to read from db")
	}

	bindAddr := ":8000"
	if len(os.Args) > 1 {
		bindAddr = os.Args[1]
	}

	server.status.Status = L(cfg.ControlPanel.Locale, "monitor.stopped")
	server.status.Shutdown = true

	monitorChan := make(chan *monitor.StatusData)
	server.monitorChan = monitorChan

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.ControlPanel.Redis.Address,
		Password: cfg.ControlPanel.Redis.Password,
	})

	var gameServer GameServer
	if cfg.Debug.Runner {
		gameServer = NewLocalDebugServer(cfg)
	} else {
		gameServer = NewConohaServer(cfg)
	}

	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	template := template.New("")
	templateEntries, err := templates.ReadDir("templates")
	for _, ent := range templateEntries {
		data, err := templates.ReadFile(filepath.Join("templates", ent.Name()))
		if err != nil {
			log.WithError(err).Fatal("Failed to load templates")
		}
		template.New(ent.Name()).Parse(string(data))
	}
	r.SetHTMLTemplate(template)

	sessionStore, err := redisSess.NewStore(4, "tcp", cfg.ControlPanel.Redis.Address, cfg.ControlPanel.Redis.Password, []byte(cfg.ControlPanel.Secret))
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize Redis store")
	}
	redisSess.SetKeyPrefix(sessionStore, "session:")
	r.Use(sessions.Sessions("session", sessionStore))

	r.NoRoute(static.Serve("/", static.LocalFile("gen", false)))

	r.GET("/", func(c *gin.Context) {
		if !isServerSetUp {
			c.HTML(200, "setup.html", nil)
			return
		}

		session := sessions.Default(c)
		if session.Get("user_id") != nil {
			c.HTML(200, "control.html", nil)
		} else {
			c.HTML(200, "login.html", nil)
		}
	})
	if !isServerSetUp {
		r.POST("/setup", func(c *gin.Context) {
			if isServerSetUp {
				c.Status(http.StatusNotFound)
				return
			}
			if c.GetHeader("Origin") != cfg.ControlPanel.Origin {
				log.WithField("cfg", cfg.ControlPanel.Origin).Println("Access from disallowed origin")
				c.Status(http.StatusBadRequest)
				return
			}

			if err := c.Request.ParseForm(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "reason": "Invalid form data"})
				return
			}

			username := c.Request.Form.Get("username")
			password := c.Request.Form.Get("password")

			if len(username) == 0 && len(password) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "reason": "username or password is empty"})
				return
			}
			if !isAllowedPassword(password) {
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "account.password.disallowed")})
				return
			}

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				log.WithError(err).Error("error registering user")
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": "Error registering user"})
				return
			}

			user := &User{
				Name:          username,
				Password:      string(hashedPassword),
				AddedByUserID: nil,
				Initialized:   true,
			}

			if err := db.Create(user).Error; err != nil {
				log.WithError(err).Error("error registering user")
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "account.user.exists")})
				return
			}

			isServerSetUp = true

			session := sessions.Default(c)
			session.Set("user_id", user.ID)
			session.Save()

			c.JSON(http.StatusOK, gin.H{"success": true})
		})
	}
	r.POST("/login", func(c *gin.Context) {
		if c.GetHeader("Origin") != cfg.ControlPanel.Origin {
			c.Status(http.StatusBadGateway)
			return
		}

		username := c.PostForm("username")
		password := c.PostForm("password")

		user := User{}
		if err := db.Where("name = ?", username).First(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}

		session := sessions.Default(c)
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil {
			if !user.Initialized {
				session.Set("change_password_user_id", user.ID)
				session.Save()
				c.JSON(http.StatusOK, gin.H{"success": true, "needsChangePassword": true})
			} else {
				session.Set("user_id", user.ID)
				session.Save()
				c.JSON(http.StatusOK, gin.H{"success": true, "needsChangePassword": false})
			}
		} else {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
		}
	})

	r.POST("/login/reset-password", func(c *gin.Context) {
		if c.GetHeader("Origin") != cfg.ControlPanel.Origin {
			c.Status(http.StatusBadGateway)
			return
		}

		session := sessions.Default(c)
		user_id := session.Get("change_password_user_id")

		username := c.PostForm("username")
		password := c.PostForm("password")

		user := User{}
		if err := db.Where("name = ?", username).First(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}

		if user.ID != user_id {
			c.JSON(http.StatusOK, gin.H{"success": false})
			return
		}

		if !isAllowedPassword(password) {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "account.password.disallowed")})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.WithError(err).Error("error registering user")
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": "Error registering user"})
			return
		}
		user.Password = string(hashedPassword)
		user.Initialized = true

		if err := db.Save(user).Error; err != nil {
			log.WithError(err).Error("error updating password")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
		}

		session.Set("user_id", user.ID)
		session.Delete("change_password_user_id")
		session.Save()

		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	r.POST("/login/hardwarekey/begin", func(c *gin.Context) {
		if c.GetHeader("Origin") != cfg.ControlPanel.Origin {
			c.Status(http.StatusBadGateway)
			return
		}

		username := c.PostForm("username")

		user := User{}
		if err := db.Where("name = ?", username).Preload("Credentials").First(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}

		waUser := webAuthnUser{
			user: user,
		}
		if len(user.Credentials) == 0 {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}
		for _, c := range user.Credentials {
			waUser.registerCredential(c)
		}

		options, sessionData, err := wauthn.BeginLogin(&waUser)
		if err != nil {
			log.WithError(err).Error("error beginning login")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
			return
		}

		marshaled, err := json.Marshal(sessionData)
		if err != nil {
			log.WithError(err).Error("error beginning login")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
			return
		}

		session := sessions.Default(c)
		session.Set("hwkey_auth_user_id", user.ID)
		session.Set("hwkey_authentication", string(marshaled))
		session.Save()

		c.JSON(http.StatusOK, gin.H{"success": true, "options": options})
	})

	r.POST("/login/hardwarekey/finish", func(c *gin.Context) {
		if c.GetHeader("Origin") != cfg.ControlPanel.Origin {
			c.Status(http.StatusBadGateway)
			return
		}

		session := sessions.Default(c)
		userID := session.Get("hwkey_auth_user_id")
		marshaledData := session.Get("hwkey_authentication")
		session.Delete("hwkey_authentication")
		session.Delete("hwkey_auth_user_id")
		defer session.Save()

		var sessionData webauthn.SessionData
		if err := json.Unmarshal([]byte(marshaledData.(string)), &sessionData); err != nil {
			log.WithError(err).Error("Failed to unmarshal session data")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
			return
		}

		user := User{}
		if err := db.Preload("Credentials").Find(&user, userID).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}

		waUser := webAuthnUser{
			user: user,
		}
		for _, c := range user.Credentials {
			waUser.registerCredential(c)
		}

		cred, err := wauthn.FinishLogin(&waUser, sessionData, c.Request)
		if err != nil {
			log.WithError(err).Error("error finishing login")
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}

		if cred.Authenticator.CloneWarning {
			log.Error("maybe a cloned authenticator used")
			c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
			return
		}

		var usedCred *Credential
		for _, c := range user.Credentials {
			if bytes.Equal(c.CredentialID, cred.ID) {
				usedCred = &c
				break
			}
		}
		if usedCred == nil {
			log.WithError(err).Error("credential to update did not found")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
			return
		}

		usedCred.AuthenticatorSignCount = cred.Authenticator.SignCount
		if err := db.Save(usedCred).Error; err != nil {
			log.WithError(err).Warn("failed to save credential")
		}

		session.Set("user_id", userID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	r.GET("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Delete("user_id")
		session.Save()
		c.Redirect(http.StatusFound, "/")
	})

	r.GET("/robots.txt", func(c *gin.Context) {
		c.Writer.Write(robotsTxt)
	})

	api := r.Group("api")
	api.Use(func(c *gin.Context) {
		// 1. Verify that request is sent from allowed origin.
		if c.Request.Method == http.MethodPost || (c.Request.Method == http.MethodGet && c.GetHeader("Upgrade") == "WebSocket") {
			if c.GetHeader("Origin") == cfg.ControlPanel.Origin {
				// 2. Verify that client is logged in.
				session := sessions.Default(c)
				if session.Get("user_id") == nil {
					c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Not logged in"})
					c.Abort()
				}
				return
			}
			log.WithField("origin", c.GetHeader("Origin")).Error("origin not allowed")
			c.JSON(400, gin.H{"success": false, "message": "Invalid request (origin not allowed)"})
			c.Abort()
		}
	})
	{
		api.GET("/status", func(c *gin.Context) {
			ch := make(chan *monitor.StatusData)
			server.addMonitorClient(ch)
			defer close(ch)
			defer server.removeMonitorClient(ch)

			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			c.Writer.Header().Set("Content-Type", "text/event-stream")
			c.Writer.Header().Set("Cache-Control", "no-store")
			c.Writer.Header().Set("X-Accel-Buffering", "no")

			writeEvent := func(status *monitor.StatusData) error {
				if _, err := c.Writer.WriteString("event: statuschanged\n"); err != nil {
					return err
				}
				if _, err := c.Writer.WriteString("data: "); err != nil {
					return err
				}

				if data, err := json.Marshal(status); err != nil {
					return err
				} else {
					if _, err := c.Writer.Write(data); err != nil {
						return err
					}
				}
				if _, err := c.Writer.WriteString("\n\n"); err != nil {
					return err
				}
				c.Writer.Flush()
				return nil
			}

			server.statusMu.Lock()
			if err := writeEvent(&server.status); err != nil {
				log.WithError(err).Error("Failed to write data")
				return
			}
			server.statusMu.Unlock()

			for {
				select {
				case status := <-ch:
					if err := writeEvent(status); err != nil {
						log.WithError(err).Error("Failed to write server-sent event")
						goto end
					}

				case <-ticker.C:
					if _, err := c.Writer.WriteString(": uhaha\n"); err != nil {
						log.WithError(err).Error("Failed to marshal status data")
						goto end
					}
					c.Writer.Flush()
				}
			}
		end:
		})

		api.POST("/launch", func(c *gin.Context) {
			server.statusMu.Lock()
			defer server.statusMu.Unlock()

			if err := c.Request.ParseForm(); err != nil {
				log.WithError(err).Error("Failed to parse form")
				c.JSON(400, gin.H{"success": false, "message": "Form parse error"})
				return
			}

			gameConfig, err := createConfigFromPostData(c.Request.Form, cfg)
			if err != nil {
				c.JSON(400, gin.H{"success": false, "message": err.Error()})
				return
			}

			machineType := c.PostForm("machine-type")
			server.machineType = machineType
			memSizeGB, _ := strconv.Atoi(strings.Replace(machineType, "g", "", 1))

			go LaunchServer(gameConfig, cfg, gameServer, memSizeGB, rdb)

			c.JSON(200, gin.H{"success": true})
		})

		api.POST("/reconfigure", func(c *gin.Context) {
			if err := c.Request.ParseForm(); err != nil {
				log.WithError(err).Error("Failed to parse form")
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Form parse error"})
				return
			}

			formValues := c.Request.Form
			formValues.Set("machine-type", server.machineType)

			gameConfig, err := createConfigFromPostData(formValues, cfg)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
				return
			}
			// Use previously generated key.
			gameConfig.AuthKey = cfg.MonitorKey

			go ReconfigureServer(gameConfig, cfg, gameServer, rdb)

			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		api.POST("/stop", func(c *gin.Context) {
			server.statusMu.Lock()
			defer server.statusMu.Unlock()

			go StopServer(cfg, gameServer, rdb)

			c.JSON(200, gin.H{"success": true})
		})

		api.GET("/backups", func(c *gin.Context) {
			if _, ok := c.GetQuery("reload"); ok {
				if _, err := rdb.Del(context.Background(), CacheKeyBackups).Result(); err != nil {
					log.WithError(err).Error("Failed to delete backup list cache")
				}
			}

			if val, err := rdb.Get(context.Background(), CacheKeyBackups).Result(); err == nil {
				c.Header("Content-Type", "application/json")
				c.Writer.Write([]byte(val))
				return
			} else if err != redis.Nil {
				log.WithError(err).Error("Error retrieving mcversions cache")
			}

			log.WithField("cache_key", CacheKeyBackups).Info("cache miss")

			backups, err := backup.GetBackupList(&cfg.Mega, cfg.Mega.FolderName)
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

			if _, err := rdb.Set(context.Background(), CacheKeyBackups, jsonData, 24*time.Hour).Result(); err != nil {
				log.WithError(err).Error("Failed to store backup list")
			}

			c.Header("Content-Type", "application/json")
			c.Writer.Write(jsonData)
		})

		api.GET("/mcversions", func(c *gin.Context) {
			if _, ok := c.GetQuery("reload"); ok {
				if _, err := rdb.Del(context.Background(), CacheKeyMCVersions).Result(); err != nil {
					log.WithError(err).Error("Failed to delete mcversions cache")
				}
			}

			if val, err := rdb.Get(context.Background(), CacheKeyMCVersions).Result(); err == nil {
				c.Header("Content-Type", "application/json")
				c.Writer.Write([]byte(val))
				return
			} else if err != redis.Nil {
				log.WithError(err).Error("Error retrieving mcversions cache")
			}

			log.WithField("cache_key", CacheKeyMCVersions).Info("cache miss")

			versions, err := mcversions.GetVersions()
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

			if _, err := rdb.Set(context.Background(), CacheKeyMCVersions, jsonData, 7*24*time.Hour).Result(); err != nil {
				log.WithError(err).Error("Failed to cache mcversions")
			}

			c.Header("Content-Type", "application/json")
			c.Writer.Write(jsonData)
		})

		api.GET("/systeminfo", func(c *gin.Context) {
			if cfg.ServerAddr == "" {
				c.Status(http.StatusTooEarly)
				return
			}

			cacheKey := fmt.Sprintf("%s:%s", CacheKeySystemInfoPrefix, cfg.ServerAddr)

			if _, ok := c.GetQuery("reload"); ok {
				if _, err := rdb.Del(context.Background(), cacheKey).Result(); err != nil {
					log.WithError(err).WithField("server_addr", cfg.ServerAddr).Error("Failed to delete system info cache")
				}
			}

			if val, err := rdb.Get(context.Background(), cacheKey).Result(); err == nil {
				c.Header("Content-Type", "application/json")
				c.Writer.Write([]byte(val))
				return
			} else if err != redis.Nil {
				log.WithError(err).WithField("server_addr", cfg.ServerAddr).Error("Error retrieving system info cache")
			}

			log.WithField("cache_key", cacheKey).Info("cache miss")

			data, err := monitor.GetSystemInfoData(cfg, cfg.ServerAddr, rdb)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			if _, err := rdb.Set(context.Background(), cacheKey, data, 24*time.Hour).Result(); err != nil {
				log.WithError(err).WithField("server_addr", cfg.ServerAddr).Error("Failed to cache mcversions")
			}

			c.Header("Content-Type", "application/json")
			c.Writer.Write(data)
		})

		api.GET("/worldinfo", func(c *gin.Context) {
			if cfg.ServerAddr == "" {
				c.Status(http.StatusTooEarly)
				return
			}

			data, err := monitor.GetWorldInfoData(cfg, cfg.ServerAddr, rdb)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			c.Header("Content-Type", "application/json")
			c.Writer.Write(data)
		})

		api.POST("/snapshot", func(c *gin.Context) {
			if cfg.ServerAddr == "" {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
				return
			}

			if err := monitor.TakeSnapshot(cfg, cfg.ServerAddr, rdb); err != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": "Server is not running"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		api.POST("/users/change-password", func(c *gin.Context) {
			session := sessions.Default(c)
			userID := session.Get("user_id")

			if err := c.Request.ParseForm(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "reason": "Invalid form data"})
				return
			}

			password := c.Request.Form.Get("password")
			newPassword := c.Request.Form.Get("new-password")

			if !isAllowedPassword(newPassword) {
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "account.password.disallowed")})
				return
			}

			user := User{}
			if err := db.Find(&user, userID).Error; err != nil {
				log.WithError(err).Error("User not found")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "login.error")})
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

			if err := db.Save(user).Error; err != nil {
				log.WithError(err).Error("error updating password")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
			}

			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		api.POST("/users/add", func(c *gin.Context) {
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
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "account.password.disallowed")})
				return
			}

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				log.WithError(err).Error("error registering user")
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": "Error registering user"})
				return
			}

			user := &User{
				Name:          newUsername,
				Password:      string(hashedPassword),
				AddedByUserID: &userID,
				Initialized:   false,
			}

			if err := db.Create(user).Error; err != nil {
				log.WithError(err).Error("error registering user")
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "account.user.exists")})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		api.GET("/hardwarekey", func(c *gin.Context) {
			session := sessions.Default(c)
			userID := session.Get("user_id")

			var credentials []Credential
			if err := db.Where("owner_id = ?", userID).Find(&credentials).Error; err != nil {
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
		})

		api.DELETE("/hardwarekey/:uuid", func(c *gin.Context) {
			session := sessions.Default(c)
			userID := session.Get("user_id")
			keyUuid := c.Param("uuid")

			var credential Credential
			if err := db.Where("owner_id = ? AND uuid = ?", userID, keyUuid).First(&credential).Error; err != nil {
				log.WithError(err).Error("Error fetching credentials")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}

			if err := db.Delete(&credential).Error; err != nil {
				log.WithError(err).Error("Error fetching credentials")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}

			c.JSON(http.StatusNoContent, gin.H{"success": true})
		})

		api.POST("/hardwarekey/begin", func(c *gin.Context) {
			session := sessions.Default(c)
			userID := session.Get("user_id")

			user := User{}
			if err := db.Find(&user, userID).Error; err != nil {
				log.WithError(err).Error("User expected to be found, but didn't")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}

			waUser := webAuthnUser{
				user: user,
			}

			var credentials []Credential
			if err := db.Where("owner_id = ?", userID).Find(&credentials).Error; err != nil {
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

			options, sessionData, err := wauthn.BeginRegistration(&waUser, registerOptions)
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
		})

		api.POST("/hardwarekey/finish", func(c *gin.Context) {
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

			user := User{}
			if err := db.Find(&user, userID).Error; err != nil {
				log.WithError(err).Error("User expected to be found, but didn't")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}

			waUser := webAuthnUser{
				user: user,
			}

			var credentials []Credential
			if err := db.Where("owner_id = ?", userID).Find(&credentials).Error; err != nil {
				log.WithError(err).Error("Error fetching credentials")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}
			for _, c := range credentials {
				waUser.registerCredential(c)
			}

			credData, err := wauthn.FinishRegistration(&waUser, sessionData, c.Request)
			if err != nil {
				log.WithError(err).WithField("info", err.(*protocol.Error).DevInfo).Error("Error registration")
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "hardwarekey.verify.error")})
				return
			}

			keyUuid := uuid.New().String()
			credential := Credential{
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
			if err := db.Model(credential).Select("count(*) > 0").Where("owner_id = ? AND credential_id = ?", userID, credential.CredentialID).Find(&exists).Error; err != nil {
				log.WithError(err).Error("Error fetching public key count")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}

			if exists {
				c.JSON(http.StatusOK, gin.H{"success": false, "reason": L(cfg.ControlPanel.Locale, "hardwarekey.already_registered")})
				return
			}

			if err := db.Create(&credential).Error; err != nil {
				log.WithError(err).Error("Error saving credential")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "reason": "internal server error"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true})
		})

	}

	go func() {
		server.dispatchMonitorEvent(rdb)
	}()

	guessAndHandleCurrentVMState(cfg, gameServer, rdb)

	log.Fatal(r.Run(bindAddr))
}

package handler

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"

	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-contrib/sessions"
	redisess "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type serverState struct {
	selectedWorld string
	machineType   string
}

type Handler struct {
	cfg         *config.Config
	bind        string
	engine      *gin.Engine
	db          *gorm.DB
	redis       *redis.Client
	webauthn    *webauthn.WebAuthn
	serverState serverState
	serverImpl  GameServer
	i18nData    *i18n.Bundle
}

func createDatabaseClient(cfg *config.Config) (*gorm.DB, error) {
	dialector := postgres.Open(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Etc/UTC", cfg.ControlPanel.Postgres.Address, cfg.ControlPanel.Postgres.Port, cfg.ControlPanel.Postgres.User, cfg.ControlPanel.Postgres.Password, cfg.ControlPanel.Postgres.DBName))
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&model.User{})
	db.AutoMigrate(&model.Credential{})

	return db, nil
}

func createRedisClient(cfg *config.Config) *redis.Client {
	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.ControlPanel.Redis.Address,
		Password: cfg.ControlPanel.Redis.Password,
	})
	return redis
}

func prepareDependencies(cfg *config.Config, h *Handler) error {
	db, err := createDatabaseClient(cfg)
	if err != nil {
		return err
	}
	h.db = db

	h.redis = createRedisClient(cfg)

	origin, err := url.Parse(h.cfg.ControlPanel.Origin)
	if err != nil {
		log.WithError(err).Fatal("Failed to parse origin URL")
	}
	wauthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Premises",
		RPID:          origin.Hostname(),
		RPOrigin:      h.cfg.ControlPanel.Origin,
	})
	if err != nil {
		return err
	}
	h.webauthn = wauthn

	if cfg.Debug.Runner {
		h.serverImpl = NewLocalDebugServer(h.cfg, h)
	} else {
		h.serverImpl = NewConohaServer(h.cfg, h)
	}

	return nil
}

func createDataDirIfNeeded(cfg *config.Config) error {
	if cfg.Debug.Env {
		if err := os.MkdirAll("/tmp/premises/gamedata", 0755); err != nil {
			return err
		}
	}
	return nil
}

func setupRoutes(h *Handler) {
	h.setupRootRoutes(h.engine.Group(""))
	h.setupWebauthnLoginRoutes(h.engine.Group("/login/hardwarekey"))
	h.setupApiRoutes(h.engine.Group("/api"))
}

func setupSessions(h *Handler) {
	sessionStore, err := redisess.NewStore(4, "tcp", h.cfg.ControlPanel.Redis.Address, h.cfg.ControlPanel.Redis.Password, []byte(h.cfg.ControlPanel.Secret))
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize Redis store")
	}
	redisess.SetKeyPrefix(sessionStore, "session:")
	h.engine.Use(sessions.Sessions("session", sessionStore))
}

func syncRemoteVMState(cfg *config.Config, gameServer GameServer, rdb *redis.Client, h *Handler) error {
	if !gameServer.VMExists() {
		return nil
	}
	if gameServer.VMRunning() {
		monitorKey, err := rdb.Get(context.TODO(), "monitor-key").Result()
		if err != nil {
			return err
		}
		cfg.MonitorKey = string(monitorKey)

		if gameServer.ImageExists() {
			log.Info("Server seems to be running, but remote image exists")
			gameServer.DeleteImage(rdb)
		}

		gameServer.UpdateDNS(rdb)

		log.Info("Start monitoring server")
		go h.monitorServer(cfg, gameServer, rdb)
	} else {
		if !gameServer.ImageExists() && !gameServer.SaveImage(rdb) {
			return errors.New("Invalid state")
		}
		if !gameServer.DeleteVM() {
			return errors.New("Failed to delete VM")
		}
	}

	return nil
}

func setupTemplates(h *Handler) error {
	template := template.New("")
	templateEntries, err := os.ReadDir("templates")
	if err != nil {
		return err
	}
	for _, ent := range templateEntries {
		data, err := os.ReadFile(filepath.Join("templates", ent.Name()))
		if err != nil {
			log.WithError(err).Fatal("Failed to load templates")
		}
		template.New(ent.Name()).Parse(string(data))
	}
	h.engine.SetHTMLTemplate(template)

	return nil
}

func NewHandler(cfg *config.Config, i18nData *i18n.Bundle, bindAddr string) (*Handler, error) {
	engine := gin.New()
	engine.SetTrustedProxies([]string{"127.0.0.1"})

	if cfg.Debug.Web {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	h := &Handler{
		cfg:      cfg,
		engine:   engine,
		bind:     bindAddr,
		i18nData: i18nData,
	}

	if err := prepareDependencies(cfg, h); err != nil {
		return nil, err
	}

	if err := createDataDirIfNeeded(cfg); err != nil {
		return nil, err
	}

	if err := setupTemplates(h); err != nil {
		return nil, err
	}

	setupSessions(h)

	setupRoutes(h)

	syncRemoteVMState(cfg, h.serverImpl, h.redis, h)

	return h, nil
}

func (h *Handler) Start() error {
	return h.engine.Run(h.bind)
}
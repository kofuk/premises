package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gin-contrib/sessions"
	redisess "github.com/gin-contrib/sessions/redis"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/model"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const htmlContent = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Premises</title>
    <meta name="viewport" content="width=device-width" />
  </head>
  <body>
    <div id="app"></div>
<script type="module">
  import RefreshRuntime from 'http://localhost:5173/@react-refresh'
  RefreshRuntime.injectIntoGlobalHook(window)
  window.$RefreshReg$ = () => {}
  window.$RefreshSig$ = () => (type) => type
  window.__vite_plugin_react_preamble_installed__ = true
</script>
<script type="module" src="http://localhost:5173/@vite/client"></script>
<script type="module" src="http://localhost:5173/main.js"></script>
  </body>
</html>
`

type serverState struct {
	selectedWorld string
	machineType   string
}

type Handler struct {
	cfg           *config.Config
	bind          string
	engine        *gin.Engine
	db            *gorm.DB
	redis         *redis.Client
	webauthn      *webauthn.WebAuthn
	serverState   serverState
	serverImpl    GameServer
	i18nData      *i18n.Bundle
	serverMutex   sync.Mutex
	serverRunning bool
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
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
		},
	})
	if err != nil {
		return err
	}
	h.webauthn = wauthn

	h.serverImpl = NewConohaServer(h.cfg, h)

	return nil
}

func setupRoutes(h *Handler) {
	if h.cfg.Debug.Web {
		log.Info("Proxying vite dev server")

		remoteUrl, err := url.Parse("http://localhost:5173")
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)

		h.engine.NoRoute(func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
		})
	} else {
		h.engine.Use(static.Serve("/", static.LocalFile("gen", false)))
		h.engine.NoRoute(func(c *gin.Context) {
			// Return a HTML file for any page to render the page with React.
			c.Status(http.StatusOK)
			c.Header("Content-Type", "text/html;charset=utf-8")

			entryFile, err := os.Open("gen/index.html")
			if err != nil {
				log.WithError(err).Error("Unable to open index.html")
				c.JSON(http.StatusInternalServerError, gin.H{"success": false})
				return
			}
			defer entryFile.Close()

			io.Copy(c.Writer, entryFile)
		})
	}

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

		h.serverRunning = true
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

func NewHandler(cfg *config.Config, i18nData *i18n.Bundle, bindAddr string) (*Handler, error) {
	if cfg.Debug.Web {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()
	engine.SetTrustedProxies([]string{"127.0.0.1"})

	h := &Handler{
		cfg:           cfg,
		engine:        engine,
		bind:          bindAddr,
		i18nData:      i18nData,
		serverRunning: false,
	}

	if err := prepareDependencies(cfg, h); err != nil {
		return nil, err
	}

	setupSessions(h)

	syncRemoteVMState(cfg, h.serverImpl, h.redis, h)

	setupRoutes(h)

	return h, nil
}

func (h *Handler) Start() error {
	return h.engine.Run(h.bind)
}

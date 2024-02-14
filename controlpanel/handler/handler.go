package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	"github.com/gin-contrib/sessions"
	sessionsRedis "github.com/gin-contrib/sessions/redis"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/backup"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/dns"
	"github.com/kofuk/premises/controlpanel/kvs"
	"github.com/kofuk/premises/controlpanel/mcversions"
	"github.com/kofuk/premises/controlpanel/model/migrations"
	"github.com/kofuk/premises/controlpanel/pollable"
	"github.com/kofuk/premises/controlpanel/streaming"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

type serverState struct {
	selectedWorld string
	machineType   string
}

type Handler struct {
	cfg           *config.Config
	bind          string
	engine        *gin.Engine
	db            *bun.DB
	redis         *redis.Client
	serverState   serverState
	GameServer    *GameServer
	serverMutex   sync.Mutex
	serverRunning bool
	Cacher        kvs.KeyValueStore
	MCVersions    mcversions.MCVersionsService
	Streaming     *streaming.StreamingService
	backup        *backup.BackupService
	runnerAction  *pollable.PollableActionService
	dnsService    *dns.DNSService
}

func createDatabaseClient(cfg *config.Config) (*bun.DB, error) {
	conn := pgdriver.NewConnector(
		pgdriver.WithAddr(fmt.Sprintf("%s:%d", cfg.ControlPanel.Postgres.Address, cfg.ControlPanel.Postgres.Port)),
		pgdriver.WithUser(cfg.ControlPanel.Postgres.User),
		pgdriver.WithPassword(cfg.ControlPanel.Postgres.Password),
		pgdriver.WithDatabase(cfg.ControlPanel.Postgres.DBName),
		pgdriver.WithInsecure(true),
		pgdriver.WithConnParams(map[string]interface{}{
			"TimeZone": "Etc/UTC",
		}),
	)
	db := bun.NewDB(sql.OpenDB(conn), pgdialect.New())
	if cfg.Debug.Web {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	migrator := migrate.NewMigrator(db, migrations.Migrations)

	slog.Info("Initializing bun migration")
	if err := migrator.Init(context.Background()); err != nil {
		return nil, err
	}

	migrator.Lock(context.Background())
	defer migrator.Unlock(context.Background())

	group, err := migrator.Migrate(context.Background())
	if err != nil {
		return nil, err
	}
	if group.IsZero() {
		slog.Info("No new migrations")
	} else {
		slog.Info("Migration completed", slog.String("to", group.String()))
	}

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

	h.GameServer = NewGameServer(h.cfg, h)

	h.backup = backup.New(h.cfg.AWS.AccessKey, h.cfg.AWS.SecretKey, h.cfg.S3.Endpoint, h.cfg.S3.Bucket)

	if h.cfg.Cloudflare.Token != "" {
		cloudflareDNS, err := dns.NewCloudflareDNS(h.cfg.Cloudflare.Token, h.cfg.Cloudflare.ZoneID)
		if err != nil {
			return err
		}

		h.dnsService = dns.New(cloudflareDNS, h.cfg.Cloudflare.GameDomainName)
	}

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
				c.JSON(http.StatusOK, entity.ErrorResponse{
					Success:   false,
					ErrorCode: entity.ErrInternal,
				})
				return
			}
			defer entryFile.Close()

			io.Copy(c.Writer, entryFile)
		})
	}

	h.setupRootRoutes(h.engine.Group(""))
	h.setupApiRoutes(h.engine.Group("/api"))
	h.setupRunnerRoutes(h.engine.Group("/_runner"))
}

func setupSessions(h *Handler) {
	sessionStore, err := sessionsRedis.NewStore(4, "tcp", h.cfg.ControlPanel.Redis.Address, h.cfg.ControlPanel.Redis.Password, []byte(h.cfg.ControlPanel.Secret))
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize Redis store")
	}

	sessionStore.Options(sessions.Options{
		MaxAge:   60 * 60 * 24 * 30,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	sessionsRedis.SetKeyPrefix(sessionStore, "session:")
	h.engine.Use(sessions.Sessions("session", sessionStore))
}

func syncRemoteVMState(cfg *config.Config, gameServer *GameServer, h *Handler) error {
	stdStream := h.Streaming.GetStream(streaming.StandardStream)

	var id string
	if err := h.Cacher.Get(context.Background(), "runner-id:default", &id); err != nil {
		slog.Info("ID for previous runner is not set. Searching for one...", slog.Any("error", err))

		var err error
		id, err = gameServer.FindVM()
		if err != nil {
			slog.Info("No running VM", slog.Any("error", err))

			if err := h.Streaming.PublishEvent(
				context.Background(),
				stdStream,
				streaming.NewStandardMessage(entity.EvStopped, entity.PageLaunch),
			); err != nil {
				log.WithError(err).Error("Failed to write status data to Redis channel")
			}

			return nil
		}
	}

	if gameServer.VMRunning(id) {
		if gameServer.ImageExists() {
			log.Info("Server seems to be running, but remote image exists")
			gameServer.DeleteImage()
		}

		h.serverRunning = true

		slog.Info("Successfully synced runner state")

		return nil
	}

	slog.Info("Recovering system state...")

	if !gameServer.ImageExists() && !gameServer.SaveImage(id) {
		return errors.New("Invalid state")
	}
	if !gameServer.DeleteVM(id) {
		return errors.New("Failed to delete VM")
	}

	slog.Info("Successfully recovered runner state")

	return nil
}

func NewHandler(cfg *config.Config, bindAddr string) (*Handler, error) {
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
		serverRunning: false,
	}

	if err := prepareDependencies(cfg, h); err != nil {
		return nil, err
	}

	kvs := kvs.New(kvs.NewRedis(h.redis))
	h.Cacher = kvs
	h.MCVersions = mcversions.New(kvs)
	h.Streaming = streaming.New(h.redis)
	h.runnerAction = pollable.New(h.redis, "runner-action")

	setupSessions(h)

	syncRemoteVMState(cfg, h.GameServer, h)

	setupRoutes(h)

	return h, nil
}

func (h *Handler) Start() error {
	return h.engine.Run(h.bind)
}

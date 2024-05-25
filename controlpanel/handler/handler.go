package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	"github.com/boj/redistore"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/sessions"
	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/backup"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/kvs"
	"github.com/kofuk/premises/controlpanel/mcversions"
	"github.com/kofuk/premises/controlpanel/pollable"
	"github.com/kofuk/premises/controlpanel/streaming"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/uptrace/bun"
)

type Handler struct {
	cfg           *config.Config
	bind          string
	engine        *echo.Echo
	db            *bun.DB
	redis         *redis.Client
	GameServer    *GameServer
	serverMutex   sync.Mutex
	serverRunning bool
	KVS           kvs.KeyValueStore
	MCVersions    mcversions.MCVersionsService
	Streaming     *streaming.StreamingService
	backup        *backup.BackupService
	runnerAction  *pollable.PollableActionService
}

func prepareDependencies(cfg *config.Config, h *Handler) error {
	h.GameServer = NewGameServer(h.cfg, h)

	h.backup = backup.New(h.cfg.AWS.AccessKey, h.cfg.AWS.SecretKey, h.cfg.S3.Endpoint, h.cfg.S3.Bucket)

	return nil
}

func setupRoutes(h *Handler) {
	if h.cfg.Debug.Web {
		slog.Info("Proxying vite dev server")

		remoteUrl, err := url.Parse("http://localhost:5173")
		if err != nil {
			slog.Error("[BUG] Failed to parse dev server URL", slog.Any("error", err))
			os.Exit(1)
		}

		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)

		h.engine.HTTPErrorHandler = func(err error, c echo.Context) {
			if err != echo.ErrNotFound {
				h.engine.DefaultHTTPErrorHandler(err, c)
				return
			}
			proxy.ServeHTTP(c.Response().Writer, c.Request())
		}
	} else {
		h.engine.Static("/", "gen")
		h.engine.HTTPErrorHandler = func(err error, c echo.Context) {
			if err != echo.ErrNotFound {
				h.engine.DefaultHTTPErrorHandler(err, c)
				return
			}

			// Return a HTML file for any page to render the page with React.

			entryFile, err := os.Open("gen/index.html")
			if err != nil {
				slog.Error("Unable to open index.html", slog.Any("error", err))
				c.JSON(http.StatusOK, web.ErrorResponse{
					Success:   false,
					ErrorCode: entity.ErrInternal,
				})
				return
			}

			c.Stream(http.StatusOK, "text/html;charset=utf-8", entryFile)
		}
	}

	h.setupRootRoutes(h.engine.Group(""))
	h.setupApiRoutes(h.engine.Group("/api"))
	h.setupRunnerRoutes(h.engine.Group("/_runner"))
}

func setupSessions(h *Handler) {
	store, err := redistore.NewRediStore(4, "tcp", h.cfg.ControlPanel.Redis.Address, h.cfg.ControlPanel.Redis.Password, []byte(h.cfg.ControlPanel.Secret))
	if err != nil {
		slog.Error("Failed to initialize Redis session store", slog.Any("error", err))
		os.Exit(1)
	}
	store.Options = &sessions.Options{
		MaxAge:   60 * 60 * 24 * 30,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	store.SetKeyPrefix("session:")
	h.engine.Use(session.Middleware(store))
}

func syncRemoteVMState(ctx context.Context, gameServer *GameServer, h *Handler) error {
	stdStream := h.Streaming.GetStream(streaming.StandardStream)

	var id string
	if err := h.KVS.Get(ctx, "runner-id:default", &id); err != nil {
		slog.Info("ID for previous runner is not set. Searching for one...", slog.Any("error", err))

		var err error
		id, err = gameServer.FindVM(ctx)
		if err != nil {
			slog.Info("No running VM", slog.Any("error", err))

			if err := h.Streaming.PublishEvent(
				ctx,
				stdStream,
				streaming.NewStandardMessage(entity.EventStopped, web.PageLaunch),
			); err != nil {
				slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
			}

			return nil
		}
	}

	if gameServer.VMRunning(ctx, id) {
		h.serverRunning = true

		slog.Info("Successfully synced runner state")

		return nil
	}

	slog.Info("Recovering system state...")

	if !gameServer.DeleteVM(ctx, id) {
		return errors.New("failed to delete VM")
	}

	slog.Info("Successfully recovered runner state")

	return nil
}

func NewHandler(cfg *config.Config, bindAddr string, db *bun.DB, redis *redis.Client) (*Handler, error) {
	engine := echo.New()
	engine.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogMethod: true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			slog.Info("Incoming request",
				slog.String("uri", values.URI),
				slog.String("method", values.Method),
				slog.Int("status", values.Status),
			)
			return nil
		},
	}))
	engine.HideBanner = true
	engine.HidePort = true

	h := &Handler{
		cfg:           cfg,
		engine:        engine,
		db:            db,
		redis:         redis,
		bind:          bindAddr,
		serverRunning: false,
		KVS:           kvs.New(kvs.NewRedis(redis)),
		Streaming:     streaming.New(redis),
		backup:        backup.New(cfg.AWS.AccessKey, cfg.AWS.SecretKey, cfg.S3.Endpoint, cfg.S3.Bucket),
		runnerAction:  pollable.New(redis, "runner-action"),
	}
	h.GameServer = NewGameServer(cfg, h)

	h.MCVersions = mcversions.New(h.KVS)

	setupSessions(h)

	syncRemoteVMState(context.Background(), h.GameServer, h)

	setupRoutes(h)

	return h, nil
}

func (h *Handler) Start() error {
	return h.engine.Start(h.bind)
}

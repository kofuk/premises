package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/boj/redistore"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/sessions"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/mcversions"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/controlpanel/internal/world"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/web"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/uptrace/bun"
)

type Handler struct {
	cfg          *config.Config
	bind         string
	engine       *echo.Echo
	db           *bun.DB
	redis        *redis.Client
	GameServer   *GameServer
	KVS          kvs.KeyValueStore
	MCVersions   mcversions.MCVersionsService
	Streaming    *streaming.StreamingService
	world        *world.WorldService
	runnerAction *longpoll.PollableActionService
}

func setupRoutes(h *Handler) {
	if h.cfg.DevMode {
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
	h.setupRunnerRoutes(h.engine.Group("/_"))
}

func setupSessions(h *Handler) {
	store, err := redistore.NewRediStore(4, "tcp", h.cfg.RedisAddress, h.cfg.RedisPassword, []byte(h.cfg.Secret))
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

func NewHandler(cfg *config.Config, bindAddr string, db *bun.DB, redis *redis.Client, longpoll *longpoll.PollableActionService, kvs kvs.KeyValueStore) (*Handler, error) {
	engine := echo.New()
	engine.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogMethod: true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			slog.Debug("Incoming request",
				slog.String("uri", values.URI),
				slog.String("method", values.Method),
				slog.Int("status", values.Status),
			)
			return nil
		},
	}))
	engine.HideBanner = true
	engine.HidePort = true

	worldService, err := world.New(context.TODO(), cfg.S3Bucket, cfg.S3ForcePathStyle)
	if err != nil {
		return nil, err
	}

	h := &Handler{
		cfg:          cfg,
		engine:       engine,
		db:           db,
		redis:        redis,
		bind:         bindAddr,
		KVS:          kvs,
		Streaming:    streaming.New(redis),
		world:        worldService,
		runnerAction: longpoll,
	}
	h.GameServer = NewGameServer(cfg, h)

	h.MCVersions = mcversions.New(h.KVS)

	setupSessions(h)

	setupRoutes(h)

	return h, nil
}

func (h *Handler) Start() error {
	return h.engine.Start(h.bind)
}

package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/internal/auth"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/mcversions"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/controlpanel/internal/world"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/web"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/kofuk/premises/controlpanel/internal/handler"

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
	authService  *auth.AuthService
}

func setupRoutes(h *Handler) {
	if h.cfg.ServeStatic {
		h.engine.Static("/", h.cfg.StaticDir)
		h.engine.HTTPErrorHandler = func(err error, c echo.Context) {
			if err != echo.ErrNotFound {
				h.engine.DefaultHTTPErrorHandler(err, c)
				return
			}

			// Return a HTML file for any page to render the page with React.

			entryFile, err := os.Open(filepath.Join(h.cfg.StaticDir, "index.html"))
			if err != nil {
				slog.Error("Unable to open index.html", slog.Any("error", err))
				c.JSON(http.StatusNotFound, web.ErrorResponse{
					Success:   false,
					ErrorCode: entity.ErrInternal,
				})
				return
			}
			defer entryFile.Close()

			c.Stream(http.StatusOK, "text/html;charset=utf-8", entryFile)
		}
	}

	h.setupRootRoutes(h.engine.Group(""))
	h.setupApiRoutes(h.engine.Group("/api/v1"))
	h.setupRunnerRoutes(h.engine.Group("/_"))
}

func NewHandler(cfg *config.Config, bindAddr string, db *bun.DB, redis *redis.Client, longpoll *longpoll.PollableActionService, kvs kvs.KeyValueStore) (*Handler, error) {
	engine := echo.New()
	engine.Use(otelecho.Middleware("web", otelecho.WithSkipper(func(c echo.Context) bool {
		path := c.Path()
		if path == "/" {
			return true
		}
		if !strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/_") {
			// Ignore static assets and health endpoint
			return true
		}
		if path == "/_/status" || path == "/_/poll" {
			// Ignore some endpoints which are frequently called by runner.
			return true
		}
		return false
	})))
	engine.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogMethod: true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			var logArgs []any

			span := trace.SpanFromContext(c.Request().Context())
			if span.IsRecording() {
				logArgs = append(logArgs, slog.String("trace_id", span.SpanContext().TraceID().String()))
			}

			slog.Info(fmt.Sprintf("%s %s", values.Method, values.URI), logArgs...)

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
		authService:  auth.New(kvs),
	}
	h.GameServer = NewGameServer(cfg, h)

	h.MCVersions = mcversions.New(h.KVS)

	setupRoutes(h)

	return h, nil
}

func (h *Handler) Start() error {
	return h.engine.Start(h.bind)
}

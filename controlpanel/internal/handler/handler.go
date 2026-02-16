package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kofuk/premises/controlpanel/internal/auth"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/launcher"
	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/mcversions"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/controlpanel/internal/world"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/web"
	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/kofuk/premises/controlpanel/internal/handler"

type Handler struct {
	cfg                 *config.Config
	bind                string
	engine              *echo.Echo
	db                  *bun.DB
	redis               *redis.Client
	KVS                 kvs.KeyValueStore
	MCVersionsService   *mcversions.MCVersionsService
	StreamingService    *streaming.StreamingService
	worldService        *world.WorldService
	runnerActionService *longpoll.LongPollService
	authService         *auth.AuthService
	launcherService     *launcher.LauncherService
}

func setupRoutes(h *Handler) {
	if h.cfg.ServeStatic {
		h.engine.Static("/", h.cfg.StaticDir)
		defaultHttpErrorHandler := echo.DefaultHTTPErrorHandler(false)
		h.engine.HTTPErrorHandler = func(c *echo.Context, err error) {
			if err != echo.ErrNotFound {
				defaultHttpErrorHandler(c, err)
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

func NewHandler(cfg *config.Config, bindAddr string, db *bun.DB, redis *redis.Client, worldService *world.WorldService, longpoll *longpoll.LongPollService, kvs kvs.KeyValueStore, launcher *launcher.LauncherService) (*Handler, error) {
	engine := echo.New()
	engine.Use(echootel.NewMiddlewareWithConfig(echootel.Config{
		ServerName: "web",
		Skipper: func(c *echo.Context) bool {
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
		},
	}))
	engine.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogMethod: true,
		LogStatus: true,
		LogValuesFunc: func(c *echo.Context, values middleware.RequestLoggerValues) error {
			var logArgs []any

			span := trace.SpanFromContext(c.Request().Context())
			if span.IsRecording() {
				logArgs = append(logArgs, slog.String("trace_id", span.SpanContext().TraceID().String()))
			}

			slog.Info(fmt.Sprintf("%s %s", values.Method, values.URI), logArgs...)

			return nil
		},
	}))

	h := &Handler{
		cfg:                 cfg,
		engine:              engine,
		db:                  db,
		redis:               redis,
		bind:                bindAddr,
		KVS:                 kvs,
		StreamingService:    streaming.NewStreamingService(redis),
		worldService:        worldService,
		runnerActionService: longpoll,
		authService:         auth.New(kvs),
		launcherService:     launcher,
	}

	h.MCVersionsService = mcversions.New(h.KVS)

	setupRoutes(h)

	return h, nil
}

func (h *Handler) Start(ctx context.Context) error {
	sc := echo.StartConfig{
		Address:    h.bind,
		HideBanner: true,
		HidePort:   true,
	}
	return sc.Start(ctx, h.engine)
}

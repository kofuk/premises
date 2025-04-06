package launcher

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/launcher/server"
	"github.com/kofuk/premises/controlpanel/internal/startup"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/web"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

type LauncherService struct {
	config    *config.Config
	kvs       kvs.KeyValueStore
	server    server.GameServer
	streaming *streaming.StreamingService
}

func NewLauncherService(config *config.Config, kvs kvs.KeyValueStore, server server.GameServer, streaming *streaming.StreamingService) *LauncherService {
	return &LauncherService{
		config:    config,
		kvs:       kvs,
		server:    server,
		streaming: streaming,
	}
}

func (s *LauncherService) lockInstance(ctx context.Context) error {
	var running bool
	if err := s.kvs.GetSet(ctx, "running", true, -1, &running); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}

		return err
	}

	if running {
		return errors.New("another server is already running")
	}

	return nil
}

func (s *LauncherService) releaseInstance(ctx context.Context) error {
	if err := s.kvs.Del(ctx, "running"); err != nil {
		return fmt.Errorf("failed to unlock instance: %w", err)
	}

	return nil
}

func (s *LauncherService) launchServer(ctx context.Context, config *LaunchConfig) {
	runnerConfig, err := config.ToRunnerConfig(s.config)
	if err != nil {
		slog.Error("Failed to convert config to runner config", slog.Any("error", err))
		s.streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		)
		s.releaseInstance(context.TODO())
		return
	}

	if err := s.kvs.Set(ctx, fmt.Sprintf("runner:%s", runnerConfig.AuthKey), "default", -1); err != nil {
		slog.Error("Failed to save runner id", slog.Any("error", err))

		s.streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerPrepare, true),
		)

		s.releaseInstance(context.TODO())
		return
	}

	s.streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessage(entity.EventCreateRunner, web.PageLoading),
	)

	if s.server.IsAvailable() {
		serverCookie, err := s.server.Start(ctx, runnerConfig, config.MachineType)
		if err != nil {
			slog.Error("Failed to start server", slog.Any("error", err))
			goto failure
		}

		if err := s.kvs.Set(ctx, "runner-id:default", serverCookie, -1); err != nil {
			slog.Error("Failed to set runner ID", slog.Any("error", err))
			return
		}

		s.streaming.PublishEvent(
			ctx,
			streaming.NewStandardMessageWithProgress(entity.EventCreateRunner, 50, web.PageLoading),
		)

		return
	}

	// Startup failed. Manual setup required.

failure:
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	authCode := encoder.EncodeToString(securecookie.GenerateRandomKey(10))

	s.streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessageWithTextData(entity.EventManualSetup, authCode, web.PageManualSetup),
	)

	startupScript, _ := startup.GenerateStartupScript(runnerConfig)
	if err := s.kvs.Set(ctx, fmt.Sprintf("startup:%s", authCode), string(startupScript), time.Hour); err != nil {
		slog.Error("Failed to set startup script", slog.Any("error", err))
		return
	}
}

func (s *LauncherService) Launch(ctx context.Context, config *LaunchConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := s.lockInstance(ctx); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	go s.launchServer(trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx)), config)

	return nil
}

func (h *LauncherService) Clean(ctx context.Context, authKey string) {
	defer h.releaseInstance(context.TODO())

	h.streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessage(entity.EventStopRunner, web.PageLoading),
	)

	var serverCookie server.ServerCookie
	if err := h.kvs.Get(ctx, "runner-id:default", &serverCookie); err != nil || !h.server.IsAvailable() {
		if err == redis.Nil {
			goto out
		}

		h.streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		)
		return
	}

	h.streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessageWithProgress(entity.EventStopRunner, 10, web.PageLoading),
	)

	if !h.server.Stop(ctx, serverCookie) {
		h.streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		)
		return
	}

	h.streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessageWithProgress(entity.EventStopRunner, 60, web.PageLoading),
	)

	if !h.server.Delete(ctx, serverCookie) {
		h.streaming.PublishEvent(
			ctx,
			streaming.NewInfoMessage(entity.InfoErrRunnerStop, true),
		)
		return
	}

out:
	if err := h.kvs.Del(ctx, "runner-id:default", "runner-info:default", "world-info:default", fmt.Sprintf("runner:%s", authKey)); err != nil {
		slog.Error("Failed to unset runner information", slog.Any("error", err))
		return
	}

	h.streaming.PublishEvent(
		ctx,
		streaming.NewStandardMessage(entity.EventStopped, web.PageLaunch),
	)

	if err := h.streaming.ClearSysstat(ctx); err != nil {
		slog.Error("Unable to clear sysstat history", slog.Any("error", err))
	}
}

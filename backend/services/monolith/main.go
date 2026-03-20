package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/kofuk/premises/backend/common/otel"
	"github.com/kofuk/premises/backend/services/common/config"
	"github.com/kofuk/premises/backend/services/common/cron"
	"github.com/kofuk/premises/backend/services/common/db"
	"github.com/kofuk/premises/backend/services/common/db/model/migrations"
	"github.com/kofuk/premises/backend/services/common/handler"
	"github.com/kofuk/premises/backend/services/common/kvs"
	"github.com/kofuk/premises/backend/services/common/launcher"
	"github.com/kofuk/premises/backend/services/common/launcher/server"
	"github.com/kofuk/premises/backend/services/common/longpoll"
	"github.com/kofuk/premises/backend/services/common/proxy"
	"github.com/kofuk/premises/backend/services/common/streaming"
	"github.com/kofuk/premises/backend/services/common/world"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/bun/migrate"
)

func createDatabaseClient(cfg *config.Config) (*bun.DB, error) {
	db, err := db.NewClient(
		db.ConnOptions{
			Host:       cfg.PostgresHost,
			Port:       cfg.PostgresPort,
			User:       cfg.PostgresUser,
			Password:   cfg.PostgresPassword,
			Database:   cfg.PostgresDB,
			SSLMode:    cfg.PostgresSSLMode,
			CACertPath: cfg.PostgresCA,
		},
	)
	if err != nil {
		return nil, err
	}

	db.AddQueryHook(bunotel.NewQueryHook())

	return db, nil
}

func createRedisClient(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Username: cfg.RedisUser,
		Password: cfg.RedisPassword,
	})
	if _, err := redis.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	redisotel.InstrumentTracing(redis)

	return redis, nil
}

func createLongPoll(redis *redis.Client) *longpoll.LongPollService {
	return longpoll.NewLongPollService(redis, "runner-action")
}

func createKVS(redis *redis.Client) kvs.KeyValueStore {
	return kvs.New(kvs.NewRedis(redis))
}

func startWeb(ctx context.Context, cfg *config.Config) {
	if _, err := otel.InitializeTracer(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed to initialize OpenTelemetry", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := createDatabaseClient(cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create database client", slog.Any("error", err))
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrator := migrate.NewMigrator(db, migrations.Migrations)
		if err := migrations.Migrate(ctx, migrator); err != nil {
			slog.ErrorContext(ctx, "Failed to migrate database", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	redis, err := createRedisClient(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create redis client", slog.Any("error", err))
		os.Exit(1)
	}

	kvs := createKVS(redis)

	launcherService := launcher.NewLauncherService(cfg, kvs, server.NewConohaServer(cfg), streaming.NewStreamingService(redis))

	worldService, err := world.New(ctx, cfg.S3Bucket, cfg.S3ForcePathStyle)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create world service", slog.Any("error", err))
		os.Exit(1)
	}

	handler, err := handler.NewHandler(cfg, ":10000", db, redis, worldService, createLongPoll(redis), kvs, launcherService)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to initialize handler", slog.Any("error", err))
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Starting web server...")
	if err := handler.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "Error starting server", slog.Any("error", err))
		os.Exit(1)
	}
}

func startProxy(ctx context.Context, cfg *config.Config) {
	redis, err := createRedisClient(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create redis client", slog.Any("error", err))
		os.Exit(1)
	}

	proxy, err := proxy.NewProxyHandler(cfg, createKVS(redis), createLongPoll(redis))
	if err != nil {
		slog.ErrorContext(ctx, "Error initializing proxy handler", slog.Any("error", err))
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Starting proxy server...")
	if err := proxy.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "Error in proxy handler", slog.Any("error", err))
		os.Exit(1)
	}
}

func startCron(ctx context.Context, config *config.Config) {
	ctx, cancelFn := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancelFn()

	worldService, err := world.New(ctx, config.S3Bucket, config.S3ForcePathStyle)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create world service", slog.Any("error", err))
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Starting cron server...")
	cron := cron.NewCronService(config, worldService)
	if err := cron.Run(ctx); err != nil {
		slog.ErrorContext(ctx, "Error in cron", slog.Any("error", err))
		os.Exit(1)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	if cfg.DevMode {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	})))

	switch cfg.Mode {
	case "web":
		startWeb(ctx, cfg)
	case "proxy":
		startProxy(ctx, cfg)
	case "cron":
		startCron(ctx, cfg)
	default:
		slog.ErrorContext(ctx, fmt.Sprintf("Unknown mode: %s", cfg.Mode))
		os.Exit(1)
	}
}

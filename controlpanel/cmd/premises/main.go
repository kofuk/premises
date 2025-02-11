package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/cron"
	"github.com/kofuk/premises/controlpanel/internal/db"
	"github.com/kofuk/premises/controlpanel/internal/db/model/migrations"
	"github.com/kofuk/premises/controlpanel/internal/handler"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/proxy"
	"github.com/kofuk/premises/internal/otel"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/bun/migrate"
)

func createDatabaseClient(cfg *config.Config) (*bun.DB, error) {
	db := db.NewClient(
		cfg.PostgresAddress,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
	)

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

	redis.AddHook(redisotel.NewTracingHook())

	return redis, nil
}

func createLongPoll(redis *redis.Client) *longpoll.PollableActionService {
	return longpoll.New(redis, "runner-action")
}

func createKVS(redis *redis.Client) kvs.KeyValueStore {
	return kvs.New(kvs.NewRedis(redis))
}

func startWeb(ctx context.Context, cfg *config.Config) {
	if _, err := otel.InitializeTracer(context.Background()); err != nil {
		slog.Error("Failed to initialize OpenTelemetry", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := createDatabaseClient(cfg)
	if err != nil {
		slog.Error("Failed to create database client", slog.Any("error", err))
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrator := migrate.NewMigrator(db, migrations.Migrations)
		if err := migrations.Migrate(context.TODO(), migrator); err != nil {
			slog.Error("Failed to migrate database", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	redis, err := createRedisClient(ctx, cfg)
	if err != nil {
		slog.Error("Failed to create redis client", slog.Any("error", err))
		os.Exit(1)
	}

	handler, err := handler.NewHandler(cfg, ":10000", db, redis, createLongPoll(redis), createKVS(redis))
	if err != nil {
		slog.Error("Failed to initialize handler", slog.Any("error", err))
		os.Exit(1)
	}
	if err := handler.Start(); err != nil {
		slog.Error("Error starting server", slog.Any("error", err))
		os.Exit(1)
	}
}

func startProxy(ctx context.Context, cfg *config.Config) {
	redis, err := createRedisClient(ctx, cfg)
	if err != nil {
		slog.Error("Failed to create redis client", slog.Any("error", err))
		os.Exit(1)
	}

	proxy, err := proxy.NewProxyHandler(cfg, createKVS(redis), createLongPoll(redis))
	if err != nil {
		slog.Error("Error initializing proxy handler", slog.Any("error", err))
		os.Exit(1)
	}

	if err := proxy.Start(context.TODO()); err != nil {
		slog.Error("Error in proxy handler", slog.Any("error", err))
		os.Exit(1)
	}
}

func startCron(ctx context.Context, config *config.Config) {
	ctx, cancelFn := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancelFn()

	cron := cron.NewCronService(config)
	if err := cron.Run(ctx); err != nil {
		slog.Error("Error in cron", slog.Any("error", err))
		os.Exit(1)
	}
}

func main() {
	godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
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

	ctx := context.Background()

	switch cfg.Mode {
	case "web":
		startWeb(ctx, cfg)
	case "proxy":
		startProxy(ctx, cfg)
	case "cron":
		startCron(ctx, cfg)
	default:
		slog.Error(fmt.Sprintf("Unknown mode: %s", cfg.Mode))
		os.Exit(1)
	}
}

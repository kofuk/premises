package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/kofuk/premises/controlpanel/internal/auth"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/cron"
	"github.com/kofuk/premises/controlpanel/internal/db"
	"github.com/kofuk/premises/controlpanel/internal/db/model/migrations"
	"github.com/kofuk/premises/controlpanel/internal/handler"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/launcher"
	"github.com/kofuk/premises/controlpanel/internal/launcher/server/conoha"
	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/controlpanel/internal/mcversions"
	"github.com/kofuk/premises/controlpanel/internal/proxy"
	"github.com/kofuk/premises/controlpanel/internal/services/mcp"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/controlpanel/internal/world"
	"github.com/kofuk/premises/internal/otel"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
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

	kvs := createKVS(redis)

	launcherService := launcher.NewLauncherService(cfg, kvs, conoha.NewConohaServer(cfg), streaming.NewStreamingService(redis))

	worldService, err := world.New(ctx, cfg.S3Bucket, cfg.S3ForcePathStyle)
	if err != nil {
		slog.Error("Failed to create world service", slog.Any("error", err))
		os.Exit(1)
	}

	handler, err := handler.NewHandler(cfg, ":10000", db, redis, worldService, createLongPoll(redis), kvs, launcherService)
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

	worldService, err := world.New(ctx, config.S3Bucket, config.S3ForcePathStyle)
	if err != nil {
		slog.Error("Failed to create world service", slog.Any("error", err))
		os.Exit(1)
	}

	cron := cron.NewCronService(config, worldService)
	if err := cron.Run(ctx); err != nil {
		slog.Error("Error in cron", slog.Any("error", err))
		os.Exit(1)
	}
}

func startMcp(ctx context.Context, config *config.Config) {
	redis, err := createRedisClient(ctx, config)
	if err != nil {
		slog.Error("Failed to create redis client", slog.Any("error", err))
		os.Exit(1)
	}
	db, err := createDatabaseClient(config)
	if err != nil {
		slog.Error("Failed to create database client", slog.Any("error", err))
		os.Exit(1)
	}

	world, err := world.New(ctx, config.S3Bucket, config.S3ForcePathStyle)
	if err != nil {
		slog.Error("Failed to create world service", slog.Any("error", err))
		os.Exit(1)
	}

	kvs := kvs.New(kvs.NewRedis(redis))

	launcherService := launcher.NewLauncherService(config, kvs, conoha.NewConohaServer(config), streaming.NewStreamingService(redis))

	mcVersionsService := mcversions.New(kvs)

	mcp := mcp.NewMCPServer(
		redis,
		db,
		world,
		auth.New(kvs),
		launcherService,
		mcVersionsService,
		config.Operators,
		config.Whitelist,
	)
	if err := mcp.Start(); err != nil {
		slog.Error("Error in MCP server", slog.Any("error", err))
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
	case "mcp":
		startMcp(ctx, cfg)
	default:
		slog.Error(fmt.Sprintf("Unknown mode: %s", cfg.Mode))
		os.Exit(1)
	}
}

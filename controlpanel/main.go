package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/handler"
	"github.com/kofuk/premises/controlpanel/kvs"
	"github.com/kofuk/premises/controlpanel/proxy"
	"github.com/kofuk/premises/internal/db"
	"github.com/kofuk/premises/internal/db/model/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

func createDatabaseClient(cfg *config.Config) (*bun.DB, error) {
	db := db.NewClient(
		cfg.PostgresAddress,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
	)
	if cfg.DebugMode {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	return db, nil
}

func createRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
	})
}

func startWeb(cfg *config.Config) {
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

	redis := createRedisClient(cfg)

	handler, err := handler.NewHandler(cfg, ":8000", db, redis)
	if err != nil {
		slog.Error("Failed to initialize handler", slog.Any("error", err))
		os.Exit(1)
	}
	if err := handler.Start(); err != nil {
		slog.Error("Error starting server", slog.Any("error", err))
		os.Exit(1)
	}
}

func startProxy(cfg *config.Config) {
	proxy := proxy.NewProxyHandler(cfg, kvs.New(kvs.NewRedis(createRedisClient(cfg))))
	if err := proxy.Start(context.TODO()); err != nil {
		slog.Error("Error in proxy handler", slog.Any("error", err))
		os.Exit(1)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		// We haven't initialized slog handler yet, so prepare an ephemeral one to output this.
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
		})).Info("Failed to load .env file. If you want to use real envvars, you can ignore this safely.", slog.Any("error", err))
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	if cfg.DebugMode {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	})))

	switch cfg.Mode {
	case "web":
		startWeb(cfg)
	case "proxy":
		startProxy(cfg)
	default:
		slog.Error(fmt.Sprintf("Unknown mode: %s", cfg.Mode))
		os.Exit(1)
	}
}

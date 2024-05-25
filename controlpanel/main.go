package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/kofuk/premises/common/db"
	"github.com/kofuk/premises/common/db/model/migrations"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/handler"
	"github.com/kofuk/premises/controlpanel/proxy"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

func createDatabaseClient(cfg *config.Config) (*bun.DB, error) {
	db := db.NewClient(
		fmt.Sprintf("%s:%d", cfg.ControlPanel.Postgres.Address, cfg.ControlPanel.Postgres.Port),
		cfg.ControlPanel.Postgres.User,
		cfg.ControlPanel.Postgres.Password,
		cfg.ControlPanel.Postgres.DBName,
	)
	if cfg.Debug.Web {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	return db, nil
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

	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.ControlPanel.Redis.Address,
		Password: cfg.ControlPanel.Redis.Password,
	})

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
	proxy := proxy.NewProxyHandler(cfg.ControlPanel.IconURL)
	if err := proxy.Start(context.Background()); err != nil {
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

	logLevel := slog.LevelInfo
	if os.Getenv("premises_debug_web") == "true" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	})))

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	mode := os.Getenv("PREMISES_MODE")
	switch mode {
	case "web":
		startWeb(cfg)
	case "proxy":
		startProxy(cfg)
	default:
		slog.Error(fmt.Sprintf("Unknown mode: %s", mode))
		os.Exit(1)
	}
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/handler"
	"github.com/kofuk/premises/controlpanel/proxy"
)

func startWeb(cfg *config.Config) {
	handler, err := handler.NewHandler(cfg, ":8000")
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

	if len(os.Args) < 2 {
		slog.Error("Mode not speficied")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	mode := os.Args[1]
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

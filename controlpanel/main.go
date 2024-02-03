package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/handler"
)

func main() {
	log.SetReportCaller(true)

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

	bindAddr := ":8000"
	if len(os.Args) > 1 {
		bindAddr = os.Args[1]
	}

	handler, err := handler.NewHandler(cfg, bindAddr)
	if err != nil {
		slog.Error("Failed to initialize handler", slog.Any("error", err))
		os.Exit(1)
	}
	if err := handler.Start(); err != nil {
		slog.Error("Error starting server", slog.Any("error", err))
		os.Exit(1)
	}
}

package main

import (
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/handler"
)

func main() {
	log.SetReportCaller(true)

	if err := godotenv.Load(); err != nil {
		log.WithError(err).Info("Failed to load .env file. If you want to use real envvars, you can ignore this diag safely.")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load config")
	}

	bindAddr := ":8000"
	if len(os.Args) > 1 {
		bindAddr = os.Args[1]
	}

	handler, err := handler.NewHandler(cfg, bindAddr)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize handler")
	}
	handler.Start()
}

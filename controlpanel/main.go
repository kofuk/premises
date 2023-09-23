package main

import (
	"embed"
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"

	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/handler"
)

//go:embed i18n/*.json
var i18nData embed.FS

func loadI18nBundle() (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	ents, err := i18nData.ReadDir("i18n")
	if err != nil {
		return nil, err
	}
	for _, ent := range ents {
		if _, err := bundle.LoadMessageFileFS(i18nData, "i18n/"+ent.Name()); err != nil {
			return nil, err
		}
	}
	return bundle, nil
}

func main() {
	log.SetReportCaller(true)

	if err := godotenv.Load(); err != nil {
		log.WithError(err).Info("Failed to load .env file. If you want to use real envvars, you can ignore this diag safely.")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load config")
	}

	i18nData, err := loadI18nBundle()
	if err != nil {
		log.WithError(err).Fatal("Failed to load i18n data")
	}

	bindAddr := ":8000"
	if len(os.Args) > 1 {
		bindAddr = os.Args[1]
	}

	handler, err := handler.NewHandler(cfg, i18nData, bindAddr)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize handler")
	}
	handler.Start()
}

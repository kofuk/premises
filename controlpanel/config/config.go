package config

import (
	"github.com/kofuk/premises/controlpanel/backup"
)

type Config struct {
	Debug struct {
		Web bool `env:"web"`
	} `env:"debug"`
	Conoha struct {
		UserName string `env:"username"`
		Password string `env:"password"`
		TenantID string `env:"tenantId"`
		Services struct {
			Identity string `env:"identity"`
			Image    string `env:"image"`
			Compute  string `env:"compute"`
		} `env:"services"`
		NameTag string `env:"nameTag"`
	} `env:"conoha"`
	Cloudflare struct {
		Token          string `env:"token"`
		ZoneID         string `env:"zoneId"`
		GameDomainName string `env:"gameDomain"`
	} `env:"cloudflare"`
	Mega backup.MegaCredentialInfo `env:"mega"`
	Game struct {
		Motd      string   `env:"motd"`
		Operators []string `env:"operators"`
		Whitelist []string `env:"whitelist"`
	} `env:"game"`
	ControlPanel struct {
		Secret   string `env:"secret"`
		Origin   string `env:"origin"`
		Postgres struct {
			Address  string `env:"address"`
			Port     int    `env:"port"`
			User     string `env:"user"`
			Password string `env:"password"`
			DBName   string `env:"dbName"`
		} `env:"postgres"`
		Redis struct {
			Address  string `env:"address"`
			Password string `env:"password"`
		} `env:"redis"`
		Locale          string `env:"locale"`
		AlertWebhookUrl string `env:"alertWebhook"`
	} `env:"controlPanel"`
	MonitorKey string `env:"-"`
	ServerAddr string `env:"-"`
}

type ServerConfig struct {
	Name      string `json:"name"`
	IsVanilla bool   `json:"isVanilla"`
}

func LoadConfig() (*Config, error) {
	var result Config
	if err := loadToStruct("premises", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

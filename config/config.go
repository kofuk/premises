package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Conoha struct {
		UserName string `json:"username"`
		Password string `json:"password"`
		TenantID string `json:"tenantId"`
		Services struct {
			Identity string `json:"identity"`
			Image    string `json:"image"`
			Compute  string `json:"compute"`
		} `json:"services"`
	} `json:"conoha"`
	Cloudflare struct {
		Token          string `json:"token"`
		DomainName     string `json:"domainName"`
		GameDomainName string `json:"gameDomainName"`
	} `json:"cloudflare"`
	Mega struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	} `json:"mega"`
	Game struct {
		Configs []struct {
			Name      string `json:"name"`
			IsVanilla bool   `json:"isVanilla"`
		} `json:"configs"`
		Motd      string   `json:"motd"`
		Operators []string `json:"operators"`
		Whitelist []string `json:"whitelist"`
	} `json:"game"`
	ControlPanel struct {
		Secret        string `json:"secret"`
		AllowedOrigin string `json:"allowedOrigin"`
		Redis         struct {
			Address  string `json:"address"`
			Password string `json:"password"`
		} `json:"redis"`
		Users []struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		} `json:"users"`
	} `json:"controlPanel"`
	Prefix     string
	MonitorKey string
	ServerAddr string
}

func LoadConfig(prefix string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(prefix, "/opt/premises/server_config.json"))
	if err != nil {
		return nil, err
	}

	var result Config
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, err
}

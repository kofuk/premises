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
		DomainName     string `json:"domain_name"`
		GameDomainName string `json:"game_domain_name"`
	} `json:"cloudflare"`
	Prefix string
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

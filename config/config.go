package config

import (
	"log"
	"strings"
)

type Config struct {
	Debug struct {
		Env    bool `env:"env"`
		Web    bool `env:"web"`
		Runner bool `env:"runner"`
	} `env:"debug"`
	Conoha struct {
		UserName string `env:"username"`
		Password string `env:"password"`
		TenantID string `env:"tenant_id"`
		Services struct {
			Identity string `env:"identity"`
			Image    string `env:"image"`
			Compute  string `env:"compute"`
		} `env:"services"`
	} `env:"conoha"`
	Cloudflare struct {
		Token          string `env:"token"`
		DomainName     string `env:"domain"`
		GameDomainName string `env:"game_domain"`
	} `env:"cloudflare"`
	Mega struct {
		Email    string `env:"email"`
		Password string `env:"password"`
	} `env:"mega"`
	Game struct {
		RawConfigs []string `env:"configs"`
		Motd       string   `env:"motd"`
		Operators  []string `env:"operators"`
		Whitelist  []string `env:"whitelist"`
	} `env:"game"`
	ControlPanel struct {
		Secret        string `env:"secret"`
		AllowedOrigin string `env:"allowed_origin"`
		Redis         struct {
			Address  string `env:"address"`
			Password string `env:"password"`
		} `env:"redis"`
		Users []struct {
			Name     string `env:"name"`
			Password string `env:"password"`
		} `env:"users"`
	} `env:"control_panel"`
	Prefix     string `env:"_ignore"`
	MonitorKey string `env:"_ignore"`
	ServerAddr string `env:"_ignore"`
}

type ServerConfig struct {
	Name      string
	IsVanilla bool
}

func (cfg *Config) GetGameConfigs() []ServerConfig {
	result := make([]ServerConfig, 0)
	for _, c := range cfg.Game.RawConfigs {
		fields := strings.Split(c, ":")
		if len(fields) != 2 {
			log.Printf("Env game.configs should consists of 2 fields, but %d field(s) detected; will ignore silently\n", len(fields))
			continue
		}
		result = append(result, ServerConfig{Name: fields[0], IsVanilla: fields[1] == "vanilla"})
	}
	return result
}

func LoadConfig() (*Config, error) {
	var result Config
	if err := loadToStruct("premises", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

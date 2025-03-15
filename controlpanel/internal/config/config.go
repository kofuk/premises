package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	DevMode               bool     `envconfig:"PREMISES_DEV_MODE"`
	ServeStatic           bool     `envconfig:"PREMISES_SERVE_STATIC"`
	StaticDir             string   `envconfig:"PREMISES_STATIC_DIR"`
	Mode                  string   `envconfig:"PREMISES_MODE"`
	ConohaUser            string   `envconfig:"PREMISES_CONOHA_USERNAME"`
	ConohaPassword        string   `envconfig:"PREMISES_CONOHA_PASSWORD"`
	ConohaTenantID        string   `envconfig:"PREMISES_CONOHA_TENANT_ID"`
	ConohaIdentityService string   `envconfig:"PREMISES_CONOHA_IDENTITY_SERVICE"`
	ConohaComputeService  string   `envconfig:"PREMISES_CONOHA_COMPUTE_SERVICE"`
	ConohaVolumeService   string   `envconfig:"PREMISES_CONOHA_VOLUME_SERVICE"`
	ConohaImageService    string   `envconfig:"PREMISES_CONOHA_IMAGE_SERVICE"`
	ConohaNameTag         string   `envconfig:"PREMISES_CONOHA_NAME_TAG"`
	S3Bucket              string   `envconfig:"PREMISES_S3_BUCKET"`
	S3ForcePathStyle      bool     `envconfig:"PREMISES_S3_FORCE_PATH_STYLE"`
	Operators             []string `envconfig:"PREMISES_GAME_OPERATORS"`
	Whitelist             []string `envconfig:"PREMISES_GAME_WHITELIST"`
	Secret                string   `envconfig:"PREMISES_SECRET"`
	Origin                string   `envconfig:"PREMISES_ALLOWED_ORIGIN"`
	PostgresAddress       string   `envconfig:"PREMISES_POSTGRES_ADDRESS"`
	PostgresUser          string   `envconfig:"PREMISES_POSTGRES_USER"`
	PostgresPassword      string   `envconfig:"PREMISES_POSTGRES_PASSWORD"`
	PostgresDB            string   `envconfig:"PREMISES_POSTGRES_DB"`
	RedisUser             string   `envconfig:"PREMISES_REDIS_USER"`
	RedisAddress          string   `envconfig:"PREMISES_REDIS_ADDRESS"`
	RedisPassword         string   `envconfig:"PREMISES_REDIS_PASSWORD"`
	ProxyBind             string   `envconfig:"PREMISES_PROXY_BIND"`
	ProxyBackendAddr      string   `envconfig:"PREMISES_PROXY_BACKEND_ADDRESS"`
	GameDomain            string   `envconfig:"PREMISES_GAME_DOMAIN"`
	IconURL               string   `envconfig:"PREMISES_ICON_URL"`
}

func LoadConfig() (*Config, error) {
	var result Config
	if err := envconfig.Process("", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

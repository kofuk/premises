package config

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
			Network  string `env:"network"`
			Volume   string `env:"volume"`
		} `env:"services"`
		NameTag string `env:"nameTag"`
	} `env:"conoha"`
	S3 struct {
		Endpoint string `env:"endpoint"`
		Bucket   string `env:"bucket"`
	} `env:"s3"`
	AWS struct {
		AccessKey string `env:"accessKey"`
		SecretKey string `env:"secretKey"`
	} `env:"aws"`
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
		ProxyAPI        string `env:"proxyApi"`
		GameDomain      string `env:"gameDomain"`
		IconURL         string `env:"iconUrl"`
		AlertWebhookUrl string `env:"alertWebhook"`
	} `env:"controlPanel"`
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

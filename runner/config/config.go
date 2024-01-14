package config

import (
	"encoding/json"
	"os"

	"github.com/kofuk/premises/common/entity/runner"
)

func Load() (*runner.Config, error) {
	data, err := os.ReadFile("/opt/premises/config.json")
	if err != nil {
		return nil, err
	}

	var config runner.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

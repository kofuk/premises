package config

import (
	"encoding/json"
	"os"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/fs"
)

func Load() (*runner.Config, error) {
	data, err := os.ReadFile(fs.DataPath("config.json"))
	if err != nil {
		return nil, err
	}

	var config runner.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

package conoha

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/kofuk/premises/internal/entity/runner"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig *runner.Config) ([]byte, error) {
	gameConfigData, err := json.Marshal(gameConfig)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(startupScriptTemplate, "\r\n", "\n"), "\n")
	var result bytes.Buffer
	for _, line := range lines {
		switch line {
		case "#__CONFIG_FILE__":
			result.Write(gameConfigData)
		default:
			result.Write([]byte(line))
		}
		result.Write([]byte("\n"))
	}
	return result.Bytes(), nil
}

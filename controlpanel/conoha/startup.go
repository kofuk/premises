package conoha

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/kofuk/premises/common/entity/runner"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig *runner.Config) (string, error) {
	gameConfigData, err := json.Marshal(gameConfig)
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.ReplaceAll(startupScriptTemplate, "\r\n", "\n"), "\n")
	var result strings.Builder
	encoder := base64.NewEncoder(base64.RawStdEncoding, &result)
	for _, line := range lines {
		switch line {
		case "#__CONFIG_FILE__":
			encoder.Write(gameConfigData)
			break
		default:
			encoder.Write([]byte(line))
			break
		}
		encoder.Write([]byte("\n"))
	}
	encoder.Close()
	return result.String(), nil
}

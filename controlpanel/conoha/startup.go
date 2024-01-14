package conoha

import (
	_ "embed"
	"encoding/base64"
	"strings"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig []byte) (string, error) {
	lines := strings.Split(strings.ReplaceAll(startupScriptTemplate, "\r\n", "\n"), "\n")
	var result strings.Builder
	encoder := base64.NewEncoder(base64.RawStdEncoding, &result)
	for _, line := range lines {
		switch line {
		case "#__CONFIG_FILE__":
			encoder.Write(gameConfig)
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

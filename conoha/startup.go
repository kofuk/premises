package conoha

import (
	_ "embed"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/kofuk/premises/config"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig []byte, cfg *config.Config) (string, error) {
	serverCrt, err := os.ReadFile(filepath.Join(cfg.Prefix, "/opt/premises/server.crt"))
	if err != nil {
		return "", err
	}
	serverKey, err := os.ReadFile(filepath.Join(cfg.Prefix, "/opt/premises/server.key"))
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.ReplaceAll(startupScriptTemplate, "\r\n", "\n"), "\n")
	var result strings.Builder
	encoder := base64.NewEncoder(base64.RawStdEncoding, &result)
	for _, line := range lines {
		switch line {
		case "#__CONFIG_FILE__":
			encoder.Write(gameConfig)
			break
		case "#__SERVER_CRT__":
			encoder.Write(serverCrt)
			break
		case "#__SERVER_KEY__":
			encoder.Write(serverKey)
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

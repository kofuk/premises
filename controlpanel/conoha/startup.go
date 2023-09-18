package conoha

import (
	"context"
	_ "embed"
	"encoding/base64"
	"strings"

	"github.com/go-redis/redis/v8"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig []byte, rdb *redis.Client) (string, error) {
	serverCrt, err := rdb.Get(context.Background(), "server-crt").Result()
	if err != nil {
		return "", err
	}
	serverKey, err := rdb.Get(context.Background(), "server-key").Result()
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
			encoder.Write([]byte(serverCrt))
			break
		case "#__SERVER_KEY__":
			encoder.Write([]byte(serverKey))
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

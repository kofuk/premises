package monitor

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
)

func GenerateTLSKey(cfg *config.Config, rdb *redis.Client) error {
	cmd := exec.Command("openssl", "req", "-x509", "-nodes", "-subj", "/C=US", "-addext", "subjectAltName = DNS:*", "-newkey", "rsa:4096", "-keyout", "/tmp/key", "-out", "/tmp/cert")
	if err := cmd.Run(); err != nil {
		return err
	}

	serverKey, err := os.ReadFile("/tmp/key")
	if err != nil {
		return err
	}
	serverCert, err := os.ReadFile("/tmp/cert")
	if err != nil {
		return err
	}

	if _, err := rdb.Set(context.Background(), "server-key", string(serverKey), 0).Result(); err != nil {
		return err
	}
	if _, err := rdb.Set(context.Background(), "server-crt", string(serverCert), 0).Result(); err != nil {
		return err
	}

	os.Remove("/tmp/key")
	os.Remove("/tmp/cert")

	return nil
}

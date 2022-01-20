package monitor

import (
	"os/exec"
	"path/filepath"

	"github.com/kofuk/premises/config"
)

func GenerateTLSKey(cfg *config.Config) error {
	keyout := filepath.Join(cfg.Prefix, "/opt/premises/server.key")
	out := filepath.Join(cfg.Prefix, "/opt/premises/server.crt")
	cmd := exec.Command("openssl", "req", "-x509", "-nodes", "-subj", "/C=US", "-addext", "subjectAltName = DNS:*", "-newkey", "rsa:4096", "-keyout", keyout, "-out", out)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

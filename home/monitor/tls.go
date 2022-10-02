package monitor

import (
	"os/exec"

	"github.com/kofuk/premises/home/config"
)

func GenerateTLSKey(cfg *config.Config) error {
	keyout := cfg.LocatePersist("server.key")
	out := cfg.LocatePersist("server.crt")
	cmd := exec.Command("openssl", "req", "-x509", "-nodes", "-subj", "/C=US", "-addext", "subjectAltName = DNS:*", "-newkey", "rsa:4096", "-keyout", keyout, "-out", out)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

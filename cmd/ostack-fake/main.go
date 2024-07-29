package main

import (
	"log/slog"
	"os"

	"github.com/kofuk/premises/internal/fake/ostack"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))

	ostack, err := ostack.NewOstack(ostack.TenantCredentials("tenantId", "user", "password"), ostack.Token("xxxxxxxxxx"))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := ostack.Start(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

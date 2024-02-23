package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))

	ostack, err := NewOstack(TenantCredentials("tenantId", "user", "password"), Token("xxxxxxxxxx"))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := ostack.Start(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

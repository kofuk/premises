package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/kofuk/premises/internal/fake/ostack"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	ostack, err := ostack.NewOstack(ostack.OstackFakeOptions{
		TenantId: "tenantId",
		User:     "user",
		Password: "password",
		Token:    "xxxxxxxxxx",
	})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	if err := ostack.Start(ctx); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

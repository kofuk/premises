package util

import (
	"context"
	"errors"
	"log/slog"

	"github.com/kofuk/go-queryalternatives"
	"github.com/kofuk/premises/runner/internal/system"
)

func findNewestJavaCommand(ctx context.Context) (string, error) {
	output, err := system.RunWithOutput(ctx, system.DefaultExecutor, "update-alternatives", []string{"--query", "java"})
	if err != nil {
		return "", err
	}

	alternatives, err := queryalternatives.ParseString(output)
	if err != nil {
		return "", err
	} else if alternatives.Best == "" {
		return "", errors.New("no alternatives found")
	}

	return alternatives.Best, nil
}

func FindJavaPath(ctx context.Context) string {
	path, err := findNewestJavaCommand(ctx)
	if err != nil {
		slog.Warn("Error finding java installation. Using the system default", slog.Any("error", err))
		return "java"
	}

	return path
}

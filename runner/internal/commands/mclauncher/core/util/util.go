package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/kofuk/go-queryalternatives"
	"github.com/kofuk/premises/runner/internal/system"
)

func IsJar(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to open input file: %s", err.Error()))
		return false
	}
	defer f.Close()

	buf := make([]byte, 4)
	if _, err := io.ReadFull(f, buf); err != nil {
		slog.Error(fmt.Sprintf("Failed to read file signature: %s", err.Error()))
		return false
	}

	return bytes.Equal([]byte{0x50, 0x4b, 0x03, 0x04}, buf)
}

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

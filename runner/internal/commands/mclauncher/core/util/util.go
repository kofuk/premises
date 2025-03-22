package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

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

func getJavaPathFromInstalledVersion(ctx context.Context, version int) (string, error) {
	output, err := system.CmdOutput(ctx, "update-alternatives", []string{"--list", "java"})
	if err != nil {
		return "", err
	}

	candidates := strings.Split(strings.TrimRight(output, "\r\n"), "\n")
	slog.Debug("Installed java versions", slog.Any("versions", candidates))

	for _, path := range candidates {
		if strings.Contains(path, fmt.Sprintf("-%d-", version)) {
			return path, nil
		}
	}

	return "", errors.New("not found")
}

func FindJavaPath(ctx context.Context, desiredVersion int) string {
	if desiredVersion == 0 {
		slog.Info("Version not specified. Using the system default")
		return "java"
	}

	path, err := getJavaPathFromInstalledVersion(ctx, desiredVersion)
	if err != nil {
		slog.Warn("Error finding java installation. Using the system default", slog.Any("error", err))
		return "java"
	}

	slog.Info("Found java installation matching requested version", slog.String("path", path), slog.Int("desiredVersion", desiredVersion))

	return path
}

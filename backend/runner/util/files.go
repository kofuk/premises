package util

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
)

func IsJar(ctx context.Context, path string) bool {
	f, err := os.Open(path)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to open input file", slog.Any("error", err))
		return false
	}
	defer f.Close()

	buf := make([]byte, 4)
	if _, err := io.ReadFull(f, buf); err != nil {
		slog.ErrorContext(ctx, "Failed to read file signature", slog.Any("error", err))
		return false
	}

	return bytes.Equal([]byte{0x50, 0x4b, 0x03, 0x04}, buf)
}

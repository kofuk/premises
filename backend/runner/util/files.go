package util

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
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

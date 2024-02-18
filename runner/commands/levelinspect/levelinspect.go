package levelinspect

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/kofuk/premises/runner/fs"
)

type Result struct {
	Success       bool   `json:"success"`
	ServerVersion string `json:"serverVersion"`
}

type limitedReader struct {
	r       io.Reader
	limit   int
	current int
}

func (self *limitedReader) Read(p []byte) (int, error) {
	if self.current >= self.limit {
		return 0, errors.New("Read limit reached")
	}

	read, err := self.r.Read(p)
	if err != nil {
		return read, err
	}
	self.current += read
	return read, nil
}

func Run() {
	levelDat, err := os.Open(fs.LocateWorldData("world/level.dat"))
	if err != nil {
		slog.Error("Failed to open level.dat", slog.Any("error", err))
		os.Exit(1)
	}
	defer levelDat.Close()

	gzipReader, err := gzip.NewReader(levelDat)
	if err != nil {
		slog.Error("Failed to decompress level.dat", slog.Any("error", err))
		os.Exit(1)
	}

	reader := &limitedReader{
		r:     gzipReader,
		limit: 4 * 1024 * 1024, // 4 MiB
	}

	_ = reader

	result := Result{
		Success: false,
	}

	json, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal result to JSON", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println(string(json))
}

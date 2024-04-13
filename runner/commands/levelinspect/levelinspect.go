package levelinspect

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/kofuk/premises/common/nbt"
	"github.com/kofuk/premises/runner/fs"
)

type Result struct {
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

type LevelDat struct {
	Data struct {
		Version struct {
			Name string
		}
	}
}

func Run(args []string) {
	levelDatFile, err := os.Open(fs.LocateWorldData("world/level.dat"))
	if err != nil {
		slog.Error("Failed to open level.dat", slog.Any("error", err))
		os.Exit(1)
	}
	defer levelDatFile.Close()

	gzipReader, err := gzip.NewReader(levelDatFile)
	if err != nil {
		slog.Error("Failed to decompress level.dat", slog.Any("error", err))
		os.Exit(1)
	}

	reader := &limitedReader{
		r:     gzipReader,
		limit: 4 * 1024 * 1024, // 4 MiB
	}

	decoder := nbt.NewDecoderWithDepthLimit(reader, 20)
	var levelDat LevelDat
	if err := decoder.Decode(&levelDat); err != nil {
		slog.Error("Failed to parse level.dat", slog.Any("error", err))
		os.Exit(1)
	}

	result := Result{
		ServerVersion: levelDat.Data.Version.Name,
	}

	json, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal result to JSON", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println(string(json))
}

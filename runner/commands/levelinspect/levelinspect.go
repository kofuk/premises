package levelinspect

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/kofuk/premises/common/mc/nbt"
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

func toServerVersionName(name string) string {
	if strings.Index(name, "Pre-Release") >= 0 {
		if match, _ := regexp.Match("^1\\.14(\\.[12])? Pre-Release [1-5]$", []byte(name)); !match {
			// The pre-release version (except for the specific versions) of level.dat stores
			// a different string than the downloadable version name.
			// We will fix this here.
			name = strings.Replace(name, " Pre-Release ", "-pre", 1)
		}
	}

	return name
}

func Run(args []string) int {
	levelDatFile, err := os.Open(fs.DataPath("gamedata/world/level.dat"))
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
		ServerVersion: toServerVersionName(levelDat.Data.Version.Name),
	}

	json, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal result to JSON", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println(string(json))

	return 0
}

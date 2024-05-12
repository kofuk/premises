package game

import (
	"compress/gzip"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kofuk/premises/common/mc/nbt"
	"github.com/kofuk/premises/runner/fs"
)

type limitedReader struct {
	r       io.Reader
	limit   int
	current int
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if r.current >= r.limit {
		return 0, errors.New("Read limit reached")
	}

	read, err := r.r.Read(p)
	if err != nil {
		return read, err
	}
	r.current += read
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
	if strings.Contains(name, "Pre-Release") {
		if match, _ := regexp.Match("^1\\.14(\\.[12])? Pre-Release [1-5]$", []byte(name)); !match {
			// The pre-release version (except for the specific versions) of level.dat stores
			// a different string than the downloadable version name.
			// We will fix this here.
			name = strings.Replace(name, " Pre-Release ", "-pre", 1)
		}
	}

	return name
}

func DetectServerVersion() (string, error) {
	levelDatFile, err := os.Open(fs.DataPath("gamedata/world/level.dat"))
	if err != nil {
		return "", err
	}
	defer levelDatFile.Close()

	gzipReader, err := gzip.NewReader(levelDatFile)
	if err != nil {
		return "", err
	}

	reader := &limitedReader{
		r:     gzipReader,
		limit: 4 * 1024 * 1024, // 4 MiB
	}

	decoder := nbt.NewDecoderWithDepthLimit(reader, 20)
	var levelDat LevelDat
	if err := decoder.Decode(&levelDat); err != nil {
		return "", err
	}

	return toServerVersionName(levelDat.Data.Version.Name), nil
}

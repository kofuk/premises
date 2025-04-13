package leveldat

import (
	"compress/gzip"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kofuk/premises/internal/mc/nbt"
)

type LevelDat struct {
	Data struct {
		Version struct {
			Name string
		}
	}
}

var (
	pre114Pattern = regexp.MustCompile(`^1\.14(\.[12])? Pre-Release [1-5]$`)
)

func CanonicalizeVersionName(name string) string {
	if strings.Contains(name, "Pre-Release") {
		if match := pre114Pattern.MatchString(name); !match {
			// The pre-release version (except for the specific versions) of level.dat stores
			// a different string than the downloadable version name.
			// We will fix this here.
			name = strings.Replace(name, " Pre-Release ", "-pre", 1)
		}
	}

	return name
}

func GetCanonicalServerVersion(levelDatPath string) (string, error) {
	levelDatFile, err := os.Open(levelDatPath)
	if err != nil {
		return "", err
	}
	defer levelDatFile.Close()

	gzipReader, err := gzip.NewReader(levelDatFile)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	reader := io.LimitReader(gzipReader, 4*1024*1024) // 4 MiB limit

	decoder := nbt.NewDecoderWithDepthLimit(reader, 20)
	var levelDat LevelDat
	if err := decoder.Decode(&levelDat); err != nil {
		return "", err
	}

	return CanonicalizeVersionName(levelDat.Data.Version.Name), nil
}

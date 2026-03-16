package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func fetchAndWrite(baseDir string, item ResourceItem, tarWriter *tar.Writer) error {
	if item.ApprovedLicense == "" {
		return fmt.Errorf("approvedLicense is required for %s", item.Destination)
	}

	fetcher := getFetcher(item.Type)
	if fetcher == nil {
		return fmt.Errorf("unsupported resource type: %s", item.Type)
	}

	resp, size, err := fetcher.Fetch(baseDir, item)
	if err != nil {
		return err
	}
	defer resp.Close()

	hash := sha256.New()
	bodyReader := io.TeeReader(resp, hash)

	header := &tar.Header{
		Name: item.Destination,
		Size: int64(size),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, bodyReader); err != nil {
		return err
	}

	if item.Checksum != "" {
		calculatedChecksum := hash.Sum(nil)
		expectedChecksum, err := hex.DecodeString(item.Checksum)
		if err != nil {
			return err
		}

		if !bytes.Equal(calculatedChecksum, expectedChecksum) {
			return fmt.Errorf("checksum mismatch for %s", item.Destination)
		}
	}

	return nil
}

func main() {
	inputFilename := flag.String("input", "resources.json", "Path to the resources definition file")
	outputFilename := flag.String("output", "resources.tar.zst", "Path to the output tar file")
	flag.Parse()

	resources, err := loadConfig(*inputFilename)
	if err != nil {
		panic(err)
	}

	baseDir := filepath.Dir(*inputFilename)

	outputFile, err := os.Create(*outputFilename)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	encoder, err := zstd.NewWriter(outputFile)
	if err != nil {
		panic(err)
	}
	defer encoder.Close()

	tarWriter := tar.NewWriter(encoder)
	defer tarWriter.Close()

	for _, item := range resources {
		if err := fetchAndWrite(baseDir, item, tarWriter); err != nil {
			panic(err)
		}
	}
}

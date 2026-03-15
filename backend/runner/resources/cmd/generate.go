package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type ResourceItem struct {
	Path     string `json:"path"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

func loadResources(filename string) ([]ResourceItem, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var resources []ResourceItem
	if err := json.Unmarshal(content, &resources); err != nil {
		return nil, err
	}

	return resources, nil
}

func downloadAndWrite(item ResourceItem, tarWriter *tar.Writer) error {
	resp, err := http.Get(item.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	hash := sha256.New()
	bodyReader := io.TeeReader(resp.Body, hash)

	header := &tar.Header{
		Name: item.Path,
		Size: resp.ContentLength,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, bodyReader); err != nil {
		return err
	}

	calculatedChecksum := hash.Sum(nil)
	expectedChecksum, err := hex.DecodeString(strings.Replace(item.Checksum, "sha256:", "", 1))
	if err != nil {
		return err
	}

	if !bytes.Equal(calculatedChecksum, expectedChecksum) {
		return fmt.Errorf("checksum mismatch for %s", item.Path)
	}

	return nil
}

func main() {
	inputFilename := flag.String("input", "resources.json", "Path to the resources definition file")
	outputFilename := flag.String("output", "resources.tar.zst", "Path to the output tar file")
	flag.Parse()

	resources, err := loadResources(*inputFilename)
	if err != nil {
		panic(err)
	}

	outputFile, err := os.Create(*outputFilename)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	encoder, err := zstd.NewWriter(outputFile)
	if err != nil {
		panic(err)
	}

	tarWriter := tar.NewWriter(encoder)
	defer tarWriter.Close()

	for _, item := range resources {
		if err := downloadAndWrite(item, tarWriter); err != nil {
			panic(err)
		}
	}
}

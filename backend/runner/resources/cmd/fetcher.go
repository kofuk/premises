package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Fetcher interface {
	Fetch(baseDir string, item ResourceItem) (io.ReadCloser, int, error)
}

func getFetcher(resourceType ResourceType) Fetcher {
	switch resourceType {
	case ResourceTypeRemote:
		return &HTTPFetcher{}
	case ResourceTypeLocal:
		return &LocalFetcher{}
	default:
		return nil
	}
}

type HTTPFetcher struct{}

func (f *HTTPFetcher) Fetch(baseDir string, item ResourceItem) (io.ReadCloser, int, error) {
	resp, err := http.Get(item.Source)
	if err != nil {
		return nil, 0, err
	}
	return resp.Body, int(resp.ContentLength), nil
}

type LocalFetcher struct{}

func (f *LocalFetcher) Fetch(baseDir string, item ResourceItem) (io.ReadCloser, int, error) {
	file, err := os.Open(filepath.Join(baseDir, item.Source))
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	return file, int(info.Size()), nil
}

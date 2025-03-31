package env

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination path_mock.go -package env . PathProvider

import (
	"os"
	"path/filepath"
)

type PathProvider interface {
	GetDataPath(path ...string) string
	GetTempDir() string
}

type defaultPathProvider struct{}

var DefaultPathProvider PathProvider = &defaultPathProvider{}

func (p *defaultPathProvider) GetDataPath(path ...string) string {
	return filepath.Join(BaseDir, filepath.Join(path...))
}

func (p *defaultPathProvider) GetTempDir() string {
	return p.GetDataPath("tmp")
}

func (p *defaultPathProvider) MkdirTemp() (string, error) {
	return os.MkdirTemp(p.GetTempDir(), "premises-temp")
}

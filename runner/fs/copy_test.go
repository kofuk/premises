package fs

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	fileList = []string{
		"file1",
		"file2",
		"a/",
		"a/file1",
		"a/file2",
		"b/",
		"b/file",
		"b/c/",
		"b/c/file",
		"b/c/d/",
		"b/c/d/file",
		"b/c/e/",
		"b/c/e/file",
		"f/",
		"f/g/",
		"f/g/file",
		"h/",
	}
)

func prepareTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "TEST-")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range fileList {
		path := filepath.Join(dir, name)

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}

		if strings.HasSuffix(name, "/") {
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(path, []byte("This is "+name), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	return dir
}

func checkDir(t *testing.T, dir string) {
	var files []string

	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() {
			path += "/"
		}
		files = append(files, path)
		return nil
	})
	assert.NoError(t, err)

	expectedFiles := slices.Clone(fileList)
	slices.Sort(expectedFiles)
	slices.Sort(files)

	assert.Equal(t, expectedFiles, files)

	for _, path := range files {
		if strings.HasSuffix(path, "/") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, path))
		assert.NoError(t, err)
		assert.Equal(t, "This is "+path, string(content))
	}
}

func checkMoved(t *testing.T, dir string) {
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		assert.True(t, d.IsDir())
		return nil
	})
	assert.NoError(t, err)
}

func Test_moveDir_moveMode(t *testing.T) {
	fromDir := prepareTestDir(t)

	toDir, err := os.MkdirTemp("", "TEST-")
	assert.NoError(t, err)

	err = MoveAll(fromDir, toDir)
	assert.NoError(t, err)

	checkDir(t, toDir)
	checkMoved(t, fromDir)
}

func Test_moveDir_copyMode(t *testing.T) {
	fromDir := prepareTestDir(t)

	toDir, err := os.MkdirTemp("", "TEST-")
	assert.NoError(t, err)

	err = CopyAll(fromDir, toDir)
	assert.NoError(t, err)

	checkDir(t, fromDir)
	checkDir(t, toDir)
}

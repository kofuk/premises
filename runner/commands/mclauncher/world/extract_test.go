package world

import (
	"bytes"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	fileList = []string{
		"file",
		"a/",
		"a/b/",
		"a/b/file1",
		"a/b/file2",
		"a/level.dat",
		"a/c/",
		"a/c/file1",
		"a/c/file2",
		"a/d/",
		"a/d/file",
		"file2",
		"b/",
		"b/file",
	}
)

func Test_FileCreator(t *testing.T) {
	outDir, err := os.MkdirTemp("", "TEST-")
	if err != nil {
		t.Fatal(err)
	}
	tmpDir, err := os.MkdirTemp("", "TEST-")
	if err != nil {
		t.Fatal(err)
	}
	c := FileCreator{
		outDir: outDir,
		tmpDir: tmpDir,
	}
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range fileList {
		if strings.HasSuffix(path, "/") {
			continue
		}
		err := c.CreateFile(path, bytes.NewBufferString("This is "+path))
		assert.NoError(t, err)
	}
	err = c.Finalize()
	assert.NoError(t, err)

	var expectedFileList []string
	for _, path := range fileList {
		if !strings.HasPrefix(path, "a/") || path == "a/" {
			continue
		}

		expectedFileList = append(expectedFileList, strings.TrimPrefix(path, "a/"))
	}

	var actualFileList []string
	err = fs.WalkDir(os.DirFS(outDir), ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() {
			path += "/"
		}
		actualFileList = append(actualFileList, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	slices.Sort(expectedFileList)
	slices.Sort(actualFileList)

	assert.Equal(t, expectedFileList, actualFileList)
}

package fs_test

import (
	gofs "io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kofuk/premises/runner/internal/fs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Copy", func() {
	var (
		fromDir string
		toDir   string
	)

	BeforeEach(func() {
		fromDir = GinkgoT().TempDir()
		toDir = GinkgoT().TempDir()

		for _, name := range fileList {
			path := filepath.Join(fromDir, name)

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				Fail("failed to create dir")
			}

			if strings.HasSuffix(name, "/") {
				if err := os.Mkdir(path, 0755); err != nil {
					Fail("failed to create dir")
				}
			} else {
				if err := os.WriteFile(path, []byte("This is "+name), 0644); err != nil {
					Fail("failed to create file")
				}
			}
		}
	})

	It("should handle move mode", func() {
		err := fs.MoveAll(fromDir, toDir)
		Expect(err).To(BeNil())

		checkDir(toDir)
		_, err = os.Stat(fromDir)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("should handle copy mode", func() {
		err := fs.CopyAll(fromDir, toDir)
		Expect(err).NotTo(HaveOccurred())

		checkDir(toDir)
		checkDir(fromDir)
	})
})

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

func checkDir(dir string) {
	var files []string

	err := gofs.WalkDir(os.DirFS(dir), ".", func(path string, d gofs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() {
			path += "/"
		}
		files = append(files, path)
		return nil
	})
	Expect(err).NotTo(HaveOccurred())

	expectedFiles := slices.Clone(fileList)
	slices.Sort(expectedFiles)
	slices.Sort(files)

	Expect(files).To(Equal(expectedFiles))

	for _, path := range files {
		if strings.HasSuffix(path, "/") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, path))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("This is " + path))
	}
}

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FS Suite")
}

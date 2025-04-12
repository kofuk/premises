package service_test

import (
	"bytes"
	"strings"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileCreator", func() {
	var (
		fileList = []string{
			"file",
			"1/",
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
			"d/",
		}
	)

	Describe("CreateFile", func() {
		It("should create files and directories", func() {
			outDir := GinkgoT().TempDir()
			tmpDir := GinkgoT().TempDir()

			c := service.NewFileCreator(outDir, tmpDir)

			for _, path := range fileList {
				if strings.HasSuffix(path, "/") {
					err := c.CreateFile(path, true, nil)
					Expect(err).ShouldNot(HaveOccurred())
					continue
				}
				err := c.CreateFile(path, false, bytes.NewBufferString("This is "+path))
				Expect(err).ShouldNot(HaveOccurred())
			}
			err := c.Finalize()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

package leveldat_test

import (
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/autoversion/leveldat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LevelDat", func() {
	DescribeTable("CanonicalizeVersionName",
		func(input, expected string) {
			result := leveldat.CanonicalizeVersionName(input)
			Expect(result).To(Equal(expected))
		},
		Entry("Stable version", "1.20.4", "1.20.4"),
		Entry("Snapshot version", "24w14a", "24w14a"),
		Entry("Easter egg version 1", "24w14potato", "24w14potato"),
		Entry("Easter egg version 2", "3D Shareware v1.34", "3D Shareware v1.34"),
		Entry("Pre-release version 1", "1.20.5 Pre-Release 2", "1.20.5-pre2"),
		Entry("Pre-release version 2", "1.14.4 Pre-Release 1", "1.14.4-pre1"),
		Entry("Pre-release version (shouldn't be replaced) 1", "1.14.2 Pre-Release 1", "1.14.2 Pre-Release 1"),
		Entry("Pre-release version (shouldn't be replaced) 2", "1.14.1 Pre-Release 1", "1.14.1 Pre-Release 1"),
		Entry("Pre-release version (shouldn't be replaced) 3", "1.14 Pre-Release 1", "1.14 Pre-Release 1"),
	)
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LevelDat Suite")
}

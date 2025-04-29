package system

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("lsbrelease", func() {
	DescribeTable("readDistroFromLsbRelease", func(input string, expected string, expectsError bool) {
		distroName, err := readDistroFromLsbRelease(bytes.NewBuffer([]byte(input)))
		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(distroName).To(Equal(expected))
	},
		Entry("Simple", `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=23.10
DISTRIB_CODENAME=mantic
DISTRIB_DESCRIPTION="Ubuntu Mantic Minotaur (development branch)"
`, "Ubuntu Mantic Minotaur (development branch)", false),
		Entry("Field value with equal", `DISTRIB_DESCRIPTION="hoge=fuga"
`, "hoge=fuga", false),
		Entry("Errnous line", `hoge
DISTRIB_DESCRIPTION="fuga"
`, "fuga", false),
		Entry("Without quote", `DISTRIB_DESCRIPTION=hoge
`, "hoge", false),
		Entry("Unmatched quote", `DISTRIB_DESCRIPTION="hoge
`, "\"hoge", false),
		Entry("No DISTRIB_DESCRIPTION", `DISTRIB_ID=hoge
DISTRIB_RELEASE=1.1
`, "", true),
	)
})

package handler

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Root", func() {
	DescribeTable("isAllowedPassword", func(password string, allowed bool) {
		result := isAllowedPassword(password)
		Expect(result).To(Equal(allowed))
	},
		Entry(
			"8 chars",
			"abcd1234",
			true,
		),
		Entry(
			"7 chars",
			"abcd123",
			false,
		),
		Entry(
			"alphabet only",
			"abcdefgh",
			false,
		),
		Entry(
			"numeric only",
			"12345678",
			false,
		),
	)
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Suite")
}

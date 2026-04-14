package gameconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GameConfig", func() {
	DescribeTable("addToSlice", func(source []string, element string, result []string) {
		actual := addToSlice(source, element)
		Expect(actual).To(Equal(result))
	},
		Entry(
			"should appended",
			[]string{"hoge", "fuga"},
			"piyo",
			[]string{"hoge", "fuga", "piyo"},
		),
		Entry(
			"should not appended",
			[]string{"hoge", "fuga"},
			"fuga",
			[]string{"hoge", "fuga"},
		),
	)

	DescribeTable("isValidLevelType", func(levelType string, isValid bool) {
		actual := isValidLevelType(levelType)
		Expect(actual).To(Equal(isValid))
	},
		Entry(
			"default",
			"default",
			true,
		),
		Entry(
			"flat",
			"flat",
			true,
		),
		Entry(
			"largeBiomes",
			"largeBiomes",
			true,
		),
		Entry(
			"amplified",
			"amplified",
			true,
		),
		Entry(
			"buffet",
			"buffet",
			true,
		),
		Entry(
			"empty string",
			"",
			false,
		),
		Entry(
			"unknown type",
			"hoge",
			false,
		),
	)

	DescribeTable("isValidDifficulty", func(difficulty string, isValid bool) {
		actual := isValidDifficulty(difficulty)
		Expect(actual).To(Equal(isValid))
	},
		Entry(
			"peaceful",
			"peaceful",
			true,
		),
		Entry(
			"easy",
			"easy",
			true,
		),
		Entry(
			"normal",
			"normal",
			true,
		),
		Entry(
			"hard",
			"hard",
			true,
		),
		Entry(
			"empty string",
			"",
			false,
		),
		Entry(
			"unknown type",
			"hoge",
			false,
		),
	)
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GameConfig Suite")
}

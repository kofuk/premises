package world

import (
	"testing"

	"github.com/kofuk/premises/internal/s3wrap"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("World", func() {
	DescribeTable("extractWorldInfoFromKey (normal cases)", func(key, world, name string) {
		actualWorld, actualName, err := extractWorldInfoFromKey(key)
		Expect(err).NotTo(HaveOccurred())
		Expect(actualWorld).To(Equal(world))
		Expect(actualName).To(Equal(name))
	},
		Entry(
			"normal",
			"foo/bar.tar.zst",
			"foo",
			"bar",
		),
		Entry(
			"multibyte",
			"ああ/いい.tar.zst",
			"ああ",
			"いい",
		),
		Entry(
			"zip file",
			"foo/bar.zip",
			"foo",
			"bar",
		),
		Entry(
			".tar.zst file",
			"foo/bar.tar.zst",
			"foo",
			"bar",
		),
		Entry(
			".tar.xz file",
			"foo/bar.tar.xz",
			"foo",
			"bar",
		),
		Entry(
			"unknown extension",
			"foo/bar.pptx",
			"foo",
			"bar.pptx",
		),
	)

	It("should raise error if key is invalid", func() {
		_, _, err := extractWorldInfoFromKey("foo.tar.zst")
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("should group by prefix", func(objs []s3wrap.ObjectMetaData, expected map[string][]s3wrap.ObjectMetaData) {
		out := groupByPrefix(objs)
		Expect(out).To(Equal(expected))
	},
		Entry(
			"Empty slice",
			[]s3wrap.ObjectMetaData{},
			make(map[string][]s3wrap.ObjectMetaData),
		),
		Entry(
			"Different prefixes",
			[]s3wrap.ObjectMetaData{
				{Key: "prefix1/object1"},
				{Key: "prefix1/object2"},
				{Key: "prefix2/object3"},
				{Key: "prefix2/object4"},
			},
			map[string][]s3wrap.ObjectMetaData{
				"prefix1": {{Key: "prefix1/object1"}, {Key: "prefix1/object2"}},
				"prefix2": {{Key: "prefix2/object3"}, {Key: "prefix2/object4"}},
			},
		),
		Entry(
			"Objects without prefixes",
			[]s3wrap.ObjectMetaData{
				{Key: "object"},
			},
			map[string][]s3wrap.ObjectMetaData{},
		),
		Entry(
			"Multiple slashes in keys",
			[]s3wrap.ObjectMetaData{
				{Key: "prefix1/subprefix1/object1"},
				{Key: "prefix1/subprefix2/object2"},
				{Key: "prefix2/subprefix1/object3"},
				{Key: "prefix2/subprefix2/object4"},
			},
			map[string][]s3wrap.ObjectMetaData{
				"prefix1": {
					{Key: "prefix1/subprefix1/object1"},
					{Key: "prefix1/subprefix2/object2"},
				},
				"prefix2": {
					{Key: "prefix2/subprefix1/object3"},
					{Key: "prefix2/subprefix2/object4"},
				},
			},
		),
	)
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "World Suite")
}

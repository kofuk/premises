package world

import (
	"testing"

	"github.com/kofuk/premises/internal/s3wrap"
	"github.com/stretchr/testify/assert"
)

func Test_extractWorldInfoFromKey_success(t *testing.T) {
	testcases := []struct {
		key   string
		world string
		name  string
	}{
		{
			key:   "foo/bar.tar.zst",
			world: "foo",
			name:  "bar",
		},
		{
			key:   "ああ/いい.tar.zst",
			world: "ああ",
			name:  "いい",
		},
		{
			key:   "foo/bar.zip",
			world: "foo",
			name:  "bar",
		},
		{
			key:   "foo/bar.tar.zst",
			world: "foo",
			name:  "bar",
		},
		{
			key:   "foo/bar.tar.xz",
			world: "foo",
			name:  "bar",
		},
		{
			key:   "foo/bar.pptx",
			world: "foo",
			name:  "bar.pptx",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.key, func(t *testing.T) {
			world, name, err := extractWorldInfoFromKey(tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.world, world)
			assert.Equal(t, tt.name, name)
		})
	}
}

func Test_extractWorldInfoFromKey_fail(t *testing.T) {
	_, _, err := extractWorldInfoFromKey("foo.tar.zst")
	assert.Error(t, err, "invalid backup key: foo.tar.zst")
}

func Test_groupByPrefix(t *testing.T) {
	tests := []struct {
		name     string
		objs     []s3wrap.ObjectMetaData
		expected map[string][]s3wrap.ObjectMetaData
	}{{
		name:     "Empty slice",
		objs:     []s3wrap.ObjectMetaData{},
		expected: make(map[string][]s3wrap.ObjectMetaData),
	}, {
		name: "Different prefixes",
		objs: []s3wrap.ObjectMetaData{
			{Key: "prefix1/object1"},
			{Key: "prefix1/object2"},
			{Key: "prefix2/object3"},
			{Key: "prefix2/object4"},
		},
		expected: map[string][]s3wrap.ObjectMetaData{
			"prefix1": {{Key: "prefix1/object1"}, {Key: "prefix1/object2"}},
			"prefix2": {{Key: "prefix2/object3"}, {Key: "prefix2/object4"}},
		},
	}, {
		name: "Objects without prefixes",
		objs: []s3wrap.ObjectMetaData{
			{Key: "object"},
		},
		expected: map[string][]s3wrap.ObjectMetaData{},
	}, {
		name: "Multiple slashes in keys",
		objs: []s3wrap.ObjectMetaData{
			{Key: "prefix1/subprefix1/object1"},
			{Key: "prefix1/subprefix2/object2"},
			{Key: "prefix2/subprefix1/object3"},
			{Key: "prefix2/subprefix2/object4"},
		},
		expected: map[string][]s3wrap.ObjectMetaData{
			"prefix1": {
				{Key: "prefix1/subprefix1/object1"},
				{Key: "prefix1/subprefix2/object2"},
			},
			"prefix2": {
				{Key: "prefix2/subprefix1/object3"},
				{Key: "prefix2/subprefix2/object4"},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupByPrefix(tt.objs)
			assert.Equal(t, tt.expected, got)
		})
	}
}

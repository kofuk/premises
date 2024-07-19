package world

import (
	"testing"

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

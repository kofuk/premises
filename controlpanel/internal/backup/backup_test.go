package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractBackupInfoFromKey_success(t *testing.T) {
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
			world, name, err := extractBackupInfoFromKey(tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.world, world)
			assert.Equal(t, tt.name, name)
		})
	}
}

func Test_extractBackupInfoFromKey_fail(t *testing.T) {
	_, _, err := extractBackupInfoFromKey("foo.tar.zst")
	assert.Error(t, err, "Invalid backup key: foo.tar.zst")
}

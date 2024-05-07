package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_toServerVersionName(t *testing.T) {
	testcase := []struct {
		name    string
		version string
		result  string
	}{
		{
			name:    "Stable",
			version: "1.20.4",
			result:  "1.20.4",
		},
		{
			name:    "Snapshot",
			version: "24w14a",
			result:  "24w14a",
		},
		{
			name:    "Easter eggs 1",
			version: "24w14potato",
			result:  "24w14potato",
		},
		{
			name:    "Easter eggs 2",
			version: "3D Shareware v1.34",
			result:  "3D Shareware v1.34",
		},
		{
			name:    "Pre-release",
			version: "1.20.5 Pre-Release 2",
			result:  "1.20.5-pre2",
		},
		{
			name:    "Pre-release 2",
			version: "1.14.4 Pre-Release 1",
			result:  "1.14.4-pre1",
		},
		{
			name:    "Pre-release (shouldn't be replaced) 1",
			version: "1.14.2 Pre-Release 1",
			result:  "1.14.2 Pre-Release 1",
		},
		{
			name:    "Pre-release (shouldn't be replaced) 2",
			version: "1.14.1 Pre-Release 1",
			result:  "1.14.1 Pre-Release 1",
		},
		{
			name:    "Pre-release (shouldn't be replaced) 3",
			version: "1.14 Pre-Release 1",
			result:  "1.14 Pre-Release 1",
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			result := toServerVersionName(tt.version)
			assert.Equal(t, tt.result, result)
		})
	}
}

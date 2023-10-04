package systemutil

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readDistroFromLsbRelease(t *testing.T) {
	testcases := []struct {
		name           string
		lsbReleaseData string
		shouldBeError  bool
		distroName     string
	}{
		{
			name: "lsb-release from Ubuntu",
			lsbReleaseData: `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=23.10
DISTRIB_CODENAME=mantic
DISTRIB_DESCRIPTION="Ubuntu Mantic Minotaur (development branch)"
`,
			shouldBeError: false,
			distroName:    "Ubuntu Mantic Minotaur (development branch)",
		},
		{
			name: "Field value with equal",
			lsbReleaseData: `DISTRIB_DESCRIPTION="hoge=fuga"
`,
			shouldBeError: false,
			distroName:    "hoge=fuga",
		},
		{
			name: "Errnous line",
			lsbReleaseData: `hoge
DISTRIB_DESCRIPTION="fuga"
`,
			shouldBeError: false,
			distroName:    "fuga",
		},
		{
			name: "Without quote",
			lsbReleaseData: `DISTRIB_DESCRIPTION=hoge
`,
			shouldBeError: false,
			distroName:    "hoge",
		},
		{
			name: "Unmatched quote",
			lsbReleaseData: `DISTRIB_DESCRIPTION="hoge
`,
			shouldBeError: false,
			distroName:    "\"hoge",
		},
		{
			name: "No DISTRIB_DESCRIPTION",
			lsbReleaseData: `DISTRIB_ID=hoge
DISTRIB_RELEASE=1.1
`,
			shouldBeError: true,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			distroName, err := readDistroFromLsbRelease(bytes.NewBuffer([]byte(tt.lsbReleaseData)))
			if tt.shouldBeError {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.distroName, distroName)
		})
	}
}

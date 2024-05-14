package system

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseMeminfoLine(t *testing.T) {
	testcase := []struct {
		name         string
		input        string
		key          string
		value        int
		expectsError bool
	}{
		{
			name:  "Line with `kB` suffix",
			input: "MemTotal:       16283152 kB",
			key:   "MemTotal",
			value: 16673947648,
		},
		{
			name:  "Line without `kB` suffix",
			input: "HugePages_Total:       1",
			key:   "HugePages_Total",
			value: 1,
		},
		{
			name:         "Line without colon",
			input:        "foobar",
			key:          "",
			value:        0,
			expectsError: true,
		},
		{
			name:         "Empty line",
			input:        "",
			key:          "",
			value:        0,
			expectsError: true,
		},
		{
			name:         "Invalid number",
			input:        "MemTotal:   foo kB",
			key:          "MemTotal",
			value:        0,
			expectsError: true,
		},
		{
			name:         "Suffix except for kB",
			input:        "MemTotal:   100 mB",
			key:          "MemTotal",
			value:        0,
			expectsError: true,
		},
		{
			name:         "Empty value",
			input:        "MemTotal:",
			key:          "MemTotal",
			value:        0,
			expectsError: true,
		},
		{
			name:  "Line with newline",
			input: "MemTotal:   10 kB\n",
			key:   "MemTotal",
			value: 10240,
		},
		{
			name:  "Line with newline (CRLF)",
			input: "MemTotal:   10 kB\r\n",
			key:   "MemTotal",
			value: 10240,
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			k, v, err := parseMeminfoLine(tt.input)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.key, k)
			assert.Equal(t, tt.value, v)
		})
	}
}

func Test_readTotalMemoryFromReader(t *testing.T) {
	testcase := []struct {
		name         string
		input        string
		totalMem     int
		expectsError bool
	}{
		{
			name: "Simple",
			input: `MemTotal:       16283152 kB
MemFree:         5143976 kB
MemAvailable:    7294960 kB
Buffers:          297112 kB
`,
			totalMem:     16673947648,
			expectsError: false,
		},
		{
			name: "No MemTotal line",
			input: `MemFree:         5143976 kB
MemAvailable:    7294960 kB
Buffers:          297112 kB
`,
			totalMem:     0,
			expectsError: true,
		},
		{
			name: "Error in non-MemTotal line",
			input: `MemFree:         5143976 kB
foobar
MemTotal:       16283152 kB
Buffers:          297112 kB
`,
			totalMem: 16673947648,
		},
		{
			name: "Error in MemTotal line",
			input: `MemFree:         5143976 kB
MemTotal:       16283152z kB
Buffers:          297112 kB
`,
			totalMem:     0,
			expectsError: true,
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			mem, err := readTotalMemoryFromReader(strings.NewReader(tt.input))

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.totalMem, mem)
		})
	}
}

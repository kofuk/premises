package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readVarInt(t *testing.T) {
	cases := []struct {
		name         string
		input        []byte
		expected     int
		expectsError bool
	}{
		{
			name:         "zero",
			input:        []byte{0x00},
			expected:     0,
			expectsError: false,
		},
		{
			name:         "single byte",
			input:        []byte{0x7F},
			expected:     0x7F,
			expectsError: false,
		},
		{
			name:         "two bytes",
			input:        []byte{0x81, 0x7F}, // 00011 1111 1000 0001
			expected:     0x3F81,
			expectsError: false,
		},
		{
			name:         "Four bytes",
			input:        []byte{0x83, 0x80, 0x80, 0x00},
			expected:     0x00000003,
			expectsError: false,
		},
		{
			name:         "malformed 1",
			input:        []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expectsError: true,
		},
		{
			name:         "malformed 2",
			input:        []byte{0x81, 0x81},
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			value, err := readVarInt(r)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, value)
			}
		})
	}
}

func Test_writeVarInt(t *testing.T) {
	cases := []struct {
		name         string
		input        int
		expected     []byte
		expectsError bool
	}{
		{
			name:         "zero",
			input:        0,
			expected:     []byte{0x00},
			expectsError: false,
		},
		{
			name:         "one byte",
			input:        0x7F,
			expected:     []byte{0x7F},
			expectsError: false,
		},
		{
			name:         "two byte 1",
			input:        0xFF,
			expected:     []byte{0xFF, 0x01},
			expectsError: false,
		},
		{
			name:         "two byte 2",
			input:        0x3FFF,
			expected:     []byte{0xFF, 0x7F},
			expectsError: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			err := writeVarInt(w, tt.input)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, w.Bytes())
			}
		})
	}
}

func Test_readShort(t *testing.T) {
	cases := []struct {
		name         string
		input        []byte
		expected     int
		expectsError bool
	}{
		{
			name:     "normal",
			input:    []byte{0x02, 0x01},
			expected: 513,
		},
		{
			name:         "too short",
			input:        []byte{0x01},
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			result, err := readShort(r)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_readLong(t *testing.T) {
	cases := []struct {
		name         string
		input        []byte
		expected     int
		expectsError bool
	}{
		{
			name:     "normal",
			input:    []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
			expected: 578437695752307201,
		},
		{
			name:         "too short",
			input:        []byte{0x01},
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			result, err := readLong(r)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_writeLong(t *testing.T) {
	cases := []struct {
		name         string
		input        int
		expected     []byte
		expectsError bool
	}{
		{
			name:     "normal",
			input:    578437695752307201,
			expected: []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			err := writeLong(w, tt.input)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, w.Bytes())
			}
		})
	}
}

type readWriteBuffer struct {
	r io.Reader
}

func (b *readWriteBuffer) Read(buf []byte) (int, error) {
	return b.r.Read(buf)
}

func (b *readWriteBuffer) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func Fuzz_Handler(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		{
			h := NewHandler(&readWriteBuffer{
				r: bytes.NewReader(data),
			})
			h.ReadHandshake()
		}
		{
			h := NewHandler(&readWriteBuffer{
				r: bytes.NewReader(data),
			})
			h.ReadStatusRequest()
		}
		{
			h := NewHandler(&readWriteBuffer{
				r: bytes.NewReader(data),
			})
			h.HandlePingPong()
		}
	})
}

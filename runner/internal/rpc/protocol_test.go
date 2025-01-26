package rpc

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type buffer struct {
	rb *bytes.Buffer
	wb *bytes.Buffer
}

func (b *buffer) Read(buf []byte) (int, error) {
	return b.rb.Read(buf)
}

func (b *buffer) Write(buf []byte) (int, error) {
	return b.wb.Write(buf)
}

func (b *buffer) Close() error {
	return nil
}

func Test_readPacket(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		expectsError bool
	}{
		{
			name:         "Normal",
			input:        "Content-Length: 2\r\n\r\n{}",
			expectsError: false,
		},
		{
			name:         "Unknown header",
			input:        "Content-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
			expectsError: false,
		},
		{
			name:         "Missing content length",
			input:        "\r\n{}",
			expectsError: true,
		},
		{
			name:         "Invalid content length",
			input:        "Content-Length: aa\r\n{}",
			expectsError: true,
		},
		{
			name:         "Negative content length",
			input:        "Content-Length: -5\r\n{}",
			expectsError: true,
		},
		{
			name:         "Missing header",
			input:        "{}",
			expectsError: true,
		},
		{
			name:         "Wrong length",
			input:        "Content-Length: 100\r\n\r\n{}",
			expectsError: true,
		},
		{
			name:         "Broken body",
			input:        "Content-Length: 1\r\n\r\n{",
			expectsError: true,
		},
		{
			name:         "Empty request",
			input:        "",
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBufferString(tt.input)
			_, err := readRequest[any](buf)
			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_writePacket(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	err := writePacket(ctx, &buf, "foo")

	assert.NoError(t, err)
	assert.Equal(t, []byte("Content-Length: 5\r\n\r\n\"foo\""), buf.Bytes())
}

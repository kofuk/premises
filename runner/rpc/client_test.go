package rpc

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_handleCall(t *testing.T) {
	cases := []struct {
		name         string
		params       any
		respJSON     string
		result       any
		resultTo     any
		expectsError bool
	}{
		{
			name:     "Normal",
			params:   struct{}{},
			respJSON: `{"jsonrpc":"2.0","id":1,"result":"foo"}`,
			result:   "foo",
			resultTo: "",
		},
		{
			name:         "Incorrect ID",
			params:       struct{}{},
			respJSON:     `{"jsonrpc":"2.0","id":2,"result":"foo"}`,
			expectsError: true,
		},
		{
			name:         "Error response",
			params:       struct{}{},
			respJSON:     `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid request"}}`,
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conn := &buffer{
				rb: bytes.NewBufferString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(tt.respJSON)), tt.respJSON)),
				wb: &bytes.Buffer{},
			}

			err := handleCall(conn, "test", tt.params, &tt.resultTo)

			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.result, tt.resultTo)
			}
		})
	}
}

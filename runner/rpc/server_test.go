package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_handleConnection(t *testing.T) {
	sut := NewServer("")
	sut.RegisterMethod("foo", func(req *AbstractRequest) (any, error) {
		var params struct {
			Arg1 string `json:"arg1"`
		}
		if err := req.Bind(&params); err != nil {
			return nil, err
		}
		return params.Arg1 == "bar", nil
	})
	sut.RegisterMethod("bar", func(req *AbstractRequest) (any, error) {
		assert.Fail(t, "This method should not be called")
		return nil, nil
	})
	body := `{"jsonrpc":"2.0","method":"foo","params":{"arg1":"bar"},"id":1}`
	conn := &buffer{
		rb: bytes.NewBufferString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(body)), body)),
		wb: &bytes.Buffer{},
	}

	err := sut.handleConnection(conn)
	assert.NoError(t, err)

	respBody, err := readPacket(conn.wb)
	assert.NoError(t, err)

	var resp Response[bool]
	err = json.Unmarshal(respBody, &resp)
	assert.NoError(t, err)

	assert.Equal(t, "2.0", resp.Version)
	assert.Equal(t, 1, resp.ID)
	assert.Equal(t, true, resp.Result)
	assert.Equal(t, (*RPCError)(nil), resp.Error)
}

func Test_handleRequest(t *testing.T) {
	cases := []struct {
		name string
		req  *AbstractRequest
		resp *Response[any]
	}{
		{
			name: "Normal",
			req: &AbstractRequest{
				Version: "2.0",
				ID:      1,
				Method:  "normal",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			resp: &Response[any]{
				Version: "2.0",
				ID:      1,
				Result:  "foo",
			},
		},
		{
			name: "Unsupported version",
			req: &AbstractRequest{
				Version: "1.0",
				ID:      1,
				Method:  "foo",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			resp: &Response[any]{
				Version: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    InvalidRequest,
					Message: InvalidRequestMessage,
				},
			},
		},
		{
			name: "Method missing",
			req: &AbstractRequest{
				Version: "2.0",
				ID:      1,
				Method:  "noMethod",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			resp: &Response[any]{
				Version: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    MethodNotFound,
					Message: MethodNotFoundMessage,
				},
			},
		},
		{
			name: "Error in method",
			req: &AbstractRequest{
				Version: "2.0",
				ID:      1,
				Method:  "error",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			resp: &Response[any]{
				Version: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    CallerError,
					Message: ServerErrorMessage,
					Data:    "Error",
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			sut := NewServer("")
			sut.RegisterMethod("normal", func(req *AbstractRequest) (any, error) {
				return "foo", nil
			})
			sut.RegisterMethod("error", func(req *AbstractRequest) (any, error) {
				return nil, errors.New("Error")
			})

			resp := sut.handleRequest(tt.req)
			assert.Equal(t, tt.resp, resp)
		})
	}
}

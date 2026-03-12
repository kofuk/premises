package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	It("should handle connection", func() {
		sut := NewServer("")
		sut.RegisterMethod("foo", func(ctx context.Context, req *AbstractRequest) (any, error) {
			return "foo", nil
		})
		sut.RegisterNotifyMethod("bar", func(ctx context.Context, req *AbstractRequest) error {
			return nil
		})

		body := `{"jsonrpc":"2.0","method":"foo","params":{"arg1":"foo"},"id":1}`
		conn := &buffer{
			rb: bytes.NewBufferString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(body)), body)),
			wb: &bytes.Buffer{},
		}

		err := sut.handleConnection(GinkgoT().Context(), conn)
		Expect(err).NotTo(HaveOccurred())

		respBody, err := readPacket(conn.wb)
		Expect(err).NotTo(HaveOccurred())

		var resp Response[any]
		err = json.Unmarshal(respBody.Body, &resp)
		Expect(err).NotTo(HaveOccurred())

		Expect(resp.Version).To(Equal("2.0"))
		Expect(resp.ID).To(Equal(1))
		Expect(resp.Result).To(Equal("foo"))
		Expect(resp.Error).To(BeNil())
	})

	var (
		reqID int
	)

	BeforeEach(func() {
		reqID = 1
	})

	DescribeTable("handleRequest call", func(req *AbstractRequest, resp *Response[any]) {
		sut := NewServer("")
		sut.RegisterMethod("normal", func(ctx context.Context, req *AbstractRequest) (any, error) {
			return "foo", nil
		})
		sut.RegisterNotifyMethod("normal", func(ctx context.Context, req *AbstractRequest) error {
			Fail("This should not be called")
			return nil
		})
		sut.RegisterMethod("error", func(ctx context.Context, req *AbstractRequest) (any, error) {
			return nil, errors.New("Error")
		})

		actualResp := sut.handleRequest(req)
		Expect(actualResp).To(Equal(resp))
	},
		Entry(
			"Normal",
			&AbstractRequest{
				Version: "2.0",
				ID:      &reqID,
				Method:  "normal",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			&Response[any]{
				Version: "2.0",
				ID:      1,
				Result:  "foo",
			},
		),
		Entry(
			"Unsupported version",
			&AbstractRequest{
				Version: "1.0",
				ID:      &reqID,
				Method:  "foo",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			&Response[any]{
				Version: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    InvalidRequest,
					Message: InvalidRequestMessage,
				},
			},
		),
		Entry(
			"Method missing",
			&AbstractRequest{
				Version: "2.0",
				ID:      &reqID,
				Method:  "noMethod",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			&Response[any]{
				Version: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    MethodNotFound,
					Message: MethodNotFoundMessage,
				},
			},
		),
		Entry(
			"Error in method",
			&AbstractRequest{
				Version: "2.0",
				ID:      &reqID,
				Method:  "error",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
			&Response[any]{
				Version: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    CallerError,
					Message: ServerErrorMessage,
					Data:    "Error",
				},
			},
		),
	)

	DescribeTable("handleRequest notify", func(req *AbstractRequest) {
		sut := NewServer("")
		sut.RegisterMethod("normal", func(ctx context.Context, req *AbstractRequest) (any, error) {
			Fail("This should not be called")
			return "", nil
		})
		sut.RegisterNotifyMethod("normal", func(ctx context.Context, req *AbstractRequest) error {
			return nil
		})
		sut.RegisterNotifyMethod("error", func(ctx context.Context, req *AbstractRequest) error {
			return errors.New("error")
		})

		actualResp := sut.handleRequest(req)
		Expect(actualResp).To(BeNil())
	},
		Entry(
			"Normal",
			&AbstractRequest{
				Version: "2.0",
				Method:  "normal",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
		),
		Entry(
			"Error",
			&AbstractRequest{
				Version: "2.0",
				Method:  "error",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
		),
		Entry(
			"Method missing",
			&AbstractRequest{
				Version: "2.0",
				Method:  "missing",
				Params:  json.RawMessage([]byte(`{"arg1":"foo"}`)),
			},
		),
	)
})

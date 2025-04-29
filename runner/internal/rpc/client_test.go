package rpc

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("Client", func() {
	DescribeTable("handleCall", func(params any, respJSON string, result, resultOut any, expectsError bool) {
		conn := &buffer{
			rb: bytes.NewBufferString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(respJSON)), respJSON)),
			wb: &bytes.Buffer{},
		}

		err := handleCall(GinkgoT().Context(), conn, "test", params, &resultOut)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(resultOut).To(Equal(result))
		}
	},
		Entry("Normal", struct{}{}, `{"jsonrpc":"2.0","id":1,"result":"foo"}`, "foo", "", false),
		Entry("Incorrect ID", struct{}{}, `{"jsonrpc":"2.0","id":2,"result":"foo"}`, nil, nil, true),
		Entry("Error response", struct{}{}, `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid request"}}`, nil, nil, true),
	)

	It("handleNotify", func() {
		conn := &buffer{
			rb: bytes.NewBufferString(""),
			wb: &bytes.Buffer{},
		}

		err := handleNotify(GinkgoT().Context(), conn, "test", struct{}{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("handleCall with ignoreResult", func() {
		respJSON := `{"jsonrpc":"2.0","id":1,"result":"foo"}`
		conn := &buffer{
			rb: bytes.NewBufferString(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(respJSON)), respJSON)),
			wb: &bytes.Buffer{},
		}

		err := handleCall(GinkgoT().Context(), conn, "test", struct{}{}, nil)
		Expect(err).NotTo(HaveOccurred())
	})
})

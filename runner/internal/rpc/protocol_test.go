package rpc

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Protocol", func() {
	DescribeTable("readPacket", func(input string, expectsError bool) {
		buf := bytes.NewBufferString(input)
		_, err := readRequest[any](buf)
		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("Normal", "Content-Length: 2\r\n\r\n{}", false),
		Entry("Unknown header", "Content-Length: 2\r\nContent-Type: application/json\r\n\r\n{}", false),
		Entry("Missing content length", "\r\n{}", true),
		Entry("Invalid content length", "Content-Length: aa\r\n{}", true),
		Entry("Negative content length", "Content-Length: -5\r\n{}", true),
		Entry("Missing header", "{}", true),
		Entry("Wrong length", "Content-Length: 100\r\n\r\n{}", true),
		Entry("Broken body", "Content-Length: 1\r\n\r\n{", true),
		Entry("Empty request", "", true),
	)

	It("writePacket", func() {
		var buf bytes.Buffer
		err := writePacket(GinkgoT().Context(), &buf, "foo")

		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("Content-Length: 5\r\n\r\n\"foo\""))
	})
})

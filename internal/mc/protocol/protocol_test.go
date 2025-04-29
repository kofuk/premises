package protocol

import (
	"bytes"
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Protocol", func() {
	DescribeTable("readVarInt", func(input []byte, expected int, expectsError bool) {
		r := bytes.NewReader(input)
		value, err := readVarInt(r)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(expected))
		}
	},
		Entry(
			"zero",
			[]byte{0x00},
			0,
			false,
		),
		Entry(
			"single byte",
			[]byte{0x7F, 0x01},
			0x7F,
			false,
		),
		Entry(
			"two bytes",
			[]byte{0x81, 0x7F}, // 00011 1111 1000 0001
			0x3F81,
			false,
		),
		Entry(
			"four bytes",
			[]byte{0x83, 0x80, 0x80, 0x00},
			0x00000003,
			false,
		),
		Entry(
			"malformed 1",
			[]byte{0xFF, 0xFF, 0xFF, 0xFF},
			0,
			true,
		),
		Entry(
			"malmalformed 2",
			[]byte{0x81, 0x81},
			0,
			true,
		),
	)

	DescribeTable("writeVarInt", func(input int, expected []byte, expectsError bool) {
		w := bytes.NewBuffer(nil)
		err := writeVarInt(w, input)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Bytes()).To(Equal(expected))
		}
	},
		Entry(
			"zero",
			0,
			[]byte{0x00},
			false,
		),
		Entry(
			"one byte",
			0x7F,
			[]byte{0x7F},
			false,
		),
		Entry(
			"two byte 1",
			0xFF,
			[]byte{0xFF, 0x01},
			false,
		),
		Entry(
			"two byte 2",
			0x3FFF,
			[]byte{0xFF, 0x7F},
			false,
		),
	)

	DescribeTable("writeShort", func(input []byte, expected int, expectsError bool) {
		r := bytes.NewReader(input)
		result, err := readShort(r)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		}
	},
		Entry(
			"normal",
			[]byte{0x02, 0x01},
			513,
			false,
		),
		Entry(
			"too short",
			[]byte{0x01},
			nil,
			true,
		),
	)

	DescribeTable("readLong", func(iput []byte, expected int, expectsError bool) {
		r := bytes.NewReader(iput)
		result, err := readLong(r)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		}
	},
		Entry(
			"normal",
			[]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
			578437695752307201,
			false,
		),
		Entry(
			"too short",
			[]byte{0x01},
			0,
			true,
		),
	)

	DescribeTable("writeLong", func(input int, expected []byte, expectsError bool) {
		w := bytes.NewBuffer(nil)
		err := writeLong(w, input)

		if expectsError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Bytes()).To(Equal(expected))
		}
	},
		Entry(
			"normal",
			578437695752307201,
			[]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
			false,
		),
	)
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Protocol Suite")
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

package nbt

import (
	"bytes"
	_ "embed"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Decoder", func() {
	DescribeTable("primitive types", func(data []byte, out any, checkFn func(value any) bool) {
		decoder := NewDecoder(bytes.NewBuffer(data))
		err := decoder.Decode(out)
		Expect(err).NotTo(HaveOccurred())
		Expect(checkFn(out)).To(BeTrue(), "Result is not matched")
	},
		Entry(
			"should decode TagByte",
			[]byte{0x1, 0x0, 0x2, 'a', 'b', 0x2},
			new(int8),
			func(v any) bool {
				return *v.(*int8) == 2
			},
		),
		Entry(
			"should decode TagShort",
			[]byte{0x2, 0x0, 0x2, 'a', 'b', 0x0, 0x2},
			new(int16),
			func(v any) bool {
				return *v.(*int16) == 2
			},
		),
		Entry(
			"should decode TagInt",
			[]byte{0x3, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x2},
			new(int32),
			func(v any) bool {
				return *v.(*int32) == 2
			},
		),
		Entry(
			"should decode TagLong",
			[]byte{0x4, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			new(int64),
			func(v any) bool {
				return *v.(*int64) == 2
			},
		),
		Entry(
			"should decode TagFloat",
			[]byte{0x5, 0x0, 0x2, 'a', 'b', 0x40, 0x20, 0x0, 0x0},
			new(float32),
			func(v any) bool {
				return *v.(*float32) == 2.5
			},
		),
		Entry(
			"should decode TagDouble",
			[]byte{0x6, 0x0, 0x2, 'a', 'b', 0x40, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			new(float64),
			func(v any) bool {
				return *v.(*float64) == 2.5
			},
		),
		Entry(
			"should decode TagByteArray",
			[]byte{0x7, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4},
			new([]byte),
			func(v any) bool {
				want := []byte{0x1, 0x2, 0x3, 0x4}
				for i, e := range *v.(*[]byte) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		),
		Entry(
			"should decode TagString",
			[]byte{0x8, 0x0, 0x2, 'a', 'b', 0x0, 0x4, 'a', 'b', 'c', 'd'},
			new(string),
			func(v any) bool {
				return *v.(*string) == "abcd"
			},
		),
		Entry(
			"should decode TagList",
			[]byte{0x9, 0x0, 0x2, 'a', 'b', 0x2, 0x0, 0x0, 0x0, 0x2, 0x0, 0x1, 0x0, 0x2},
			new([]int16),
			func(v any) bool {
				want := []int16{1, 2}
				for i, e := range *v.(*[]int16) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		),
		Entry(
			"should decode empty list of TAG_End",
			[]byte{0x9, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x0, 0x0},
			new([]int8),
			func(v any) bool {
				return len(*v.(*[]int8)) == 0
			},
		),
		Entry(
			"should decode TagIntArray",
			[]byte{0xB, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2},
			new([]int32),
			func(v any) bool {
				want := []int32{1, 2}
				for i, e := range *v.(*[]int32) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		),
		Entry(
			"should decode TagLongArray",
			[]byte{0xC, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			new([]int64),
			func(v any) bool {
				want := []int64{1, 2}
				for i, e := range *v.(*[]int64) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		),
	)

	It("should decode data into a struct", func() {
		type s struct {
			A int16 `nbt:"A"`
			B int32 `nbt:"Q"`
			C []byte
			D struct {
				E int16
				F struct {
					H int16
				}
			}
		}

		data := []byte{
			0xA, 0x0, 0x0, // TAG_Compound() header
			0x2, 0x0, 0x1, 'A', // TAG_Short(A) header
			0x0, 0x1, // 1
			0x3, 0x0, 0x1, 'Q', // TAG_Int(Q) header
			0x0, 0x0, 0x0, 0x2, // 2
			0x7, 0x0, 0x1, 'C', // TAG_ByteArray(C) header
			0x0, 0x0, 0x0, 0x2, 0x1, 0x2, // len=2, [1, 2]
			0xA, 0x0, 0x1, 'D', // TAG_Compound(D) header
			0x2, 0x0, 0x1, 'E', // TAG_Short(E) header
			0x0, 0x3, // 3
			0xA, 0x0, 0x1, 'F', // TAG_Compound(F) header
			0x2, 0x0, 0x1, 'G', // TAG_Short(G) header
			0x0, 0x4, // 4
			0x2, 0x0, 0x1, 'H', // TAG_Short(H) header
			0x0, 0x5, // 5
			0x0, // TAG_End
			0x0, // TAG_End
			0x0, // TAG_End
		}

		decoder := NewDecoder(bytes.NewBuffer(data))
		var out s
		err := decoder.Decode(&out)
		Expect(err).NotTo(HaveOccurred())

		e := s{}
		e.A = 1
		e.B = 2
		e.C = []byte{1, 2}
		e.D.E = 3
		e.D.F.H = 5
		Expect(e).To(Equal(out))
	})

	DescribeTable("should raise error when decoding invalid data", func(data []byte) {
		decoder := NewDecoder(bytes.NewBuffer(data))
		var a struct{}
		err := decoder.Decode(&a)
		Expect(err).To(HaveOccurred())
	},
		Entry(
			"EOF in name length",
			[]byte{0x3, 0x0},
		),
		Entry(
			"EOF in name",
			[]byte{0x3, 0x0, 0x2, 'a'},
		),
		Entry(
			"EOF in name",
			[]byte{0x3, 0x0, 0x2, 'a'},
		),
		Entry(
			"TAG_End without TAG_Compound",
			[]byte{0x0, 0x0, 0x1, 'a'},
		),
		Entry(
			"Invalid type of tag",
			[]byte{0xFF, 0x0, 0x1, 'a'},
		),
		Entry(
			"Empty",
			[]byte{},
		),
		Entry(
			"Missing Byte payload",
			[]byte{0x1, 0x0, 0x0},
		),
		Entry(
			"Missing Short payload",
			[]byte{0x2, 0x0, 0x0},
		),
		Entry(
			"Missing Long payload",
			[]byte{0x4, 0x0, 0x0},
		),
		Entry(
			"Missing Float payload",
			[]byte{0x5, 0x0, 0x0},
		),
		Entry(
			"Missing Double payload",
			[]byte{0x6, 0x0, 0x0},
		),
		Entry(
			"Missing ByteArray length",
			[]byte{0x7, 0x0, 0x0},
		),
		Entry(
			"Not enough ByteArray members",
			[]byte{0x7, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1},
		),
		Entry(
			"Missing String length",
			[]byte{0x8, 0x0, 0x0},
		),
		Entry(
			"Not enough String length",
			[]byte{0x8, 0x0, 0x0, 0x0, 0x2, 'A'},
		),
		Entry(
			"Missing List type",
			[]byte{0x9, 0x0, 0x0},
		),
		Entry(
			"Invalid List type",
			[]byte{0x9, 0x0, 0x0, 0xFF, 0x0, 0x0, 0x0, 0x1},
		),
		Entry(
			"Missing List length",
			[]byte{0x9, 0x0, 0x0, 0x1},
		),
		Entry(
			"Missing End tag",
			[]byte{0xA, 0x0, 0x0},
		),
		Entry(
			"Missing Compound tag name",
			[]byte{0xA, 0x0, 0x0, 0x1, 0x0, 0x1},
		),
		Entry(
			"Invalid tag in Compound",
			[]byte{0xA, 0x0, 0x0, 0xFF, 0x0, 0x0},
		),
		Entry(
			"Missing IntArray length",
			[]byte{0xB, 0x0, 0x0},
		),
		Entry(
			"Invalid IntArray member",
			[]byte{0xB, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1},
		),
		Entry(
			"Missing LongArray length",
			[]byte{0xC, 0x0, 0x0},
		),
		Entry(
			"Invalid LongArray member",
			[]byte{0xC, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1},
		),
	)

	It("should raise error when decoding into non-pointer data", func() {
		decoder := NewDecoder(bytes.NewBuffer([]byte{0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}))
		var out int32
		err := decoder.Decode(out)
		Expect(err).To(HaveOccurred())
	})

	It("should not raise error when using depth limit and limit is not reached", func() {
		decoder := NewDecoderWithDepthLimit(bytes.NewBuffer([]byte{0xA, 0x0, 0x0, 0x1, 0x0, 0x1, 'A', 0x1, 0x0}), 1)
		var out struct{}
		err := decoder.Decode(&out)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should raise error when using depth limit and limit is reached", func() {
		decoder := NewDecoderWithDepthLimit(bytes.NewBuffer([]byte{
			0xA, 0x0, 0x0, // TAG_Compound()
			0xA, 0x0, 0x1, 'B', // TAG_Compound(B)
			0x1, 0x0, 0x1, 'A', // TAG_Byte(A)
			0x1, // 1
			0x0, // TAG_End
			0x0, // TAG_End
		}), 1)
		var out struct{}
		err := decoder.Decode(&out)
		Expect(err).To(HaveOccurred())
	})

	It("should dispose list node in input data if input struct does not contain the key", func() {
		decoder := NewDecoder(bytes.NewBuffer([]byte{
			0xA, 0x0, 0x0, // TAG_Compound()
			0x9, 0x0, 0x1, 'A', // TAG_List(A)
			0x1,                // type=TAG_Byte
			0x0, 0x0, 0x0, 0x2, // len=2
			0x1, 0x2, // [1, 2]
			0x0, // TAG_End
		}))
		var out struct{}
		err := decoder.Decode(&out)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should raise error when decoding struct to non-struct pointer", func() {
		decoder := NewDecoder(bytes.NewBuffer([]byte{
			0xA, 0x0, 0x0, // TAG_Compound()
			0x0, // TAG_End
		}))
		var out int16
		err := decoder.Decode(&out)
		Expect(err).To(HaveOccurred())
	})

	It("should correctly decode real level.dat", func() {
		decoder := NewDecoder(bytes.NewBuffer(levelDat))
		var out struct {
			Data struct {
				WorldGenSettings struct {
					Seed int `nbt:"seed"`
				}
				Version struct {
					Name string
				}
				LevelName string
			}
		}
		err := decoder.Decode(&out)
		Expect(err).NotTo(HaveOccurred())
		Expect(out.Data.WorldGenSettings.Seed).To(Equal(3416194518646519871))
		Expect(out.Data.Version.Name).To(Equal("1.20.4"))
		Expect(out.Data.LevelName).To(Equal("sample world"))
	})
})

//go:embed level.nbt
var levelDat []byte

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NBT Suite")
}

func Fuzz_Decode(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var a struct{}
		dec := NewDecoder(bytes.NewBuffer(data))
		_ = dec.Decode(&a)

		var b []int32
		dec = NewDecoder(bytes.NewBuffer(data))
		_ = dec.Decode(&b)

		var c []int
		dec = NewDecoder(bytes.NewBuffer(data))
		_ = dec.Decode(&c)

		var d string
		dec = NewDecoder(bytes.NewBuffer(data))
		_ = dec.Decode(&d)

		var e int
		dec = NewDecoder(bytes.NewBuffer(data))
		_ = dec.Decode(&e)
		// XXX  For now, We'll check that Decode don't panic.
	})
}

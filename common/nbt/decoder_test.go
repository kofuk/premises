package nbt

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecoder_primitiveTypes(t *testing.T) {
	testcase := []struct {
		name string
		data []byte
		to   any
		ok   func(any) bool
	}{
		{
			name: "TagByte",
			data: []byte{0x1, 0x0, 0x2, 'a', 'b', 0x2},
			to:   new(int8),
			ok: func(v any) bool {
				return *v.(*int8) == 2
			},
		},
		{
			name: "TagShort",
			data: []byte{0x2, 0x0, 0x2, 'a', 'b', 0x0, 0x2},
			to:   new(int16),
			ok: func(v any) bool {
				return *v.(*int16) == 2
			},
		},
		{
			name: "TagInt",
			data: []byte{0x3, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x2},
			to:   new(int32),
			ok: func(v any) bool {
				return *v.(*int32) == 2
			},
		},
		{
			name: "TagLong",
			data: []byte{0x4, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			to:   new(int64),
			ok: func(v any) bool {
				return *v.(*int64) == 2
			},
		},
		{
			name: "TagFloat",
			data: []byte{0x5, 0x0, 0x2, 'a', 'b', 0x40, 0x20, 0x0, 0x0},
			to:   new(float32),
			ok: func(v any) bool {
				return *v.(*float32) == 2.5
			},
		},
		{
			name: "TagDouble",
			data: []byte{0x6, 0x0, 0x2, 'a', 'b', 0x40, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			to:   new(float64),
			ok: func(v any) bool {
				return *v.(*float64) == 2.5
			},
		},
		{
			name: "TagByteArray",
			data: []byte{0x7, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4},
			to:   new([]byte),
			ok: func(v any) bool {
				want := []byte{0x1, 0x2, 0x3, 0x4}
				for i, e := range *v.(*[]byte) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		},
		{
			name: "TagString",
			data: []byte{0x8, 0x0, 0x2, 'a', 'b', 0x0, 0x4, 'a', 'b', 'c', 'd'},
			to:   new(string),
			ok: func(v any) bool {
				return *v.(*string) == "abcd"
			},
		},
		{
			name: "TagList",
			data: []byte{0x9, 0x0, 0x2, 'a', 'b', 0x2, 0x0, 0x0, 0x0, 0x2, 0x0, 0x1, 0x0, 0x2},
			to:   new([]int16),
			ok: func(v any) bool {
				want := []int16{1, 2}
				for i, e := range *v.(*[]int16) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		},
		{
			name: "TagList (empty list of TAG_End)",
			data: []byte{0x9, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x0, 0x0},
			to:   new([]int8),
			ok: func(v any) bool {
				return len(*v.(*[]int8)) == 0
			},
		},
		{
			name: "TagIntArray",
			data: []byte{0xB, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x2},
			to:   new([]int32),
			ok: func(v any) bool {
				want := []int32{1, 2}
				for i, e := range *v.(*[]int32) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		},
		{
			name: "TagLongArray",
			data: []byte{0xC, 0x0, 0x2, 'a', 'b', 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			to:   new([]int64),
			ok: func(v any) bool {
				want := []int64{1, 2}
				for i, e := range *v.(*[]int64) {
					if want[i] != e {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder(bytes.NewBuffer(tt.data))
			err := decoder.Decode(tt.to)
			assert.NoError(t, err)
			if !tt.ok(tt.to) {
				assert.Fail(t, "Result is not matched")
			}
		})
	}
}

func TestDecode_Struct(t *testing.T) {
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
	var to s
	err := decoder.Decode(&to)
	assert.NoError(t, err)

	e := s{}
	e.A = 1
	e.B = 2
	e.C = []byte{1, 2}
	e.D.E = 3
	e.D.F.H = 5
	assert.Equal(t, e, to)
}

func TestDecode_DataError(t *testing.T) {
	testcase := []struct {
		name string
		data []byte
	}{
		{
			name: "EOF in name length",
			data: []byte{0x3, 0x0},
		},
		{
			name: "EOF in name",
			data: []byte{0x3, 0x0, 0x2, 'a'},
		},
		{
			name: "TAG_End without TAG_Compound",
			data: []byte{0x0, 0x0, 0x1, 'a'},
		},
		{
			name: "Invalid type of tag",
			data: []byte{0xFF, 0x0, 0x1, 'a'},
		},
		{
			name: "Empty",
			data: []byte{},
		},
		{
			name: "Missing Byte payload",
			data: []byte{0x1, 0x0, 0x0},
		},
		{
			name: "Missing Short payload",
			data: []byte{0x2, 0x0, 0x0},
		},
		{
			name: "Missing Long payload",
			data: []byte{0x4, 0x0, 0x0},
		},
		{
			name: "Missing Float payload",
			data: []byte{0x5, 0x0, 0x0},
		},
		{
			name: "Missing Double payload",
			data: []byte{0x6, 0x0, 0x0},
		},
		{
			name: "Missing ByteArray length",
			data: []byte{0x7, 0x0, 0x0},
		},
		{
			name: "Not enough ByteArray members",
			data: []byte{0x7, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1},
		},
		{
			name: "Missing String length",
			data: []byte{0x8, 0x0, 0x0},
		},
		{
			name: "Not enough String length",
			data: []byte{0x8, 0x0, 0x0, 0x0, 0x2, 'A'},
		},
		{
			name: "Missing List type",
			data: []byte{0x9, 0x0, 0x0},
		},
		{
			name: "Invalid List type",
			data: []byte{0x9, 0x0, 0x0, 0xFF, 0x0, 0x0, 0x0, 0x1},
		},
		{
			name: "Missing List length",
			data: []byte{0x9, 0x0, 0x0, 0x1},
		},
		{
			name: "Missing End tag",
			data: []byte{0xA, 0x0, 0x0},
		},
		{
			name: "Missing Compound tag name",
			data: []byte{0xA, 0x0, 0x0, 0x1, 0x0, 0x1},
		},
		{
			name: "Invalid tag in Compound",
			data: []byte{0xA, 0x0, 0x0, 0xFF, 0x0, 0x0},
		},
		{
			name: "Missing IntArray length",
			data: []byte{0xB, 0x0, 0x0},
		},
		{
			name: "Invalid IntArray member",
			data: []byte{0xB, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1},
		},
		{
			name: "Missing LongArray length",
			data: []byte{0xC, 0x0, 0x0},
		},
		{
			name: "Invalid LongArray member",
			data: []byte{0xC, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x1},
		},
	}
	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder(bytes.NewBuffer(tt.data))
			var a struct{}
			err := decoder.Decode(&a)
			assert.Error(t, err)
		})
	}
}

func TestDecode_NonPointer(t *testing.T) {
	decoder := NewDecoder(bytes.NewBuffer([]byte{0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}))
	var a int32
	err := decoder.Decode(a)
	if err == nil {
		t.Fatal("Error should not be nil")
	}
}

func TestDecode_DepthLimit_NotReached(t *testing.T) {
	decoder := NewDecoderWithDepthLimit(bytes.NewBuffer([]byte{0xA, 0x0, 0x0, 0x1, 0x0, 0x1, 'A', 0x1, 0x0}), 1)
	var a struct{}
	err := decoder.Decode(&a)
	assert.NoError(t, err)
}

func TestDecode_DepthLimit_Reached(t *testing.T) {
	decoder := NewDecoderWithDepthLimit(bytes.NewBuffer([]byte{
		0xA, 0x0, 0x0, // TAG_Compound()
		0xA, 0x0, 0x1, 'B', // TAG_Compound(B)
		0x1, 0x0, 0x1, 'A', // TAG_Byte(A)
		0x1, // 1
		0x0, // TAG_End
		0x0, // TAG_End
	}), 1)
	var a struct{}
	err := decoder.Decode(&a)
	assert.Error(t, err)
}

func TestDecode_Dispose_List(t *testing.T) {
	decoder := NewDecoder(bytes.NewBuffer([]byte{
		0xA, 0x0, 0x0, // TAG_Compound()
		0x9, 0x0, 0x1, 'A', // TAG_List(A)
		0x1,                // type=TAG_Byte
		0x0, 0x0, 0x0, 0x2, // len=2
		0x1, 0x2, // [1, 2]
		0x0, // TAG_End
	}))
	var a struct{}
	err := decoder.Decode(&a)
	assert.NoError(t, err)
}

func TestDecode_Compound_NonStruct(t *testing.T) {
	decoder := NewDecoder(bytes.NewBuffer([]byte{
		0xA, 0x0, 0x0, // TAG_Compound()
		0x0, // TAG_End
	}))
	var a int16
	err := decoder.Decode(&a)
	assert.Error(t, err)
}

//go:embed level.nbt
var levelDat []byte

func TestDecode_LevelDat(t *testing.T) {
	decoder := NewDecoder(bytes.NewBuffer(levelDat))
	var a struct {
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
	err := decoder.Decode(&a)
	assert.NoError(t, err)
	assert.Equal(t, 3416194518646519871, a.Data.WorldGenSettings.Seed)
	assert.Equal(t, "1.20.4", a.Data.Version.Name)
	assert.Equal(t, "sample world", a.Data.LevelName)
}

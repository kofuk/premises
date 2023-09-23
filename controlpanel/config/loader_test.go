package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loadToStruct_withoutPrefix(t *testing.T) {
	type InnerStruct struct {
		Foo     string
		Bar     []string
		Baz     []int
		Hoge    string `env:"fuga"`
		Ignored string `env:"-"`
	}
	type Struct struct {
		Name       string
		Num        int
		SignedNum  int
		Uint       uint
		Bool       bool
		Float      float64
		Ignored    string `env:"-"`
		unexported string
		Inner      InnerStruct
	}
	var v Struct
	v.Ignored = "hoge"
	v.unexported = "hoge"
	v.Inner.Ignored = "hoge"
	os.Setenv("name", "hoge")
	os.Setenv("num", "5")
	os.Setenv("signednum", "-5")
	os.Setenv("uint", "5")
	os.Setenv("bool", "true")
	os.Setenv("float", "1.5")
	os.Setenv("ignored", "1234")
	os.Setenv("-", "1234")
	os.Setenv("unexported", "1234")
	os.Setenv("inner_foo", "hogehoge")
	os.Setenv("inner_bar", "foo,bar")
	os.Setenv("inner_baz", "1,3,5")
	os.Setenv("inner_fuga", "moge")
	os.Setenv("inner_-", "moge")
	if err := loadToStruct("", &v); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, Struct{
		Name:       "hoge",
		Num:        5,
		SignedNum:  -5,
		Uint:       5,
		Bool:       true,
		Float:      1.5,
		Ignored:    "hoge",
		unexported: "hoge",
		Inner: InnerStruct{
			Foo:     "hogehoge",
			Bar:     []string{"foo", "bar"},
			Baz:     []int{1, 3, 5},
			Hoge:    "moge",
			Ignored: "hoge",
		},
	}, v)
}

func Test_loadToStruct_withPrefix(t *testing.T) {
	type Struct struct {
		Value string
	}

	var v Struct
	os.Setenv("prefix_value", "hoge")
	if err := loadToStruct("prefix", &v); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, Struct{
		Value: "hoge",
	}, v)
}

func Test_loadToStruct_heavilyNested(t *testing.T) {
	type StructInner2 struct {
		Value string
	}
	type StructInner1 struct {
		Inner2 StructInner2
	}
	type Struct struct {
		Inner1 StructInner1
	}

	var v Struct
	os.Setenv("inner1_inner2_value", "hoge")
	if err := loadToStruct("", &v); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, Struct{
		Inner1: StructInner1{
			Inner2: StructInner2{
				Value: "hoge",
			},
		},
	}, v)
}

func Test_loadToStruct_shouldError(t *testing.T) {
	type Struct struct {
		Value int
	}

	var v Struct
	os.Setenv("value", "hoge")
	if err := loadToStruct("", &v); err != nil {
		assert.True(t, true)
		return
	}
	t.FailNow()
}

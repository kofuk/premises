package config

import (
	"math"
	"os"
	"testing"
)

func Test_loadToStruct(t *testing.T) {
	type Struct struct {
		Name  string
		Num   int
		Bool  bool
		Float float64
		Inner struct {
			Foo  string
			Bar  []string
			Baz  []int
			Hoge string `env:"fuga"`
		}
	}
	var v Struct
	os.Setenv("name", "hoge")
	os.Setenv("num", "5")
	os.Setenv("bool", "true")
	os.Setenv("float", "1.5")
	os.Setenv("inner.foo", "hogehoge")
	os.Setenv("inner.bar", "foo,bar")
	os.Setenv("inner.baz", "1,3,5")
	os.Setenv("inner.fuga", "moge")
	if err := loadToStruct("", &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "hoge" {
		t.Fatal("v.Name not match", v.Name)
	}
	if v.Num != 5 {
		t.Fatal("v.Num not match: ", v.Num)
	}
	if !v.Bool {
		t.Fatal("v.Bool not match: ", v.Bool)
	}
	if math.Abs(v.Float-1.5) > 1e-9 {
		t.Fatal("v.Float not match: ", v.Float)
	}
	if v.Inner.Foo != "hogehoge" {
		t.Fatal("v.Inner.Foo not match: ", v.Inner.Foo)
	}
	if len(v.Inner.Bar) != 2 {
		t.Fatal("len(v.Inner.Bar) not match: ", len(v.Inner.Bar))
	}
	if v.Inner.Bar[0] != "foo" {
		t.Fatal("v.Inner.Bar[0] not match: ", v.Inner.Bar[0])
	}
	if v.Inner.Bar[1] != "bar" {
		t.Fatal("v.Inner.Bar[1] not match: ", v.Inner.Bar[1])
	}
	if len(v.Inner.Baz) != 3 {
		t.Fatal("len(v.Inner.Baz) not match: ", len(v.Inner.Baz))
	}
	if v.Inner.Baz[0] != 1 {
		t.Fatal("v.Inner.Baz[0] not match: ", v.Inner.Baz[0])
	}
	if v.Inner.Baz[1] != 3 {
		t.Fatal("v.Inner.Baz[1] not match: ", v.Inner.Baz[1])
	}
	if v.Inner.Baz[2] != 5 {
		t.Fatal("v.Inner.Baz[2] not match: ", v.Inner.Baz[2])
	}
	if v.Inner.Hoge != "moge" {
		t.Fatal("v.Inner.Hoge not match: ", v.Inner.Hoge)
	}
}

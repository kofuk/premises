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
			Foo string
		}
	}
	var v Struct
	os.Setenv("name", "hoge")
	os.Setenv("num", "5")
	os.Setenv("bool", "true")
	os.Setenv("float", "1.5")
	os.Setenv("inner.foo", "hogehoge")
	if err := loadToStruct(&v); err != nil {
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
}

package gofield_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/henrylee2cn/gofield"
)

type M struct {
	Apple  string `mapper:"a"`
	banana int    `mapper:"b"`
}

func TestMapper1(t *testing.T) {
	maker := func(ft *gofield.FieldType) (string, bool) {
		tag, ok := ft.Tag.Lookup("mapper")
		return tag, ok
	}
	mapper := gofield.NewMapper(maker)

	var p M
	p.Apple = "red"
	p.banana = 7
	m := mapper.MustMake(&p)
	assert.Equal(t, "red", m["a"].String())
	assert.Equal(t, 7, int(m["b"].Int()))
}

func BenchmarkTag_Mapper1(b *testing.B) {
	b.ReportAllocs()
	maker := func(ft *gofield.FieldType) (string, bool) {
		tag, ok := ft.Tag.Lookup("mapper")
		return tag, ok
	}
	mapper := gofield.NewMapper(maker)
	var p M
	p.Apple = "red"
	p.banana = 7
	m := mapper.MustMake(&p)
	assert.Equal(b, "red", m["a"].String())
	assert.Equal(b, 7, int(m["b"].Int()))
	typMap, _ := mapper.StructTypeMap(&p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ := typMap.StructMap(&p)
		a := m.Get("a")
		_ = a.String()
		a.SetString("a1")
		b := m.Get("b")
		_ = b.Int()
		// b.SetInt(71)
	}
	b.StopTimer()
}

func BenchmarkTag_Reflect1(b *testing.B) {
	b.ReportAllocs()
	get := func(tagName string, i interface{}) reflect.Value {
		val := reflect.ValueOf(i)
		if val.Kind() != reflect.Ptr {
			panic("")
		}
		val = val.Elem()
		if val.Kind() != reflect.Struct {
			panic("")
		}
		typ := val.Type()
		mum := typ.NumField()
		for i := 0; i < mum; i++ {
			ft := typ.Field(i)
			if ft.Tag.Get("mapper") == tagName {
				return val.Field(i)
			}
		}
		panic("")
	}

	var p M
	p.Apple = "red"
	p.banana = 7
	assert.Equal(b, "red", get("a", &p).String())
	assert.Equal(b, 7, int(get("b", &p).Int()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := get("a", &p)
		_ = a.String()
		a.SetString("a1")
		b := get("b", &p)
		_ = b.Int()
		// b.SetInt(71)
	}
	b.StopTimer()
}

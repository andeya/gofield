package gofield_test

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"github.com/henrylee2cn/gofield"
)

type (
	P1 struct {
		A int
		b int
		P2
	}
	P2 struct {
		C int
		d *int
		*P3
	}
	P3 struct {
		E int
		f *int
		g **int
	}
)

func BenchmarkGofield(b *testing.B) {
	b.ReportAllocs()
	var p P1
	s := gofield.Access(&p)
	ids := s.Filter(func(t *gofield.FieldType) bool {
		return t.UnderlyingKind() == reflect.Int
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, id := range ids {
			v := s.FieldValue(id)
			v.SetInt(int64(id + 1))
		}
	}
	b.StopTimer()

	assert.Equal(b, 9, s.NumField())
	assert.Equal(b, 1, p.A)
	assert.Equal(b, 2, p.b)
	assert.Equal(b, 4, p.C)
	assert.Equal(b, 5, *p.d)
	assert.Equal(b, 7, p.E)
	assert.Equal(b, 8, *p.f)
	assert.Equal(b, 9, **p.g)
}

func BenchmarkReflect(b *testing.B) {
	b.ReportAllocs()
	var valInt = 1
	var setVal func(v reflect.Value)
	setVal = func(s reflect.Value) {
		num := s.NumField()
		for i := 0; i < num; i++ {
			f := s.Field(i)
			for f.Kind() == reflect.Ptr {
				if f.IsNil() {
					reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).
						Elem().Set(reflect.New(f.Type().Elem()))
				}
				f = f.Elem()
				if f.Kind() == reflect.Ptr && f.IsNil() {
					reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr()))
				}
			}
			switch f.Kind() {
			case reflect.Int:
				if f.CanSet() {
					f.SetInt(int64(valInt))
				} else {
					reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).
						Elem().SetInt(int64(valInt))
				}
				valInt++
			case reflect.Struct:
				valInt++
				setVal(f)
			}
		}
	}
	var p P1
	s := reflect.ValueOf(&p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valInt = 1
		setVal(s.Elem())
	}
	b.StopTimer()

	assert.Equal(b, 3, s.Elem().NumField())
	assert.Equal(b, 1, p.A)
	assert.Equal(b, 2, p.b)
	assert.Equal(b, 4, p.C)
	assert.Equal(b, 5, *p.d)
	assert.Equal(b, 7, p.E)
	assert.Equal(b, 8, *p.f)
	assert.Equal(b, 9, **p.g)
}

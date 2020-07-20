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
		g **int `fe:"target"`
	}
)

func TestGofield(t *testing.T) {
	var p P1
	s, err := gofield.AccessWithErr(&p)
	assert.NoError(t, err)
	ids := s.Filter(func(f *gofield.FieldType) bool {
		return f.UnderlyingKind() == reflect.Int
	})
	ids2 := s.Filter(func(f *gofield.FieldType) bool {
		t.Logf("fid=%d, fullpath=%s tag=%s", f.ID(), f.FullPath(), f.Subtags.String())
		return f.Tag.Get("fe") == "target"
	})
	for _, id := range ids {
		v := s.FieldValue(id)
		v.SetInt(int64(id + 1))
	}
	for _, id := range ids2 {
		v := s.FieldValue(id)
		v.SetInt(999)
	}
	assert.Equal(t, 9, s.NumField())
	assert.Equal(t, 2, p.b)
	assert.Equal(t, 1, p.A)
	assert.Equal(t, 4, p.C)
	assert.Equal(t, 5, *p.d)
	assert.Equal(t, 7, p.E)
	assert.Equal(t, 8, *p.f)
	assert.Equal(t, 999, **p.g)
}

func BenchmarkGofield1(b *testing.B) {
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

func BenchmarkReflect1(b *testing.B) {
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

func BenchmarkGofield2(b *testing.B) {
	b.ReportAllocs()
	var p P1
	s := gofield.Access(&p)
	ids := s.Filter(func(t *gofield.FieldType) bool {
		return t.UnderlyingKind() == reflect.Int
	})
	ids2 := s.Filter(func(f *gofield.FieldType) bool {
		return f.Tag.Get("fe") == "target"
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, id := range ids {
			v := s.FieldValue(id)
			v.SetInt(int64(id + 1))
		}
		for _, id := range ids2 {
			v := s.FieldValue(id)
			v.SetInt(999)
		}
		var p P1
		s = gofield.Access(&p)
	}
	b.StopTimer()

	assert.Equal(b, 9, s.NumField())
	assert.Equal(b, 1, p.A)
	assert.Equal(b, 2, p.b)
	assert.Equal(b, 4, p.C)
	assert.Equal(b, 5, *p.d)
	assert.Equal(b, 7, p.E)
	assert.Equal(b, 8, *p.f)
	assert.Equal(b, 999, **p.g)
}

func BenchmarkReflect2(b *testing.B) {
	b.ReportAllocs()
	var valInt = 1
	var rangeFields func(reflect.Value, func(reflect.Value, reflect.StructTag))
	rangeFields = func(s reflect.Value, fn func(v reflect.Value, tag reflect.StructTag)) {
		num := s.NumField()
		t := s.Type()
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
			fn(f, t.Field(i).Tag)
			if f.Kind() == reflect.Struct {
				rangeFields(f, fn)
			}
		}
	}
	var p P1
	s := reflect.ValueOf(&p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valInt = 1
		rangeFields(s.Elem(), func(v reflect.Value, _ reflect.StructTag) {
			if v.Kind() == reflect.Int {
				if v.CanSet() {
					v.SetInt(int64(valInt))
				} else {
					reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).
						Elem().SetInt(int64(valInt))
				}
			}
			valInt++
		})
		rangeFields(s.Elem(), func(v reflect.Value, tag reflect.StructTag) {
			if tag.Get("fe") == "target" {
				if v.CanSet() {
					v.SetInt(int64(valInt))
				} else {
					reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).
						Elem().SetInt(999)
				}
			}
			valInt++
		})
		var p P1
		s = reflect.ValueOf(&p)
	}
	b.StopTimer()

	assert.Equal(b, 3, s.Elem().NumField())
	assert.Equal(b, 1, p.A)
	assert.Equal(b, 2, p.b)
	assert.Equal(b, 4, p.C)
	assert.Equal(b, 5, *p.d)
	assert.Equal(b, 7, p.E)
	assert.Equal(b, 8, *p.f)
	assert.Equal(b, 999, **p.g)
}

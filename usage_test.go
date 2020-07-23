package gofield_test

import (
	"reflect"
	"testing"

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

func TestGofield1(t *testing.T) {
	var p P1
	s, err := gofield.Access(&p)
	assert.NoError(t, err)
	ids := s.Filter(func(f *gofield.FieldType) bool {
		return f.UnderlyingKind() == reflect.Int
	})
	ids2 := s.Filter(func(f *gofield.FieldType) bool {
		t.Logf("fid=%d, selector=%s, deep=%d, tag=%s", f.ID(), f.Selector(), f.Deep(), f.Tag)
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
	assert.Equal(t, 3, s.Depth())
	assert.Equal(t, 2, p.b)
	assert.Equal(t, 1, p.A)
	assert.Equal(t, 4, p.C)
	assert.Equal(t, 5, *p.d)
	assert.Equal(t, 7, p.E)
	assert.Equal(t, 8, *p.f)
	assert.Equal(t, 999, **p.g)

	t.Logf("\n%s", s.StructType.String())
	for _, fieldType := range s.StructType.FieldTree() {
		t.Logf("\n%s", fieldType.String())
	}
}

func TestGofield2(t *testing.T) {
	st, err := gofield.Analyze(&P1{})
	assert.NoError(t, err)
	ids := st.Filter(func(f *gofield.FieldType) bool {
		return f.UnderlyingKind() == reflect.Int
	})
	ids2 := st.Filter(func(f *gofield.FieldType) bool {
		t.Logf("fid=%d, selector=%s tag=%s", f.ID(), f.Selector(), f.Tag)
		return f.Tag.Get("fe") == "target"
	})
	var p P1
	s := st.MustAccess(&p)
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

func TestGofield3(t *testing.T) {
	st, err := gofield.Analyze(reflect.ValueOf(&P1{}))
	assert.NoError(t, err)
	ids := st.Filter(func(f *gofield.FieldType) bool {
		return f.UnderlyingKind() == reflect.Int
	})
	ids2 := st.Filter(func(f *gofield.FieldType) bool {
		t.Logf("fid=%d, selector=%s tag=%s", f.ID(), f.Selector(), f.Tag)
		return f.Tag.Get("fe") == "target"
	})
	var p P1
	s := st.MustAccess(reflect.ValueOf(&p))
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

func TestGofield4(t *testing.T) {
	accessor := gofield.New(gofield.WithIterator(func(ft *gofield.FieldType) gofield.IterPolicy {
		if ft.Name == "P3" {
			return gofield.SkipAndStop
		}
		switch ft.UnderlyingKind() {
		case reflect.Int, reflect.Struct:
			return gofield.Take
		default:
			return gofield.Skip
		}
	}))
	var p P1
	s := accessor.MustAccess(reflect.ValueOf(&p))
	s.Range(func(t *gofield.FieldType, v reflect.Value) bool {
		if t.UnderlyingKind() != reflect.Struct {
			v.SetInt(int64(t.ID() + 1))
		}
		return true
	})
	assert.Equal(t, 5, s.NumField())
	assert.Equal(t, 1, p.A)
	assert.Equal(t, 2, p.b)
	assert.Equal(t, 4, p.C)
	assert.Equal(t, 5, *p.d)
}

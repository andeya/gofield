# gofield [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/gofield?style=flat-square)](http://goreportcard.com/report/henrylee2cn/gofield) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/henrylee2cn/gofield?tab=doc)

High-performance struct field accessor based on unsafe pointers.

- `gofield` is **simple** to use and has **three times** the performance of `reflect`
- The more complex the struct, the more obvious the advantage of `gofield`
- Support to operate of non-exported fields
- Use unsafe and pre-analysis of types to improve performance

## Compare

- benchmark result
```sh
goos: darwin
goarch: amd64
pkg: github.com/henrylee2cn/gofield
BenchmarkGofield
BenchmarkGofield-4   	26278818	        45.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkReflect
BenchmarkReflect-4   	 6808384	       180 ns/op	       0 B/op	       0 allocs/op
PASS
```

- struct example

```go
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
```

- gofield example
```go
func BenchmarkGofield(b *testing.B) {
	b.ReportAllocs()
	var p P1
	s := gofield.MustAccess(&p)
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
```

- reflect example
```go
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
```

## Mapper

- example

```go
func TestMapper1(t *testing.T) {
	maker := func(ft *gofield.FieldType) (string, bool) {
		tag, err := ft.Subtags.Get("mapper")
		if err != nil {
			return "", false
		}
		return tag.Name, true
	}
	mapper := gofield.NewMapper(maker)

	type P struct {
		Apple  string `mapper:"a"`
		banana int    `mapper:"b"`
	}
	var p P
	p.Apple = "red"
	p.banana = 7
	m := mapper.MustMake(&p)
	assert.Equal(t, "red", m["a"].String())
	assert.Equal(t, 7, int(m["b"].Int()))
}
```

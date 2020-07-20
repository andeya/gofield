# gofield [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/gofield?style=flat-square)](http://goreportcard.com/report/henrylee2cn/gofield) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/henrylee2cn/gofield?tab=doc)

High-performance struct field accessor based on unsafe pointers.

- `gofield` is **simple** to use and has **three times** the performance of `reflect`
- The more complex the struct, the more obvious the advantage of `gofield`
- Support to operate of non-exported fields
- Use unsafe and pre-analysis of types to improve performance

## Compare1

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

## Compare2

- benchmark result
```sh
goos: darwin
goarch: amd64
pkg: github.com/henrylee2cn/gofield
BenchmarkTag_Group1-4            1310958               884 ns/op             608 B/op          7 allocs/op
BenchmarkTag_Reflect1-4           213937              5381 ns/op             856 B/op         53 allocs/op
PASS
```

- struct example

```go
type G struct {
	Apple  string `mapper:"a"`
	banana int    `mapper:"b"`
	C      string `mapper:"c"`
	D      string `mapper:"d"`
	E      string `mapper:"e"`
	E2     string `mapper:"e"`
	E3     string `mapper:"e"`
	E4     string `mapper:"e"`
	E5     string `mapper:"e"`
}
```

- gofield example
```go
func BenchmarkTag_Group1(b *testing.B) {
	b.ReportAllocs()

	maker := func(ft *gofield.FieldType) (string, bool) {
		tag, ok := ft.Tag.Lookup("mapper")
		return tag, ok
	}
	accessor := gofield.New(gofield.WithGroupBy(maker))

	var p G
	p.Apple = "red"
	p.banana = 7
	s := accessor.MustAccess(&p)
	assert.Equal(b, "red", s.GroupValues("a")[0].String())
	assert.Equal(b, 7, int(s.GroupValues("b")[0].Int()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s = accessor.MustAccess(&p)
		a := s.GroupValues("a")
		for _, value := range a {
			_ = value.String()
			value.SetString("a1")
		}
		b := s.GroupValues("b")
		for _, value := range b {
			_ = value.Int()
		}
		c := s.GroupValues("c")
		for _, value := range c {
			_ = value.String()
			value.SetString("a1")
		}
		d := s.GroupValues("d")
		for _, value := range d {
			_ = value.String()
			value.SetString("a1")
		}
		e := s.GroupValues("e")
		for _, value := range e {
			_ = value.String()
			value.SetString("a1")
		}
	}
	b.StopTimer()
}
```

- reflect example
```go
func BenchmarkTag_Reflect1(b *testing.B) {
	b.ReportAllocs()
	var get func(tagName string, i interface{}) []reflect.Value
	get = func(tagName string, i interface{}) []reflect.Value {
		var r []reflect.Value
		val := reflect.ValueOf(i)
		for val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			panic("")
		}
		typ := val.Type()
		mum := typ.NumField()
		for i := 0; i < mum; i++ {
			ft := typ.Field(i)
			if ft.Tag.Get("mapper") == tagName {
				r = append(r, val.Field(i))
			}
		}
		return r
	}

	var p G
	p.Apple = "red"
	p.banana = 7
	assert.Equal(b, "red", get("a", &p)[0].String())
	assert.Equal(b, 7, int(get("b", &p)[0].Int()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := get("a", &p)
		for _, value := range a {
			_ = value.String()
			value.SetString("a1")
		}
		b := get("b", &p)
		for _, value := range b {
			_ = value.Int()
		}
		c := get("c", &p)
		for _, value := range c {
			_ = value.String()
			value.SetString("a1")
		}
		d := get("d", &p)
		for _, value := range d {
			_ = value.String()
			value.SetString("a1")
		}
		e := get("e", &p)
		for _, value := range e {
			_ = value.String()
			value.SetString("a1")
		}
	}
	b.StopTimer()
}
```

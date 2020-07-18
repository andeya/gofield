# gofield

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
BenchmarkGofield-4   	23937832	        48.3 ns/op
BenchmarkReflect
BenchmarkReflect-4   	 7347177	       158 ns/op
PASS
```

- gofield example
```go
func BenchmarkGofield(b *testing.B) {
	var p P1
	s, err := gofield.Access(p)
	assert.EqualError(b, err, "type is not struct pointer")

	s, err = gofield.Access(&p)
	assert.NoError(b, err)
	b.ResetTimer()
	assert.Equal(b, 9, s.NumField())
	for i := 0; i < b.N; i++ {
		num := s.NumField()
		for i := 0; i < num; i++ {
			v := s.FieldValue(i)
			switch v.Kind() {
			case reflect.Int:
				v.SetInt(int64(i))
			case reflect.Struct:
			}
		}
	}
	b.StopTimer()
	assert.Equal(b, 0, p.A)
	assert.Equal(b, 1, p.b)
	assert.Equal(b, 3, p.C)
	assert.Equal(b, 4, p.d)
	assert.Equal(b, 6, p.E)
	assert.Equal(b, 7, *p.f)
	assert.Equal(b, 8, **p.g)
}
```

- reflect example
```go
func BenchmarkReflect(b *testing.B) {
	var p P1
	s := reflect.ValueOf(&p)
	b.ResetTimer()
	s = s.Elem()
	assert.Equal(b, 3, s.NumField())
	var valInt int
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
	for i := 0; i < b.N; i++ {
		valInt = 0
		setVal(s)
	}
	b.StopTimer()
	assert.Equal(b, 0, p.A)
	assert.Equal(b, 1, p.b)
	assert.Equal(b, 3, p.C)
	assert.Equal(b, 4, p.d)
	assert.Equal(b, 6, p.E)
	assert.Equal(b, 7, *p.f)
	assert.Equal(b, 8, **p.g)
}
```

# gofield

High-performance struct field accessor based on unsafe pointers.

## Example

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

## Benchmark

```sh
goos: darwin
goarch: amd64
pkg: github.com/henrylee2cn/gofield
BenchmarkGofield
BenchmarkGofield-4   	23960054	        49.1 ns/op
BenchmarkReflect
BenchmarkReflect-4   	12137930	       101 ns/op
PASS
```

[testing](./access_test.go)

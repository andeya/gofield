# gofield [![report card](https://goreportcard.com/badge/github.com/henrylee2cn/gofield?style=flat-square)](http://goreportcard.com/report/henrylee2cn/gofield) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/henrylee2cn/gofield)

High-performance struct field accessor based on unsafe pointers.

- Compared to `reflect`, it improves performance by two to ten times
- The more complex the struct, the more obvious the performance advantage
- Support to operate of non-exported fields
- Use unsafe and pre-analysis of types to improve performance
- Very simple operation interface

## Example

```go
func Example() {
	type B struct {
		b int
	}
	type A struct {
		a string
		b *B
	}
	var v A
	v.a = "x"
	s := gofield.MustAccess(&v)
	fmt.Println(s.NumField()) // 3
	a := s.FieldValue(0)
	fmt.Println(a.String()) // x
	a.SetString("y")
	fmt.Println(a.String()) // y
	b := s.FieldValue(2)
	fmt.Println(b.Int()) // 0
	b.SetInt(1)
	fmt.Println(b.Int()) // 1
	// output:
	// 3
	// x
	// y
	// 0
	// 1
}
```

## Benchmark

- Various

```sh
goos: darwin
goarch: amd64
pkg: github.com/henrylee2cn/gofield
BenchmarkTag_Gofield1
BenchmarkTag_Gofield1-4      	 1952533	       579 ns/op	     296 B/op	       7 allocs/op
BenchmarkTag_Reflect1
BenchmarkTag_Reflect1-4      	  246312	      4629 ns/op	     856 B/op	      53 allocs/op
BenchmarkNested_Gofield2
BenchmarkNested_Gofield2-4       1703742	       810 ns/op	     176 B/op	       8 allocs/op
BenchmarkNested_Reflect2
BenchmarkNested_Reflect2-4       468402	           2443 ns/op	     288 B/op	      26 allocs/op
BenchmarkNested_Gofield1
BenchmarkNested_Gofield1-4   	 1570126	       598 ns/op	     176 B/op	       8 allocs/op
BenchmarkNested_Reflect1
BenchmarkNested_Reflect1-4   	 1493688	       814 ns/op	     112 B/op	       6 allocs/op
PASS
```

- Overall

```sh
name          reflect time/op      gofield time/op       delta
overall-4     2.52µs ±96%          0.66µs ±11%  -73.96%  (p=0.000 n=90+88)

name          reflect alloc/op     gofield alloc/op      delta
overall-4     419B ±104%           216B ±37%     ~       (p=0.186 n=90+90)

name          reflect allocs/op    gofield allocs/op     delta
overall-4     28.3 ±87%            7.7 ± 9%  -72.94%     (p=0.000 n=90+90)
```

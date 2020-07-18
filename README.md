# gofield

High-performance struct field accessor based on unsafe pointers.

## Benchmark

```sh
goos: darwin
goarch: amd64
pkg: github.com/henrylee2cn/gofield
BenchmarkAccess
BenchmarkAccess-4    	23628463	        48.3 ns/op
BenchmarkReflect
BenchmarkReflect-4   	12691815	        89.9 ns/op
PASS
```

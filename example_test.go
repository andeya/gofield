package gofield_test

import (
	"fmt"

	"github.com/henrylee2cn/gofield"
)

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

// Copyright 2020 Henry Lee. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

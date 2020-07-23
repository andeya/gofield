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

package gofield

import (
	"reflect"
	"unsafe"
)

type (
	// Struct struct accessor
	Struct struct {
		*StructType
		structPtrs []uintptr // idx is struct id
	}
	// Value field value
	Value struct {
		elemVal reflect.Value
		elemPtr uintptr
	}
)

func newStruct(typ *StructType, elemPtr uintptr) *Struct {
	s := &Struct{
		StructType: typ,
		structPtrs: make([]uintptr, typ.structNum),
	}
	s.structPtrs[0] = elemPtr
	return s
}

// FieldValue get the field value corresponding to the id.
// NOTE:
//  By the way, the relevant nil pointer fields will be initialized
func (s *Struct) FieldValue(id int) reflect.Value {
	if !s.checkID(id) {
		return zero
	}
	return s.getOrInit(s.StructType.fields[id], true).elemVal
}

// Field get the field type and value corresponding to the id.
// NOTE:
//  By the way, the relevant nil pointer fields will be initialized
func (s *Struct) Field(id int) (*FieldType, reflect.Value) {
	if !s.checkID(id) {
		return nil, zero
	}
	t := s.StructType.fields[id]
	return t, s.getOrInit(t, true).elemVal
}

// Range traverse all fields, and exit the traversal when fn returns false.
// NOTE:
//  By the way, the relevant nil pointer fields will be initialized
func (s *Struct) Range(fn func(*FieldType, reflect.Value) bool) {
	for _, t := range s.fields {
		if !fn(t, s.getOrInit(t, true).elemVal) {
			return
		}
	}
}

// GroupValues return the field values by group.
// NOTE:
//  By the way, the relevant nil pointer fields will be initialized
func (s *Struct) GroupValues(group string) []reflect.Value {
	a := s.StructType.GroupTypes(group)
	r := make([]reflect.Value, len(a))
	for i, ft := range a {
		r[i] = s.getOrInit(ft, true).elemVal
	}
	return r
}

// NOTE:
//  By the way, the relevant nil pointer fields will be initialized
func (s *Struct) getOrInit(f *FieldType, needValue bool) Value {
	var v Value
	if f.parent == nil {
		// the original caller ensures that it has been initialized
		v.elemPtr = s.structPtrs[0]
		return v
	}
	if f.structID > 0 {
		v.elemPtr = s.structPtrs[f.structID]
		if v.elemPtr > 0 {
			if needValue {
				elemVal := f.elemVal
				elemVal.ptr = unsafe.Pointer(v.elemPtr)
				v.elemVal = (*(*reflect.Value)(unsafe.Pointer(&elemVal))).Elem()
				// v.elemVal = reflect.NewAt(f.elemTyp, unsafe.Pointer(v.elemPtr)).Elem()
			}
			return v
		}
	}
	v.elemPtr = s.getOrInit(f.parent, false).elemPtr + f.Offset
	if f.ptrNum > 0 {
		rawVal := f.rawVal
		rawVal.ptr = unsafe.Pointer(v.elemPtr)
		valPtr := *(*reflect.Value)(unsafe.Pointer(&rawVal))
		// valPtr := reflect.NewAt(f.StructField.Type, unsafe.Pointer(v.elemPtr))
		valPtr = derefPtrAndInit(valPtr, f.ptrNum)
		v.elemPtr = valPtr.Pointer()
		if needValue {
			v.elemVal = valPtr.Elem()
		}
	} else if needValue {
		elemVal := f.elemVal
		elemVal.ptr = unsafe.Pointer(v.elemPtr)
		v.elemVal = (*(*reflect.Value)(unsafe.Pointer(&elemVal))).Elem()
		// valPtr := reflect.NewAt(f.elemTyp, unsafe.Pointer(v.elemPtr))
		// v.elemVal = valPtr.Elem()
	}
	if f.structID > 0 {
		s.structPtrs[f.structID] = v.elemPtr
	}
	return v
}

func derefPtrAndInit(v reflect.Value, numPtr int) reflect.Value {
	for ; numPtr > 0; numPtr-- {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	return v
}

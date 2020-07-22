package gofield

import "unsafe"

type reflectValue struct {
	typ  *uintptr
	ptr  unsafe.Pointer
	flag uintptr
}

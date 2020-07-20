package gofield

import (
	"errors"
	"reflect"

	"github.com/henrylee2cn/ameda"
)

func parseStructPtr(structPtr interface{}) (int32, uintptr) {
	if j, ok := structPtr.(reflect.Value); ok {
		tid := ameda.RuntimeTypeIDOf(structPtr)
		ptr := j.Pointer()
		return tid, ptr
	}
	val := ameda.ValueOf(structPtr)
	tid := val.RuntimeTypeID()
	ptr := val.Pointer()
	return tid, ptr
}

func parseStructPtrWithCheck(structPtr interface{}) (int32, uintptr, error) {
	if j, ok := structPtr.(reflect.Value); ok {
		if j.Kind() != reflect.Ptr || j.Elem().Kind() != reflect.Struct {
			return 0, 0, errors.New("type is not struct pointer")
		}
		tid := ameda.RuntimeTypeIDOf(structPtr)
		ptr := j.Pointer()
		return tid, ptr, nil
	}
	val := ameda.ValueOf(structPtr)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return 0, 0, errors.New("type is not struct pointer")
	}
	tid := val.RuntimeTypeID()
	ptr := val.Pointer()
	return tid, ptr, nil
}

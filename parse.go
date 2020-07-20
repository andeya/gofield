package gofield

import (
	"reflect"

	"github.com/henrylee2cn/ameda"
)

// func parseStructPtr(structPtr interface{}) uintptr {
// 	if val, ok := structPtr.(reflect.Value); ok {
// 		return val.Pointer()
// 	}
// 	val := ameda.ValueOf(structPtr)
// 	return val.Pointer()
// }

func parseStructPtrWithCheck(structPtr interface{}) (uintptr, error) {
	if val, ok := structPtr.(reflect.Value); ok {
		if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
			return 0, errIllegalType
		}
		return val.Pointer(), nil
	}
	val := ameda.ValueOf(structPtr)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return 0, errIllegalType
	}
	return val.Pointer(), nil
}

func parseStructInfo(structPtr interface{}) (int32, uintptr) {
	if val, ok := structPtr.(reflect.Value); ok {
		tid := ameda.RuntimeTypeIDOf(structPtr)
		ptr := val.Pointer()
		return tid, ptr
	}
	val := ameda.ValueOf(structPtr)
	tid := val.RuntimeTypeID()
	ptr := val.Pointer()
	return tid, ptr
}

func parseStructInfoWithCheck(structPtr interface{}) (int32, uintptr, error) {
	if val, ok := structPtr.(reflect.Value); ok {
		if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
			return 0, 0, errIllegalType
		}
		tid := ameda.RuntimeTypeIDOf(structPtr)
		ptr := val.Pointer()
		return tid, ptr, nil
	}
	val := ameda.ValueOf(structPtr)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return 0, 0, errIllegalType
	}
	tid := val.RuntimeTypeID()
	ptr := val.Pointer()
	return tid, ptr, nil
}

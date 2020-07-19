package gofield

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"

	"github.com/henrylee2cn/ameda"
)

type (
	Struct struct {
		typ         *StructType
		value       *Value
		fieldValues []*Value // idx is int
	}
	Value struct {
		elemVal reflect.Value
		elemPtr uintptr
	}
	StructType struct {
		tid      int32
		elemType reflect.Type
		fields   []*FieldType
		deep     int
	}
	FieldType struct {
		id       int
		fullPath string
		reflect.StructField
		ptrNum    int
		elemTyp   reflect.Type
		parent    *FieldType
		cacheable bool
	}
	StructTypeStore struct {
		dict map[int32]*StructType // key is runtime type ID
		sync.RWMutex
	}
)

var (
	cacheAnyFields = true
	store          = newStructTypeStore()
)

func newStructTypeStore() *StructTypeStore {
	return &StructTypeStore{
		dict: make(map[int32]*StructType, 128),
	}
}

func (s *StructTypeStore) load(tid int32) (*StructType, bool) {
	s.RLock()
	sTyp, ok := s.dict[tid]
	s.RUnlock()
	return sTyp, ok
}

func (s *StructTypeStore) store(sTyp *StructType) {
	s.Lock()
	s.dict[sTyp.tid] = sTyp
	s.Unlock()
}

func Prepare(structPtr interface{}) error {
	var val ameda.Value
	switch j := structPtr.(type) {
	case reflect.Value:
		val = ameda.ValueFrom2(&j)
	default:
		val = ameda.ValueOf(structPtr)
	}
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("type is not struct pointer")
	}
	tid := val.RuntimeTypeID()
	_, ok := store.load(tid)
	if ok {
		return nil
	}
	sTyp := newStructType(structPtr)
	store.store(sTyp)
	return nil
}

func Access(structPtr interface{}) *Struct {
	var val ameda.Value
	switch j := structPtr.(type) {
	case reflect.Value:
		val = ameda.ValueFrom2(&j)
	default:
		val = ameda.ValueOf(structPtr)
	}
	tid := val.RuntimeTypeID()
	sTyp, ok := store.load(tid)
	if !ok {
		sTyp = newStructType(structPtr)
		store.store(sTyp)
	}
	return newStruct(sTyp, val.Pointer())
}

func newStruct(typ *StructType, elemPtr uintptr) *Struct {
	return &Struct{
		typ: typ,
		value: &Value{
			elemPtr: elemPtr,
		},
		fieldValues: make([]*Value, len(typ.fields)),
	}
}

var zero reflect.Value

func (s *Struct) NumField() int {
	return len(s.typ.fields)
}

func (s *Struct) FieldType(id int) *FieldType {
	if !s.checkID(id) {
		return nil
	}
	return s.typ.fields[id]
}

func (s *Struct) Filter(fn func(*FieldType) bool) []int {
	list := make([]int, 0, s.NumField())
	for id, field := range s.typ.fields {
		if fn(field) {
			list = append(list, id)
		}
	}
	return list
}

func (s *Struct) FieldValue(id int) reflect.Value {
	if !s.checkID(id) {
		return zero
	}
	v := s.fieldValues[id]
	if v != nil {
		return v.elemVal
	}
	return s.typ.fields[id].init(s).elemVal
}

func (s *Struct) checkID(id int) bool {
	return id >= 0 && id < len(s.fieldValues)
}

func (f *FieldType) init(s *Struct) *Value {
	if f.parent == nil {
		return s.value // the original caller ensures that it has been initialized
	}
	v := s.fieldValues[f.id]
	if v != nil {
		return v
	}
	pVal := f.parent.init(s)
	ptr := pVal.elemPtr + f.Offset
	valPtr := reflect.NewAt(f.StructField.Type, unsafe.Pointer(ptr))
	if f.ptrNum > 0 {
		valPtr = derefPtrAndInit(valPtr, f.ptrNum)
	}
	val := &Value{
		elemVal: valPtr.Elem(),
		elemPtr: valPtr.Pointer(),
	}
	if f.cacheable {
		s.fieldValues[f.id] = val
	}
	return val
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

func (f *FieldType) ID() int {
	return f.id
}

func (f *FieldType) FullPath() string {
	return f.fullPath
}

func (f *FieldType) Kind() reflect.Kind {
	return f.StructField.Type.Kind()
}

func (f *FieldType) UnderlyingKind() reflect.Kind {
	return f.elemTyp.Kind()
}

const maxDeep = 16

func newStructType(structPtr interface{}) *StructType {
	v, ok := structPtr.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(structPtr)
	}
	structTyp := v.Elem().Type()
	sTyp := &StructType{
		tid:      ameda.ValueFrom(v).RuntimeTypeID(),
		elemType: structTyp,
		fields:   make([]*FieldType, 0, 8),
	}
	sTyp.parseFields(&FieldType{}, structTyp)
	return sTyp
}

func (s *StructType) parseFields(parent *FieldType, structTyp reflect.Type) {
	if s.deep >= maxDeep {
		return
	}
	baseId := len(s.fields)
	numField := structTyp.NumField()
	s.fields = append(s.fields, make([]*FieldType, numField)...)

	for i := 0; i < numField; i++ {
		f := structTyp.Field(i)
		elemTyp := f.Type
		var ptrNum int
		for elemTyp.Kind() == reflect.Ptr {
			elemTyp = elemTyp.Elem()
			ptrNum++
		}
		isStruct := elemTyp.Kind() == reflect.Struct
		field := &FieldType{
			id:          baseId + i, // 0, 1, 2, ...
			fullPath:    joinFieldName(parent.fullPath, f.Name),
			StructField: f,
			ptrNum:      ptrNum,
			elemTyp:     elemTyp,
			parent:      parent,
			cacheable:   cacheAnyFields || isStruct,
		}
		s.fields[field.id] = field
		if isStruct {
			s.parseFields(field, elemTyp)
		}
	}
}

func joinFieldName(parentPath, name string) string {
	if parentPath == "" {
		return name
	}
	return parentPath + "." + name
}

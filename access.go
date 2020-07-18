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
		ptrNum  int
		elemTyp reflect.Type
		parent  *FieldType
	}
	StructTypeStore struct {
		dict map[int32]*StructType // key is runtime type ID
		sync.RWMutex
	}
)

var (
	store = newStructTypeStore()
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

func Access(structPtr interface{}) (*Struct, error) {
	var val reflect.Value
	switch j := structPtr.(type) {
	case reflect.Value:
		if j.Kind() != reflect.Ptr || j.Elem().Kind() != reflect.Struct {
			return nil, errors.New("type is not struct pointer")
		}
		val = j
	default:
		val = reflect.ValueOf(j)
		if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
			return nil, errors.New("type is not struct pointer")
		}
	}
	tid := ameda.ValueFrom(val).RuntimeTypeID()
	sTyp, ok := store.load(tid)
	if !ok {
		var err error
		sTyp, err = newStructType(structPtr)
		if err != nil {
			return nil, err
		}
		store.store(sTyp)
	}
	return newStruct(sTyp, val), nil
}

func newStruct(typ *StructType, val reflect.Value) *Struct {
	return &Struct{
		typ: typ,
		value: &Value{
			elemVal: val.Elem(),
			elemPtr: val.Pointer(),
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

func (s *Struct) FieldValue(id int) reflect.Value {
	v := s.getOrInit(id)
	if v == nil {
		return zero
	}
	return v.elemVal
}

func (s *Struct) checkID(id int) bool {
	return id >= 0 && id < len(s.fieldValues)
}

func (s *Struct) getOrInit(id int) *Value {
	if !s.checkID(id) {
		return nil
	}
	v := s.fieldValues[id]
	if v != nil {
		return v
	}
	s.typ.fields[id].init(s)
	return s.fieldValues[id]
}

func (f *FieldType) init(s *Struct) uintptr {
	if f.parent == nil {
		return s.value.elemPtr // the original caller ensures that it has been initialized
	}
	pptr := f.parent.init(s)
	v := s.fieldValues[f.id]
	if v != nil {
		return v.elemPtr
	}
	ptr := pptr + f.Offset
	valPtr := reflect.NewAt(f.StructField.Type, unsafe.Pointer(ptr))
	if f.ptrNum > 0 {
		valPtr = derefPtrAndInit(valPtr, f.ptrNum)
	}
	elemPtr := valPtr.Pointer()
	s.fieldValues[f.id] = &Value{
		elemVal: valPtr.Elem(),
		elemPtr: elemPtr,
	}
	return elemPtr
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

func newStructType(structPtr interface{}) (*StructType, error) {
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
	return sTyp, sTyp.parseFields(&FieldType{}, structTyp)
}

func (s *StructType) parseFields(parent *FieldType, structTyp reflect.Type) error {
	if s.deep >= maxDeep {
		return nil
	}
	baseId := len(s.fields)
	numField := structTyp.NumField()
	s.fields = append(s.fields, make([]*FieldType, numField)...)

	for i := 0; i < numField; i++ {
		f := structTyp.Field(i)
		if f.PkgPath != "" {
			// TODO: skip exported field
		}
		elemTyp := f.Type
		var ptrNum int
		for elemTyp.Kind() == reflect.Ptr {
			elemTyp = elemTyp.Elem()
			ptrNum++
		}
		field := &FieldType{
			id:          baseId + i, // 0, 1, 2, ...
			fullPath:    joinFieldName(parent.fullPath, f.Name),
			StructField: f,
			ptrNum:      ptrNum,
			elemTyp:     elemTyp,
			parent:      parent,
		}
		s.fields[field.id] = field
		if elemTyp.Kind() == reflect.Struct {
			err := s.parseFields(field, elemTyp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func joinFieldName(parentPath, name string) string {
	if parentPath == "" {
		return name
	}
	return parentPath + "." + name
}

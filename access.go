package gofield

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/structtag"
)

type (
	// Struct struct accessor
	Struct struct {
		*StructType
		value       Value
		fieldValues []Value // idx is int
	}
	// StructType struct type info
	StructType struct {
		tid      int32
		elemType reflect.Type
		fields   []*FieldType
		deep     int
	}
	// FieldID id assigned to each field in sequence
	FieldID = int
	// FieldType field type info
	FieldType struct {
		id       int
		fullPath string
		reflect.StructField
		Subtags structtag.Tags
		ptrNum  int
		elemTyp reflect.Type
		parent  *FieldType
	}
	// Value field value
	Value struct {
		elemVal reflect.Value
		elemPtr uintptr
	}
	// StructTypeStore struct type info global cache
	StructTypeStore struct {
		dict map[int32]*StructType // key is runtime type ID
		sync.RWMutex
	}
)

const maxFieldDeep = 16

var (
	store = &StructTypeStore{
		dict: make(map[int32]*StructType, 1024),
	}
	zero reflect.Value
)

//go:nosplit
func (s *StructTypeStore) load(tid int32) (*StructType, bool) {
	s.RLock()
	sTyp, ok := s.dict[tid]
	s.RUnlock()
	return sTyp, ok
}

//go:nosplit
func (s *StructTypeStore) store(sTyp *StructType) {
	s.Lock()
	s.dict[sTyp.tid] = sTyp
	s.Unlock()
}

// Analyze analyze the struct and return its type info.
//go:nosplit
func Analyze(structPtr interface{}) (*StructType, error) {
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
		for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			return nil, errors.New("type is not struct pointer")
		}
		sTyp = newStructType(tid, structPtr)
		store.store(sTyp)
	}
	return sTyp, nil
}

// AccessWithErr analyze the struct type info and create struct accessor.
//go:nosplit
func AccessWithErr(structPtr interface{}) (*Struct, error) {
	var val ameda.Value
	switch j := structPtr.(type) {
	case reflect.Value:
		val = ameda.ValueFrom2(&j)
	default:
		val = ameda.ValueOf(structPtr)
	}
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return nil, errors.New("type is not struct pointer")
	}
	tid := val.RuntimeTypeID()
	sTyp, ok := store.load(tid)
	if !ok {
		sTyp = newStructType(tid, structPtr)
		store.store(sTyp)
	}
	return newStruct(sTyp, val.Pointer()), nil
}

// Access analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
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
		sTyp = newStructType(tid, structPtr)
		store.store(sTyp)
	}
	return newStruct(sTyp, val.Pointer())
}

// AccessWithErr create a new struct accessor.
//go:nosplit
func (s *StructType) AccessWithErr(structPtr interface{}) (*Struct, error) {
	var val ameda.Value
	switch j := structPtr.(type) {
	case reflect.Value:
		val = ameda.ValueFrom2(&j)
	default:
		val = ameda.ValueOf(structPtr)
	}
	tid := val.RuntimeTypeID()
	if s.tid != tid {
		return nil, errors.New("type mismatch")
	}
	return newStruct(s, val.Pointer()), nil
}

// Access create a new struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func (s *StructType) Access(structPtr interface{}) *Struct {
	var val ameda.Value
	switch j := structPtr.(type) {
	case reflect.Value:
		val = ameda.ValueFrom2(&j)
	default:
		val = ameda.ValueOf(structPtr)
	}
	tid := val.RuntimeTypeID()
	if s.tid != tid {
		panic("type mismatch")
	}
	return newStruct(s, val.Pointer())
}

//go:nosplit
func newStruct(typ *StructType, elemPtr uintptr) *Struct {
	return &Struct{
		StructType: typ,
		value: Value{
			elemPtr: elemPtr,
		},
		fieldValues: make([]Value, len(typ.fields)),
	}
}

// RuntimeID get the runtime id of struct.
//go:nosplit
func (s *StructType) RuntimeID() int32 {
	return s.tid
}

// NumField get the number of fields.
//go:nosplit
func (s *StructType) NumField() int {
	return len(s.fields)
}

// FieldType get the field type info corresponding to the id.
//go:nosplit
func (s *StructType) FieldType(id int) *FieldType {
	if !s.checkID(id) {
		return nil
	}
	return s.fields[id]
}

// FieldValue get the field value corresponding to the id.
//go:nosplit
func (s *Struct) FieldValue(id int) reflect.Value {
	if !s.checkID(id) {
		return zero
	}
	v := s.fieldValues[id]
	if v.elemPtr > 0 {
		return v.elemVal
	}
	return s.StructType.fields[id].init(s).elemVal
}

// Field get the field type and value corresponding to the id.
//go:nosplit
func (s *Struct) Field(id int) (*FieldType, reflect.Value) {
	if !s.checkID(id) {
		return nil, zero
	}
	t := s.StructType.fields[id]
	v := s.fieldValues[id]
	if v.elemPtr > 0 {
		return t, v.elemVal
	}
	return t, t.init(s).elemVal
}

// Filter filter all fields and return a list of their ids.
//go:nosplit
func (s *StructType) Filter(fn func(*FieldType) bool) []int {
	list := make([]int, 0, s.NumField())
	for id, field := range s.fields {
		if fn(field) {
			list = append(list, id)
		}
	}
	return list
}

//go:nosplit
func (s *StructType) checkID(id int) bool {
	return id >= 0 && id < len(s.fields)
}

func (f *FieldType) init(s *Struct) Value {
	if f.parent == nil {
		return s.value // the original caller ensures that it has been initialized
	}
	v := s.fieldValues[f.id]
	if v.elemPtr > 0 {
		return v
	}
	pVal := f.parent.init(s)
	ptr := pVal.elemPtr + f.Offset
	valPtr := reflect.NewAt(f.StructField.Type, unsafe.Pointer(ptr))
	if f.ptrNum > 0 {
		valPtr = derefPtrAndInit(valPtr, f.ptrNum)
	}
	v = Value{
		elemVal: valPtr.Elem(),
		elemPtr: valPtr.Pointer(),
	}
	s.fieldValues[f.id] = v
	return v
}

//go:nosplit
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

// ID get the field id.
//go:nosplit
func (f *FieldType) ID() int {
	return f.id
}

// FullPath get the field full path.
//go:nosplit
func (f *FieldType) FullPath() string {
	return f.fullPath
}

// Kind get the field kind.
//go:nosplit
func (f *FieldType) Kind() reflect.Kind {
	return f.StructField.Type.Kind()
}

// UnderlyingKind get the underlying kind of the field
//go:nosplit
func (f *FieldType) UnderlyingKind() reflect.Kind {
	return f.elemTyp.Kind()
}

//go:nosplit
func newStructType(tid int32, structPtr interface{}) *StructType {
	v, ok := structPtr.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(structPtr)
	}
	v = ameda.DereferencePtrValue(v)
	structTyp := v.Type()
	sTyp := &StructType{
		tid:      tid,
		elemType: structTyp,
		fields:   make([]*FieldType, 0, 16),
	}
	sTyp.parseFields(&FieldType{}, structTyp)
	return sTyp
}

func (s *StructType) parseFields(parent *FieldType, structTyp reflect.Type) {
	if s.deep >= maxFieldDeep {
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
		field := &FieldType{
			id:          baseId + i, // 0, 1, 2, ...
			fullPath:    joinFieldName(parent.fullPath, f.Name),
			StructField: f,
			ptrNum:      ptrNum,
			elemTyp:     elemTyp,
			parent:      parent,
		}
		tags, _ := structtag.Parse(string(f.Tag))
		if tags != nil {
			field.Subtags = *tags
		}
		s.fields[field.id] = field
		if elemTyp.Kind() == reflect.Struct {
			s.parseFields(field, elemTyp)
		}
	}
}

//go:nosplit
func joinFieldName(parentPath, name string) string {
	if parentPath == "" {
		return name
	}
	return parentPath + "." + name
}

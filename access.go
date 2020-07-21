package gofield

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"

	"github.com/henrylee2cn/ameda"
)

type (
	// Accessor struct accessor factory
	Accessor struct {
		dict      map[int32]*StructType // key is runtime type ID
		rw        sync.RWMutex
		groupFunc FieldGroupFunc
	}
	// Struct struct accessor
	Struct struct {
		*StructType
		value       Value
		fieldValues []Value // idx is int
	}
	// StructType struct type info
	StructType struct {
		tid        int32
		elemType   reflect.Type
		fields     []*FieldType
		fieldGroup map[string][]*FieldType
		deep       int
	}
	// FieldID id assigned to each field in sequence
	FieldID = int
	// FieldType field type info
	FieldType struct {
		id       int
		selector string
		reflect.StructField
		ptrNum  int
		elemTyp reflect.Type
		parent  *FieldType
	}
	// Value field value
	Value struct {
		elemVal reflect.Value
		elemPtr uintptr
	}
	// FieldGroupFunc create the group of the field type
	FieldGroupFunc func(*FieldType) (string, bool)
	// Option accessor option
	Option func(*Accessor)
)

const maxFieldDeep = 16

var (
	defaultGofield  = New()
	zero            = reflect.Value{}
	errTypeMismatch = errors.New("type mismatch")
	errIllegalType  = errors.New("type is not struct pointer")
)

// WithGroupBy set FieldGroupFunc to *Accessor.
func WithGroupBy(fn FieldGroupFunc) Option {
	return func(g *Accessor) {
		g.groupFunc = fn
	}
}

// New create a new struct accessor factory.
func New(opt ...Option) *Accessor {
	g := &Accessor{
		dict: make(map[int32]*StructType, 1024),
	}
	for _, fn := range opt {
		fn(g)
	}
	return g
}

//go:nosplit
func (s *Accessor) load(tid int32) (*StructType, bool) {
	s.rw.RLock()
	sTyp, ok := s.dict[tid]
	s.rw.RUnlock()
	return sTyp, ok
}

//go:nosplit
func (s *Accessor) store(sTyp *StructType) {
	s.rw.Lock()
	s.dict[sTyp.tid] = sTyp
	s.rw.Unlock()
}

// MustAnalyze analyze the struct and return its type info.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func MustAnalyze(structPtr interface{}) *StructType {
	return defaultGofield.MustAnalyze(structPtr)
}

// MustAnalyze analyze the struct and return its type info.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func (g *Accessor) MustAnalyze(structPtr interface{}) *StructType {
	s, err := g.Analyze(structPtr)
	if err != nil {
		panic(err)
	}
	return s
}

// Analyze analyze the struct and return its type info.
//go:nosplit
func Analyze(structPtr interface{}) (*StructType, error) {
	return defaultGofield.Analyze(structPtr)
}

// Analyze analyze the struct and return its type info.
//go:nosplit
func (g *Accessor) Analyze(structPtr interface{}) (*StructType, error) {
	tid, _, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	return g.analyze(tid, structPtr), nil
}

//go:nosplit
func (g *Accessor) analyze(tid int32, structPtr interface{}) *StructType {
	sTyp, ok := g.load(tid)
	if !ok {
		sTyp = g.newStructType(tid, structPtr)
		g.store(sTyp)
	}
	return sTyp
}

// MustAccess analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func MustAccess(structPtr interface{}) *Struct {
	return defaultGofield.MustAccess(structPtr)
}

// MustAccess analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func (g *Accessor) MustAccess(structPtr interface{}) *Struct {
	tid, ptr := parseStructInfo(structPtr)
	sTyp, ok := g.load(tid)
	if !ok {
		sTyp = g.newStructType(tid, structPtr)
		g.store(sTyp)
	}
	return newStruct(sTyp, ptr)
}

// Access analyze the struct type info and create struct accessor.
//go:nosplit
func Access(structPtr interface{}) (*Struct, error) {
	return defaultGofield.Access(structPtr)
}

// Access analyze the struct type info and create struct accessor.
//go:nosplit
func (g *Accessor) Access(structPtr interface{}) (*Struct, error) {
	tid, ptr, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	sTyp, ok := g.load(tid)
	if !ok {
		sTyp = g.newStructType(tid, structPtr)
		g.store(sTyp)
	}
	return newStruct(sTyp, ptr), nil
}

// MustAccess create a new struct accessor.
// NOTE:
//  If structPtr is not a struct pointer or type mismatch, it will cause panic.
//go:nosplit
func (s *StructType) MustAccess(structPtr interface{}) *Struct {
	a, err := s.Access(structPtr)
	if err != nil {
		panic(err)
	}
	return a
}

// Access create a new struct accessor.
//go:nosplit
func (s *StructType) Access(structPtr interface{}) (*Struct, error) {
	tid, ptr := parseStructInfo(structPtr)
	if s.tid != tid {
		return nil, errTypeMismatch
	}
	return newStruct(s, ptr), nil
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

// Depth return the struct nesting depth(at least 1).
//go:nosplit
func (s *StructType) Depth() int {
	return s.deep
}

// RuntimeTypeID get the runtime type id of struct.
//go:nosplit
func (s *StructType) RuntimeTypeID() int32 {
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

// GroupTypes return the field types by group.
//go:nosplit
func (s *StructType) GroupTypes(group string) []*FieldType {
	a := s.fieldGroup[group]
	return a
}

// GroupValues return the field values by group.
//go:nosplit
func (s *Struct) GroupValues(group string) []reflect.Value {
	a := s.StructType.GroupTypes(group)
	r := make([]reflect.Value, len(a))
	for i, ft := range a {
		v := s.fieldValues[ft.id]
		if v.elemPtr > 0 {
			r[i] = v.elemVal
		} else {
			r[i] = ft.init(s).elemVal
		}
	}
	return r
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

// Selector get the field full path.
//go:nosplit
func (f *FieldType) Selector() string {
	return f.selector
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
func (g *Accessor) newStructType(tid int32, structPtr interface{}) *StructType {
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
	if g.groupFunc != nil {
		sTyp.groupBy(g.groupFunc)
	}
	return sTyp
}

func (s *StructType) parseFields(parent *FieldType, structTyp reflect.Type) {
	if s.deep >= maxFieldDeep {
		return
	}
	s.deep++
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
			selector:    joinFieldName(parent.selector, f.Name),
			StructField: f,
			ptrNum:      ptrNum,
			elemTyp:     elemTyp,
			parent:      parent,
		}
		s.fields[field.id] = field
		if elemTyp.Kind() == reflect.Struct {
			s.parseFields(field, elemTyp)
		}
	}
}

//go:nosplit
func (s *StructType) groupBy(fn FieldGroupFunc) {
	s.fieldGroup = make(map[string][]*FieldType, len(s.fields))
	for _, field := range s.fields {
		group, ok := fn(field)
		if ok {
			a := s.fieldGroup[group]
			s.fieldGroup[group] = append(a, field)
		}
	}
}

//go:nosplit
func joinFieldName(parentPath, name string) string {
	return parentPath + "." + name
}

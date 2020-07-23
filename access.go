package gofield

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/henrylee2cn/ameda"
)

type (
	// Accessor struct accessor factory
	Accessor struct {
		dict     map[int32]*StructType // key is runtime type ID
		rw       sync.RWMutex
		groupBy  GroupByFunc
		iterator IteratorFunc
		maxDeep  int
	}
	// Struct struct accessor
	Struct struct {
		*StructType
		structPtrs []uintptr // idx is struct id
	}
	// StructType struct type info
	StructType struct {
		tid        int32
		fields     []*FieldType
		fieldGroup map[string][]*FieldType
		depth      int
		tree       *FieldType // id = -1
		structNum  int
	}
	// FieldID id assigned to each field in sequence
	FieldID = int
	// FieldType field type info
	FieldType struct {
		id       int
		structID int // 1, 2, 3, ...
		selector string
		deep     int
		ptrNum   int
		elemTyp  reflect.Type
		elemVal  reflectValue
		rawVal   reflectValue
		parent   *FieldType
		children []*FieldType
		reflect.StructField
	}
	// Value field value
	Value struct {
		elemVal reflect.Value
		elemPtr uintptr
	}
)

const rootID = -1

var (
	defaultAccessor = New()
	zero            = reflect.Value{}
	errTypeMismatch = errors.New("type mismatch")
	errIllegalType  = errors.New("type is not struct pointer")
)

// New create a new struct accessor factory.
func New(opt ...Option) *Accessor {
	a := &Accessor{
		dict:    make(map[int32]*StructType, 1024),
		maxDeep: 16,
	}
	for _, fn := range opt {
		fn(a)
	}
	return a
}

func (a *Accessor) load(tid int32) (*StructType, bool) {
	a.rw.RLock()
	sTyp, ok := a.dict[tid]
	a.rw.RUnlock()
	return sTyp, ok
}

func (a *Accessor) store(sTyp *StructType) {
	a.rw.Lock()
	a.dict[sTyp.tid] = sTyp
	a.rw.Unlock()
}

// MustAnalyze analyze the struct and return its type info.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func MustAnalyze(structPtr interface{}) *StructType {
	return defaultAccessor.MustAnalyze(structPtr)
}

// MustAnalyze analyze the struct and return its type info.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func (a *Accessor) MustAnalyze(structPtr interface{}) *StructType {
	s, err := a.Analyze(structPtr)
	if err != nil {
		panic(err)
	}
	return s
}

// Analyze analyze the struct and return its type info.
func Analyze(structPtr interface{}) (*StructType, error) {
	return defaultAccessor.Analyze(structPtr)
}

// Analyze analyze the struct and return its type info.
func (a *Accessor) Analyze(structPtr interface{}) (*StructType, error) {
	tid, _, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	return a.analyze(tid, structPtr), nil
}

func (a *Accessor) analyze(tid int32, structPtr interface{}) *StructType {
	sTyp, ok := a.load(tid)
	if !ok {
		sTyp = a.newStructType(tid, structPtr)
		a.store(sTyp)
	}
	return sTyp
}

// MustAccess analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func MustAccess(structPtr interface{}) *Struct {
	return defaultAccessor.MustAccess(structPtr)
}

// MustAccess analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func (a *Accessor) MustAccess(structPtr interface{}) *Struct {
	tid, ptr := parseStructInfo(structPtr)
	sTyp, ok := a.load(tid)
	if !ok {
		sTyp = a.newStructType(tid, structPtr)
		a.store(sTyp)
	}
	return newStruct(sTyp, ptr)
}

// Access analyze the struct type info and create struct accessor.
func Access(structPtr interface{}) (*Struct, error) {
	return defaultAccessor.Access(structPtr)
}

// Access analyze the struct type info and create struct accessor.
func (a *Accessor) Access(structPtr interface{}) (*Struct, error) {
	tid, ptr, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	sTyp, ok := a.load(tid)
	if !ok {
		sTyp = a.newStructType(tid, structPtr)
		a.store(sTyp)
	}
	return newStruct(sTyp, ptr), nil
}

// MustAccess create a new struct accessor.
// NOTE:
//  If structPtr is not a struct pointer or type mismatch, it will cause panic.
func (s *StructType) MustAccess(structPtr interface{}) *Struct {
	a, err := s.Access(structPtr)
	if err != nil {
		panic(err)
	}
	return a
}

// Access create a new struct accessor.
func (s *StructType) Access(structPtr interface{}) (*Struct, error) {
	tid, ptr := parseStructInfo(structPtr)
	if s.tid != tid {
		return nil, errTypeMismatch
	}
	return newStruct(s, ptr), nil
}

func newStruct(typ *StructType, elemPtr uintptr) *Struct {
	s := &Struct{
		StructType: typ,
		structPtrs: make([]uintptr, typ.structNum),
	}
	s.structPtrs[0] = elemPtr
	return s
}

// Depth return the struct nesting depth(at least 1).
func (s *StructType) Depth() int {
	return s.depth
}

// RuntimeTypeID get the runtime type id of struct.
func (s *StructType) RuntimeTypeID() int32 {
	return s.tid
}

// NumField get the number of fields.
func (s *StructType) NumField() int {
	return len(s.fields)
}

// FieldType get the field type info corresponding to the id.
func (s *StructType) FieldType(id int) *FieldType {
	if !s.checkID(id) {
		return nil
	}
	return s.fields[id]
}

// Filter filter all fields and return a list of their ids.
func (s *StructType) Filter(fn func(*FieldType) bool) []int {
	list := make([]int, 0, s.NumField())
	for id, field := range s.fields {
		if fn(field) {
			list = append(list, id)
		}
	}
	return list
}

// FieldValue get the field value corresponding to the id.
func (s *Struct) FieldValue(id int) reflect.Value {
	if !s.checkID(id) {
		return zero
	}
	return s.getOrInit(s.StructType.fields[id], true).elemVal
}

// Field get the field type and value corresponding to the id.
func (s *Struct) Field(id int) (*FieldType, reflect.Value) {
	if !s.checkID(id) {
		return nil, zero
	}
	t := s.StructType.fields[id]
	return t, s.getOrInit(t, true).elemVal
}

// Range traverse all fields, and exit the traversal when fn returns false.
func (s *Struct) Range(fn func(*FieldType, reflect.Value) bool) {
	for _, t := range s.fields {
		if !fn(t, s.getOrInit(t, true).elemVal) {
			return
		}
	}
}

// GroupTypes return the field types by group.
func (s *StructType) GroupTypes(group string) []*FieldType {
	a := s.fieldGroup[group]
	return a
}

// GroupValues return the field values by group.
func (s *Struct) GroupValues(group string) []reflect.Value {
	a := s.StructType.GroupTypes(group)
	r := make([]reflect.Value, len(a))
	for i, ft := range a {
		r[i] = s.getOrInit(ft, true).elemVal
	}
	return r
}

func (s *StructType) checkID(id int) bool {
	return id >= 0 && id < len(s.fields)
}

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

// ID get the field id.
func (f *FieldType) ID() int {
	return f.id
}

// Selector get the field full path.
func (f *FieldType) Selector() string {
	return f.selector
}

// String dump the id and selector on the field tree.
func (s *StructType) String() string {
	return s.Dump()
}

// Dump dump the id and selector on the field tree.
func (s *StructType) Dump() string {
	return s.tree.dump("")
}

// String dump the id and selector on the field subtree.
func (f *FieldType) String() string {
	return f.Dump()
}

// Dump dump the id and selector on the field subtree.
func (f *FieldType) Dump() string {
	return f.dump("")
}

func (f *FieldType) dump(prefix string) string {
	var buf bytes.Buffer
	if f.id != rootID {
		buf.WriteString(fmt.Sprintf("%sid=%d selector=%s\n", prefix, f.id, f.selector))
		prefix += "····"
	}
	for _, child := range f.children {
		buf.WriteString(child.dump(prefix))
	}
	return buf.String()
}

// Deep get the nesting depth of the field.
func (f *FieldType) Deep() int {
	return f.deep
}

// Kind get the field kind.
func (f *FieldType) Kind() reflect.Kind {
	return f.StructField.Type.Kind()
}

// UnderlyingKind get the underlying kind of the field
func (f *FieldType) UnderlyingKind() reflect.Kind {
	return f.elemTyp.Kind()
}

func (a *Accessor) newStructType(tid int32, structPtr interface{}) *StructType {
	v, ok := structPtr.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(structPtr)
	}
	v = ameda.DereferencePtrValue(v)
	structTyp := v.Type()
	sTyp := &StructType{
		tid:    tid,
		fields: make([]*FieldType, 0, 16),
		tree:   &FieldType{id: rootID, elemTyp: structTyp},
	}
	var structID int
	sTyp.traversalFields(&structID, a.maxDeep, a.iterator, sTyp.tree)
	sTyp.structNum = structID + 1
	if a.groupBy != nil {
		sTyp.groupBy(a.groupBy)
	}
	return sTyp
}

func (s *StructType) traversalFields(structID *int, maxFieldDeep int, iterator IteratorFunc, parent *FieldType) {
	if s.depth >= maxFieldDeep {
		return
	}
	s.depth++
	structTyp := parent.elemTyp
	numField := structTyp.NumField()
	var structFields []*FieldType
L:
	for i := 0; i < numField; i++ {
		f := structTyp.Field(i)
		elemTyp := f.Type
		var ptrNum int
		for elemTyp.Kind() == reflect.Ptr {
			elemTyp = elemTyp.Elem()
			ptrNum++
		}
		_ = reflect.PtrTo(elemTyp)
		_ = reflect.PtrTo(f.Type)
		elemVal := reflect.New(elemTyp)
		rawVal := reflect.New(f.Type)
		field := &FieldType{
			parent:      parent,
			id:          len(s.fields), // 0, 1, 2, ...
			selector:    joinFieldName(parent.selector, f.Name),
			deep:        s.depth,
			ptrNum:      ptrNum,
			elemTyp:     elemTyp,
			elemVal:     *(*reflectValue)(unsafe.Pointer(&elemVal)),
			rawVal:      *(*reflectValue)(unsafe.Pointer(&rawVal)),
			StructField: f,
		}
		isStruct := elemTyp.Kind() == reflect.Struct
		if isStruct {
			*structID++
			field.structID = *structID
		}
		if iterator != nil {
			switch p := iterator(field); p {
			default:
				fallthrough
			case Take, TakeAndStop:
				parent.children = append(parent.children, field)
				s.fields = append(s.fields, field)
				if isStruct {
					structFields = append(structFields, field)
				}
				if TakeAndStop == p {
					break L
				}
			case SkipOffspring, SkipOffspringAndStop:
				parent.children = append(parent.children, field)
				s.fields = append(s.fields, field)
				if SkipOffspringAndStop == p {
					break L
				}
			case Skip:
				continue L
			case SkipAndStop:
				break L
			}
		} else {
			parent.children = append(parent.children, field)
			s.fields = append(s.fields, field)
			if isStruct {
				structFields = append(structFields, field)
			}
		}
	}
	for _, field := range structFields {
		s.traversalFields(structID, maxFieldDeep, iterator, field)
	}
}

func (s *StructType) groupBy(fn GroupByFunc) {
	s.fieldGroup = make(map[string][]*FieldType, len(s.fields))
	for _, field := range s.fields {
		group, ok := fn(field)
		if ok {
			a := s.fieldGroup[group]
			s.fieldGroup[group] = append(a, field)
		}
	}
}

// FieldTree return the field tree.
func (s *StructType) FieldTree() []*FieldType {
	return s.tree.children
}

// Parent return the parent field.
// NOTE:
//  may return nil
func (f *FieldType) Parent() *FieldType {
	if f.parent == nil || f.parent.id == rootID {
		return nil
	}
	return f.parent
}

// Children return the child fields.
func (f *FieldType) Children() []*FieldType {
	return f.children
}

func joinFieldName(parentPath, name string) string {
	return parentPath + "." + name
}

package gofield

import (
	"bytes"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/henrylee2cn/ameda"
)

type (
	// StructType struct type info
	StructType struct {
		tid        int32
		fields     []*FieldType
		fieldGroup map[string][]*FieldType
		depth      int
		tree       *FieldType // id = -1
		structNum  int
	}
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
	reflectValue struct {
		typ  *uintptr
		ptr  unsafe.Pointer
		flag uintptr
	}
)

func newStructType(a *Accessor, tid int32, structPtr interface{}) *StructType {
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

func joinFieldName(parentPath, name string) string {
	return parentPath + "." + name
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

func (s *StructType) checkID(id int) bool {
	return id >= 0 && id < len(s.fields)
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

// GroupTypes return the field types by group.
func (s *StructType) GroupTypes(group string) []*FieldType {
	a := s.fieldGroup[group]
	return a
}

// FieldTree return the field tree.
func (s *StructType) FieldTree() []*FieldType {
	return s.tree.children
}

// String dump the id and selector on the field tree.
func (s *StructType) String() string {
	return s.Dump()
}

// Dump dump the id and selector on the field tree.
func (s *StructType) Dump() string {
	return s.tree.dump("")
}

// ID get the field id.
func (f *FieldType) ID() int {
	return f.id
}

// Selector get the field full path.
func (f *FieldType) Selector() string {
	return f.selector
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

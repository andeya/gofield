package gofield

import (
	"reflect"
	"sync"

	"github.com/henrylee2cn/ameda"
)

type (
	// Mapper struct fields map factory
	Mapper struct {
		structTypes map[int32]StructMap // key:runtime type id
		keyMaker    FieldKeyFunc
		mutex       sync.RWMutex
	}
	// StructMap struct fields map info
	StructMap struct {
		t *StructType
		m map[string]FieldID
	}
	// FieldKeyFunc create the key of the field map
	FieldKeyFunc func(*FieldType) (string, bool)
)

// NewMapper create a struct fields map factory.
//go:nosplit
func NewMapper(fn FieldKeyFunc) *Mapper {
	b := &Mapper{
		structTypes: make(map[int32]StructMap, 16),
		keyMaker:    fn,
	}
	return b
}

// MustMake make the fields map of the structPtr.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func (b *Mapper) MustMake(structPtr interface{}) map[string]reflect.Value {
	m, err := b.Make(structPtr)
	if err != nil {
		panic(err)
	}
	return m
}

// Make make the fields map of the structPtr.
//go:nosplit
func (b *Mapper) Make(structPtr interface{}) (map[string]reflect.Value, error) {
	tid := ameda.RuntimeTypeIDOf(structPtr)
	b.mutex.RLock()
	m, ok := b.structTypes[tid]
	b.mutex.RUnlock()
	if ok {
		return m.Make(structPtr)
	}
	m, s, err := b.initStruct(tid, structPtr)
	if err != nil {
		return nil, err
	}
	return m.parse(s), nil
}

// MustMake make the fields map of the structPtr.
// NOTE:
//  If structPtr is not a struct pointer or type mismatch, it will cause panic.
//go:nosplit
func (m *StructMap) MustMake(structPtr interface{}) map[string]reflect.Value {
	r, err := m.Make(structPtr)
	if err != nil {
		panic(err)
	}
	return r
}

// Make make the fields map of the structPtr.
//go:nosplit
func (m *StructMap) Make(structPtr interface{}) (map[string]reflect.Value, error) {
	tid, ptr, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	if m.t.tid != tid {
		return nil, errTypeMismatch
	}
	s := newStruct(m.t, ptr)
	return m.parse(s), nil
}

//go:nosplit
func (m *StructMap) parse(s *Struct) map[string]reflect.Value {
	num := s.NumField()
	r := make(map[string]reflect.Value, num)
	for key, fid := range m.m {
		r[key] = s.FieldValue(fid)
	}
	return r
}

//go:nosplit
func (b *Mapper) initStruct(tid int32, structPtr interface{}) (StructMap, *Struct, error) {
	ptr, err := parseStructPtrWithCheck(structPtr)
	if err != nil {
		return StructMap{}, nil, err
	}
	t := analyze(tid, structPtr)
	s := newStruct(t, ptr)
	num := s.NumField()
	structMap := StructMap{
		t: t,
		m: make(map[string]FieldID, num),
	}
	for i := 0; i < num; i++ {
		ft := s.FieldType(i)
		key, ok := b.keyMaker(ft)
		if ok {
			structMap.m[key] = ft.ID()
		}
	}

	b.mutex.Lock()
	b.structTypes[s.RuntimeTypeID()] = structMap
	b.mutex.Unlock()

	return structMap, s, nil
}

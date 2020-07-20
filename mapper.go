package gofield

import (
	"reflect"
	"sync"

	"github.com/henrylee2cn/ameda"
)

type (
	Mapper struct {
		structTypes map[int32]StructMap // key:runtime type id
		keyMaker    KeyMaker
		mutex       sync.RWMutex
	}
	StructMap struct {
		t *StructType
		m map[string]FieldID
	}
	KeyMaker func(*FieldType) (string, bool)
)

//go:nosplit
func NewMapper(fn KeyMaker) *Mapper {
	b := &Mapper{
		structTypes: make(map[int32]StructMap, 16),
		keyMaker:    fn,
	}
	return b
}

//go:nosplit
func (b *Mapper) MustGet(structPtr interface{}) map[string]reflect.Value {
	m, err := b.Get(structPtr)
	if err != nil {
		panic(err)
	}
	return m
}

//go:nosplit
func (b *Mapper) Get(structPtr interface{}) (map[string]reflect.Value, error) {
	tid := ameda.RuntimeTypeIDOf(structPtr)
	b.mutex.RLock()
	m, ok := b.structTypes[tid]
	b.mutex.RUnlock()
	if ok {
		return m.Get(structPtr)
	}
	m, s, err := b.initStruct(tid, structPtr)
	if err != nil {
		return nil, err
	}
	return m.get(s), nil
}

// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
//go:nosplit
func (m *StructMap) MustGet(structPtr interface{}) map[string]reflect.Value {
	r, err := m.Get(structPtr)
	if err != nil {
		panic(err)
	}
	return r
}

//go:nosplit
func (m *StructMap) Get(structPtr interface{}) (map[string]reflect.Value, error) {
	tid, ptr, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	if m.t.tid != tid {
		return nil, errTypeMismatch
	}
	s := newStruct(m.t, ptr)
	return m.get(s), nil
}

//go:nosplit
func (m *StructMap) get(s *Struct) map[string]reflect.Value {
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

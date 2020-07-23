package gofield

import (
	"errors"
	"reflect"
	"sync"
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

// MustAnalyze analyze the struct and return its type info.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func MustAnalyze(structPtr interface{}) *StructType {
	return defaultAccessor.MustAnalyze(structPtr)
}

// Analyze analyze the struct and return its type info.
func Analyze(structPtr interface{}) (*StructType, error) {
	return defaultAccessor.Analyze(structPtr)
}

// MustAccess analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func MustAccess(structPtr interface{}) *Struct {
	return defaultAccessor.MustAccess(structPtr)
}

// Access analyze the struct type info and create struct accessor.
func Access(structPtr interface{}) (*Struct, error) {
	return defaultAccessor.Access(structPtr)
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
		sTyp = newStructType(a, tid, structPtr)
		a.store(sTyp)
	}
	return sTyp
}

// MustAccess analyze the struct type info and create struct accessor.
// NOTE:
//  If structPtr is not a struct pointer, it will cause panic.
func (a *Accessor) MustAccess(structPtr interface{}) *Struct {
	tid, ptr := parseStructInfo(structPtr)
	sTyp, ok := a.load(tid)
	if !ok {
		sTyp = newStructType(a, tid, structPtr)
		a.store(sTyp)
	}
	return newStruct(sTyp, ptr)
}

// Access analyze the struct type info and create struct accessor.
func (a *Accessor) Access(structPtr interface{}) (*Struct, error) {
	tid, ptr, err := parseStructInfoWithCheck(structPtr)
	if err != nil {
		return nil, err
	}
	sTyp, ok := a.load(tid)
	if !ok {
		sTyp = newStructType(a, tid, structPtr)
		a.store(sTyp)
	}
	return newStruct(sTyp, ptr), nil
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

package gofield

type (
	// Option accessor option
	Option func(*Accessor)
	// GroupByFunc create the group of the field type
	GroupByFunc func(*FieldType) (string, bool)
	// IteratorFunc determine whether the field needs to be iterated.
	IteratorFunc func(*FieldType) IterPolicy
	// IterPolicy iteration policy
	IterPolicy int8
)

const (
	// Take take the field
	Take IterPolicy = iota
	// SkipSelf skip the field, but not skip its subfields
	SkipSelf
	// SkipOffspring accept the field, but skip its subfields
	SkipOffspring
	// Skip skip the field and its subfields
	Skip
	// Stop stop iteration
	Stop
)

// WithGroupBy set GroupByFunc to *Accessor.
//go:nosplit
func WithGroupBy(fn GroupByFunc) Option {
	return func(a *Accessor) {
		a.groupBy = fn
	}
}

// WithIterator set IteratorFunc to *Accessor.
//go:nosplit
func WithIterator(fn IteratorFunc) Option {
	return func(a *Accessor) {
		a.iterator = fn
	}
}

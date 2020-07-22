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
	// SkipOffspring take the field, but skip its subfields
	SkipOffspring
	// Skip skip the field and its subfields
	Skip
	// TakeAndStop take the field and stop iteration
	TakeAndStop
	// SkipOffspringAndStop take the field, but skip its subfields, and stop iteration
	SkipOffspringAndStop
	// SkipAndStop skip the field and its subfields, and stop iteration
	SkipAndStop
)

// WithGroupBy set GroupByFunc to *Accessor.
func WithGroupBy(fn GroupByFunc) Option {
	return func(a *Accessor) {
		a.groupBy = fn
	}
}

// WithIterator set IteratorFunc to *Accessor.
func WithIterator(fn IteratorFunc) Option {
	return func(a *Accessor) {
		a.iterator = fn
	}
}

// WithMaxDeep set the maximum traversal depth.
func WithMaxDeep(maxDeep int) Option {
	return func(a *Accessor) {
		a.maxDeep = maxDeep
	}
}

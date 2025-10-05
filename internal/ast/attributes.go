package ast

import (
	"iter"
	"slices"
)

// Attribute is an arbitrary key-value pair associated with a [Node].
type Attribute struct {
	Key   string
	Value any
}

// AttributeMap is a collection of key-value attributes for a [Node]. Unlike
// a native Go map, insertion order is preserved.
type AttributeMap struct {
	attrs []*Attribute
}

// Get searches for an attribute with the given key. If finds one, it returns
// its value, and the second return value will be true. If no matching
// attribute is found, `value` will be `nil` and `ok` will be `false`.
func (a *AttributeMap) Get(key string) (value any, ok bool) {
	_, attr := a.attr(key)
	if attr != nil {
		return attr.Value, true
	} else {
		return nil, false
	}
}

// Set updates or inserts an attribute with the given key, and sets it to the
// given value. It returns the modified [AttributeMap].
func (a *AttributeMap) Set(key string, value any) *AttributeMap {
	_, attr := a.attr(key)
	if attr == nil {
		attr = &Attribute{Key: key}
		a.attrs = append(a.attrs, attr)
	}

	attr.Value = value

	return a
}

// Delete removes the attribute with the given key if it exists. It returns true
// if an attribute was deleted, and false otherwise.
func (a *AttributeMap) Delete(key string) bool {
	i, _ := a.attr(key)
	if i == -1 {
		return false
	}

	a.attrs = slices.Delete(a.attrs, i, i+i)
	return true
}

// Size returns the number of attributes in the map.
func (a *AttributeMap) Size() int {
	return len(a.attrs)
}

// Keys returns an iterator over the map's keys.
func (a *AttributeMap) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, entry := range a.attrs {
			if !yield(entry.Key) {
				return
			}
		}
	}
}

// Values returns an iterator over the map's values.
func (a *AttributeMap) Values() iter.Seq[any] {
	return func(yield func(any) bool) {
		for _, entry := range a.attrs {
			if !yield(entry.Value) {
				return
			}
		}
	}
}

// Entries returns an iterator over the map's key-value pairs.
func (a *AttributeMap) Entries() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for _, entry := range a.attrs {
			if !yield(entry.Key, entry.Value) {
				return
			}
		}
	}
}

// ToMap returns the attribute map as a native Go map.
func (a *AttributeMap) ToMap() map[string]any {
	m := make(map[string]any)

	for _, attr := range a.attrs {
		m[attr.Key] = attr.Value
	}

	return m
}

// attr searches for an attribute by key and, if it finds it, returns its index
// and the [Attribute] itself. If no attribute with the given key exists, attr
// returns -1, nil.
func (a *AttributeMap) attr(key string) (index int, attr *Attribute) {
	for i, attr := range a.attrs {
		if attr.Key == key {
			return i, attr
		}
	}

	return -1, nil
}

func NewAttributeMap(init ...*Attribute) *AttributeMap {
	return &AttributeMap{
		attrs: init,
	}
}

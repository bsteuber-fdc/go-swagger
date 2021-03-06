package strfmt

import (
	"encoding"
	"reflect"
	"strings"
	"sync"

	"github.com/go-swagger/go-swagger/errors"
)

// Default is the default formats registry
var Default = NewSeededFormats(nil, nil)

// Validator represents a validator for a string format
type Validator func(string) bool

// Format represents a string format
type Format interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

// Registry is a registry of string formats
type Registry interface {
	Add(string, Format, Validator) bool
	DelByName(string) bool
	GetType(string) (reflect.Type, bool)
	ContainsName(string) bool
	Validates(string, string) bool
	Parse(string, string) (interface{}, error)
}

type knownFormat struct {
	Name      string
	OrigName  string
	Type      reflect.Type
	Validator Validator
}

// NameNormalizer is a function that normalizes a format name
type NameNormalizer func(string) string

// DefaultNameNormalizer removes all dashes
func DefaultNameNormalizer(name string) string {
	return strings.Replace(name, "-", "", -1)
}

type defaultFormats struct {
	sync.Mutex
	data          []knownFormat
	normalizeName NameNormalizer
}

// NewFormats creates a new formats registry seeded with the values from the default
func NewFormats() Registry {
	return NewSeededFormats(Default.(*defaultFormats).data, nil)
}

// NewSeededFormats creates a new formats registry
func NewSeededFormats(seeds []knownFormat, normalizer NameNormalizer) Registry {
	if normalizer == nil {
		normalizer = DefaultNameNormalizer
	}
	// copy here, don't modify original
	d := append([]knownFormat(nil), seeds...)
	return &defaultFormats{
		data:          d,
		normalizeName: normalizer,
	}
}

// Add adds a new format, return true if this was a new item instead of a replacement
func (f *defaultFormats) Add(name string, strfmt Format, validator Validator) bool {
	f.Lock()
	defer f.Unlock()

	nme := f.normalizeName(name)

	tpe := reflect.TypeOf(strfmt)
	if tpe.Kind() == reflect.Ptr {
		tpe = tpe.Elem()
	}

	for i := range f.data {
		v := &f.data[i]
		if v.Name == nme {
			v.Type = tpe
			v.Validator = validator
			return false
		}
	}

	// turns out it's new after all
	f.data = append(f.data, knownFormat{Name: nme, OrigName: name, Type: tpe, Validator: validator})
	return true
}

// GetType gets the type for the specified name
func (f *defaultFormats) GetType(name string) (reflect.Type, bool) {
	nme := f.normalizeName(name)
	for _, v := range f.data {
		if v.Name == nme {
			return v.Type, true
		}
	}
	return nil, false
}

// DelByName removes the format by the specified name, returns true when an item was actually removed
func (f *defaultFormats) DelByName(name string) bool {
	f.Lock()
	defer f.Unlock()

	nme := f.normalizeName(name)

	for i, v := range f.data {
		if v.Name == nme {
			f.data[i] = knownFormat{} // release
			f.data = append(f.data[:i], f.data[i+1:]...)
			return true
		}
	}
	return false
}

// DelByType removes the specified format, returns true when an item was actually removed
func (f *defaultFormats) DelByFormat(strfmt Format) bool {
	f.Lock()
	defer f.Unlock()

	tpe := reflect.TypeOf(strfmt)
	if tpe.Kind() == reflect.Ptr {
		tpe = tpe.Elem()
	}

	for i, v := range f.data {
		if v.Type == tpe {
			f.data[i] = knownFormat{} // release
			f.data = append(f.data[:i], f.data[i+1:]...)
			return true
		}
	}
	return false
}

// ContainsName returns true if this registry contains the specified name
func (f *defaultFormats) ContainsName(name string) bool {
	nme := f.normalizeName(name)
	for _, v := range f.data {
		if v.Name == nme {
			return true
		}
	}
	return false
}

// ContainsFormat returns true if this registry contains the specified format
func (f *defaultFormats) ContainsFormat(strfmt Format) bool {
	tpe := reflect.TypeOf(strfmt)
	if tpe.Kind() == reflect.Ptr {
		tpe = tpe.Elem()
	}

	for _, v := range f.data {
		if v.Type == tpe {
			return true
		}
	}
	return false
}

func (f *defaultFormats) Validates(name, data string) bool {
	nme := f.normalizeName(name)
	for _, v := range f.data {
		if v.Name == nme {
			return v.Validator(data)
		}
	}
	return false
}

func (f *defaultFormats) Parse(name, data string) (interface{}, error) {
	nme := f.normalizeName(name)
	for _, v := range f.data {
		if v.Name == nme {
			nw := reflect.New(v.Type).Interface()
			if dec, ok := nw.(encoding.TextUnmarshaler); ok {
				if err := dec.UnmarshalText([]byte(data)); err != nil {
					return nil, err
				}
				return nw, nil
			}
			return nil, errors.InvalidTypeName(name)
		}
	}
	return nil, errors.InvalidTypeName(name)
}

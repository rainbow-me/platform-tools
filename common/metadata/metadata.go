package metadata

import (
	"strings"
)

type Metadata map[string][]string

func NewMetadata(m map[string]string) Metadata {
	md := make(Metadata, len(m))
	for k, val := range m {
		key := strings.ToLower(k)
		md[key] = append(md[key], val)
	}
	return md
}

func (md Metadata) Copy() Metadata {
	out := make(Metadata, len(md))
	for k, v := range md {
		out[k] = copyOf(v)
	}
	return out
}

// Get obtains the values for a given key.
//
// k is converted to lowercase before searching in md.
func (md Metadata) Get(k string) []string {
	k = strings.ToLower(k)
	return md[k]
}

// Set sets the value of a given key with a slice of values.
//
// k is converted to lowercase before storing in md.
func (md Metadata) Set(k string, vals ...string) {
	if len(vals) == 0 {
		return
	}
	k = strings.ToLower(k)
	md[k] = vals
}

// Append adds the values to key k, not overwriting what was already stored at
// that key.
//
// k is converted to lowercase before storing in md.
func (md Metadata) Append(k string, vals ...string) {
	if len(vals) == 0 {
		return
	}
	k = strings.ToLower(k)
	md[k] = append(md[k], vals...)
}

// Delete removes the values for a given key k which is converted to lowercase
// before removing it from md.
func (md Metadata) Delete(k string) {
	k = strings.ToLower(k)
	delete(md, k)
}

func copyOf(v []string) []string {
	vals := make([]string, len(v))
	copy(vals, v)
	return vals
}

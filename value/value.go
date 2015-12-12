// Package value defines the common interface shared by caches to represent
// cached values.
package value

// Interface defines the required behaviour of a cached value, which is to
// return its nominal size as a non-negative integer.  The units of size are
// dependent on the cache: Values in a cache whose capacity is a number of
// elements will have size 1.  Values in a cache whose capacity is in bytes
// will return byte counts.
type Interface interface {
	// Size returns a non-negative integer expressing the size of the value.
	// If a negative value is returned, cache operations will panic.
	Size() int
}

// String is a convenience wrapper for storing a string as a cache value.
// Its size is the length of the string in bytes.
type String string

func (s String) Size() int { return len(s) }

// Bytes is a convenience wrapper for storing a byte slice as a cache value.
// Its size is the length of the slice.
type Bytes []byte

func (b Bytes) Size() int { return len(b) }

// Nil is a placeholder value to use in a cache where the keys are the values
// being cached.  Nil has size 1.
const Nil = nilValue(0)

type nilValue byte

func (nilValue) Size() int { return 1 }

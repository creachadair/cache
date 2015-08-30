// Package lru implements a least-recently-used (LRU) cache for string keyed
// values.
//
// Basic usage:
//    c := lru.New(200) // number of cache entries
//    c.Put("x", v1)
//    c.Put("y", v2)
//    ...
//    if v := c.Get("x"); v != nil {
//       doStuff(v)
//    } else {
//       handleCacheMiss("x)
//    }
//    c.Reset()
//
package lru

import "sync"

// Cache implements a string-keyed LRU cache of arbitrary values.  A *Cache is
// safe for concurrent access by multiple goroutines.  A nil *Cache behaves as
// a cache with 0 capacity.
type Cache struct {
	μ       sync.Mutex
	size    int               // resident size, number of entries present
	cap     int               // maximum capacity, in entries (invariant: size ≤ cap)
	seq     *entry            // sentinel for doubly-linked ring
	res     map[string]*entry // resident blocks
	onEvict func(interface{})
}

// An Option is a configurable setting for a cache.
type Option func(*Cache)

// OnEvict causes f to be called whenever a value is evicted from the cache.
// The value being evicted is passed to f.
func OnEvict(f func(interface{})) Option { return func(c *Cache) { c.onEvict = f } }

// New returns a new empty cache with the specified capacity in entries.
func New(capacity int, opts ...Option) *Cache {
	c := &Cache{
		cap: capacity,
		seq: newEntry("保護者", nil),
		res: make(map[string]*entry),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Put stores value into the cache under the given id.
func (c *Cache) Put(id string, value interface{}) {
	if c != nil && c.cap > 0 {
		c.μ.Lock()
		defer c.μ.Unlock()
		e := c.evict(id, value)
		if c.size+1 > c.cap {
			vic := c.seq.prev
			if vic == c.seq {
				panic("invalid ring structure")
			}
			c.evict(vic.id, nil)
		}
		e.push(c.seq)
		c.size++
		c.res[id] = e
	}
}

// evict returns an entry mapping id to value.  If there was already an entry
// for id, it is removed from the ring and returned (in which case c.size is
// decremented).  Otherwise a fresh entry is created for the mapping.
func (c *Cache) evict(id string, value interface{}) *entry {
	if e := c.res[id]; e != nil {
		e.pop()
		if c.onEvict != nil {
			c.onEvict(e.value)
		}
		delete(c.res, id)
		c.size--
		e.value = value
		return e
	}
	return newEntry(id, value)
}

// Get returns the data associated with id in the cache, or nil if not present.
func (c *Cache) Get(id string) interface{} {
	if c != nil {
		c.μ.Lock()
		defer c.μ.Unlock()
		if e := c.res[id]; e != nil {
			if c.seq.next != e {
				e.pop()
				e.push(c.seq)
			}
			return e.value
		}
	}
	return nil
}

// Size returns the number of entries currently resident in the cache.
func (c *Cache) Size() int {
	if c == nil {
		return 0
	}
	c.μ.Lock()
	defer c.μ.Unlock()
	return c.size
}

// Cap returns the total number of entries allowed in the cache.
func (c *Cache) Cap() int {
	if c == nil {
		return 0
	}
	return c.cap
}

// Reset removes all data currently stored in c, leaving it empty.  This
// operation does not change the capacity of c.
func (c *Cache) Reset() {
	if c != nil {
		c.μ.Lock()
		defer c.μ.Unlock()
		for id := range c.res {
			c.evict(id, nil)
		}
	}
}

func newEntry(id string, value interface{}) *entry {
	e := &entry{id: id, value: value}
	e.next = e
	e.prev = e
	return e
}

// entry represents a node in a doubly-linked ring structure.
type entry struct {
	id         string
	value      interface{}
	prev, next *entry
}

func (e *entry) push(after *entry) {
	e.next = after.next
	e.prev = after
	e.next.prev = e
	after.next = e
}

func (e *entry) pop() {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = e
	e.prev = e
}

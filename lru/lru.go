// Package lru implements a least-recently-used (LRU) cache for string keyed
// byte buffers.
//
// Basic usage:
//    c := lru.New(8<<20)
//    c.Put("x", v1)
//    c.Put("y", v2)
//    ...
//    if v := c.Get("x"); v != nil {
//      doStuff(v)
//    } else {
//       handleCacheMiss("x)
//    }
//    c.Reset()
//
package lru

import "sync"

// Cache implements a string-keyed LRU cache of byte buffers.  A *Cache is safe
// for concurrent access by multiple goroutines.
type Cache struct {
	μ    sync.Mutex
	size int               // resident size, in bytes
	cap  int               // maximum capacity, in bytes (invariant: size ≤ cap)
	seq  *entry            // sentinel for doubly-linked ring
	res  map[string]*entry // resident blocks
}

// New returns a new empty cache with the specified capacity in bytes.
func New(capacity int) *Cache {
	return &Cache{
		cap: capacity,
		seq: newEntry("保護者", nil),
		res: make(map[string]*entry),
	}
}

// Put stores of data into the cache under the given id.
func (c *Cache) Put(id string, data []byte) {
	c.μ.Lock()
	defer c.μ.Unlock()
	if len(data) > c.cap {
		return // no room for this block no matter what
	}
	e := c.evict(id, data)
	for c.size+len(data) > c.cap {
		vic := c.seq.prev
		if vic == c.seq {
			panic("invalid ring structure")
		}
		c.evict(vic.id, nil)
	}
	e.push(c.seq)
	c.size += len(data)
	c.res[id] = e
}

func (c *Cache) evict(id string, data []byte) *entry {
	if e := c.res[id]; e != nil {
		e.pop()
		delete(c.res, id)
		c.size -= len(e.data)
		e.data = data
		return e
	}
	return newEntry(id, data)
}

// Get returns the data associated with id in the cache, or nil if not present.
func (c *Cache) Get(id string) []byte {
	c.μ.Lock()
	defer c.μ.Unlock()
	if e := c.res[id]; e != nil {
		if c.seq.next != e {
			e.pop()
			e.push(c.seq)
		}
		return e.data
	}
	return nil
}

// Size returns the current resident size of the cached data, in bytes.
func (c *Cache) Size() int {
	c.μ.Lock()
	defer c.μ.Unlock()
	return c.size
}

// Reset removes all data currently stored in c, leaving it empty.
// This operation does not change the capacity of c.
func (c *Cache) Reset() {
	c.μ.Lock()
	defer c.μ.Unlock()
	for id := range c.res {
		c.evict(id, nil)
	}
}

func newEntry(id string, data []byte) *entry {
	e := &entry{id: id, data: data}
	e.next = e
	e.prev = e
	return e
}

type entry struct {
	id         string
	data       []byte
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

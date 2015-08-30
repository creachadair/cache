// Package lfu implements a least-frequently-used (LFU) cache for string keyed
// values.
//
// Basic usage:
//    c := lfu.New(200) // number of cache entries
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
package lfu

import "sync"

// Cache implements a string-keyed LFU cache of arbitrary values.  A *Cache is
// safe for concurrent access by multiple goroutines.  A nil *Cache behaves as
// a cache with 0 capacity.
type Cache struct {
	μ       sync.Mutex
	size    int            // resident size, number of entries present
	cap     int            // maximum capacity, in entries (invariant: size ≤ cap)
	heap    []*entry       // min-heap by frequency of use
	res     map[string]int // resident blocks, id → heap-index
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
		res: make(map[string]int),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Put stores value into the cache under the given id.  A Put counts as a use
// on first insertion, but not subsequently.
func (c *Cache) Put(id string, value interface{}) {
	if c != nil && c.cap > 0 {
		c.μ.Lock()
		defer c.μ.Unlock()
		pos, ok := c.res[id]
		if !ok {
			if c.size+1 > c.cap {
				c.evict()
			}
			c.add(id, value)
			c.size++
			return
		}
		cur := c.heap[pos]
		if c.onEvict != nil {
			c.onEvict(cur.value)
		}
		cur.value = value
	}
}

// Get returns the data associated with id in the cache, or nil if not present.
func (c *Cache) Get(id string) interface{} {
	if c != nil {
		c.μ.Lock()
		defer c.μ.Unlock()
		if pos, ok := c.res[id]; ok {
			elt := c.heap[pos]
			elt.uses++
			c.fix(pos)
			return elt.value
		}
	}
	return nil
}

// Size returns the number of entries currently resident in the cache.
func (c *Cache) Size() int {
	if c != nil {
		c.μ.Lock()
		defer c.μ.Unlock()
		return c.size
	}
	return 0
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
		for c.size > 0 {
			c.evict()
		}
	}
}

// entry represents a node in a min-heap by frequency of use.
type entry struct {
	id    string
	value interface{}
	uses  int
}

// add inserts a new entry into the cache mapping id to value.  Assumes id is
// not already resident, and that c.μ is held.
func (c *Cache) add(id string, value interface{}) {
	pos := len(c.heap)
	elt := &entry{id: id, value: value, uses: 1}
	c.heap = append(c.heap, elt)
	for pos > 0 {
		par := pos / 2
		if up := c.heap[par]; up.uses > 1 {
			c.heap[par] = elt
			c.heap[pos] = up
			c.res[up.id] = pos
			pos = par
			continue
		}
		break
	}
	c.res[id] = pos
}

// evict removes the least-frequently used element from the cache, calling the
// eviction handler if necessary for its value.  Assumes that c.μ is held.
func (c *Cache) evict() {
	vic := c.heap[0]
	if c.onEvict != nil {
		c.onEvict(vic.value)
	}
	delete(c.res, vic.id)
	n := len(c.heap) - 1
	c.heap[0] = c.heap[n]
	c.heap = c.heap[:n]
	c.fix(0)
	c.size--
}

// fix restores heap order to c.heap at or below pos, assuming that the weight
// of pos has remained the same or increased.  Assumes c.μ is held.
func (c *Cache) fix(pos int) {
	for {
		mc := 2 * pos
		if mc >= len(c.heap) {
			return
		} else if rc := mc + 1; rc < len(c.heap) && c.heap[rc].uses < c.heap[mc].uses {
			mc = rc
		}
		cur := c.heap[pos]
		min := c.heap[mc]
		if cur.uses <= min.uses {
			return
		}
		c.heap[pos] = min
		c.res[min.id] = pos
		c.heap[mc] = cur
		c.res[cur.id] = mc
		pos = mc
	}
}

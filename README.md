# cache

This repository provides packages to implement string-keyed in-memory caching.

Package [lru](http://godoc.org/github.com/creachadair/cache/lru) implements
a cache with a least-recently used (LRU) replacement policy.

Package [lfu](http://godoc.org/github.com/creachadair/cache/lfu) implements
a cache with a least-frequently used (LFU) replacement policy.

The capacity of a cache is specified in user-defined units.  Values stored in
the cache report their size by implementing value.Interface, and may use any
non-negative metric (typically number of entries or size in bytes will make the
most sense).

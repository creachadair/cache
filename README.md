# cache

This repository provides packages to implement string-keyed in-memory caching.

Package [lru](http://godoc.org/bitbucket.org/creachadair/cache/lru) implements
a cache with a least-recently used (LRU) replacement policy.

Package [lfu](http://godoc.org/bitbucket.org/creachadair/cache/lfu) implements
a cache with a least-frequently used (LFU) replacement policy.

Both caches are bounded in size by a number of entries.

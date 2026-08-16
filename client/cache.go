package etagclient

import (
	"net/http"
	"sync"
)

// Stats reports conditional-cache activity.
type Stats struct {
	// Hits is the number of 304 responses converted to cached 200s.
	Hits int64
	// Stored is the number of 200 responses added to the cache.
	Stored int64
	// Entries is the current number of cached responses.
	Entries int
}

// cacheEntry is one stored response: its validator, header map, and body.
type cacheEntry struct {
	etag   string
	status int
	header http.Header
	body   []byte
}

// responseCache is an in-memory conditional GET store, safe for concurrent
// use, bounded by FIFO eviction of the oldest entry.
type responseCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	order   []string
	max     int
	hits    int64
	stored  int64
}

func newResponseCache(maxEntries int) *responseCache {
	return &responseCache{
		mu:      sync.Mutex{},
		entries: make(map[string]cacheEntry, maxEntries),
		order:   nil,
		max:     maxEntries,
		hits:    0,
		stored:  0,
	}
}

func (c *responseCache) get(key string) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]

	return entry, ok
}

func (c *responseCache) set(key string, entry cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; !exists {
		c.order = append(c.order, key)

		for len(c.order) > c.max {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.entries, oldest)
		}
	}

	c.entries[key] = entry
	c.stored++
}

func (c *responseCache) countHit() {
	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
}

func (c *responseCache) stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return Stats{Hits: c.hits, Stored: c.stored, Entries: len(c.entries)}
}

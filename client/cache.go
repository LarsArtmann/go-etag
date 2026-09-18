package etagclient

import (
	"net/http"
	"sync"

	"github.com/larsartmann/go-etag/entitytag"
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

// storedValidator is the revalidation identity of a stored response: the
// wire-format entity-tag value replayed verbatim as If-None-Match, plus the
// parsed RFC 7232 §2.3 form computed once at store time. An unparseable
// field value (a broken server) stays representable — it is replayed as-is
// and never weak-matches another validator, so it can neither confirm stored
// state (RFC 9111 §4.3.5) nor have its claim adopted from a 304 (§4.3.4).
type storedValidator struct {
	wire   string
	parsed entitytag.ETag
	valid  bool
}

// newStoredValidator parses a wire-format entity-tag value once so every
// later comparison reuses the parsed form instead of re-parsing stored state.
func newStoredValidator(wire string) storedValidator {
	parsed, ok := entitytag.ParseETag(wire)

	return storedValidator{wire: wire, parsed: parsed, valid: ok}
}

// weaklyMatches reports whether candidate, a wire-format entity-tag field
// value, carries the same opaque tag as the stored validator, ignoring
// strength (the RFC 9110 §8.8.3.2 weak comparison If-None-Match uses). A
// value that does not parse as an RFC 7232 §2.3 entity-tag on either side
// can never match.
func (v storedValidator) weaklyMatches(candidate string) bool {
	parsedCandidate, ok := entitytag.ParseETag(candidate)
	if !ok || !v.valid {
		return false
	}

	return v.parsed.WeakEqual(parsedCandidate)
}

// storedResponse is one cached response: its revalidation identity, header
// map, body, and whether net/http transparently decompressed that body
// (which exempts Content-Encoding from freshening; RFC 9111 §3.2).
type storedResponse struct {
	validator    storedValidator
	status       int
	header       http.Header
	body         []byte
	uncompressed bool
}

// responseCache is an in-memory conditional GET store, safe for concurrent
// use, bounded by FIFO eviction of the oldest entry.
type responseCache struct {
	mu      sync.Mutex
	entries map[string]storedResponse
	order   []string
	max     int
	hits    int64
	stored  int64
}

func newResponseCache(maxEntries int) *responseCache {
	return &responseCache{
		mu:      sync.Mutex{},
		entries: make(map[string]storedResponse, maxEntries),
		order:   nil,
		max:     maxEntries,
		hits:    0,
		stored:  0,
	}
}

func (c *responseCache) get(key string) (storedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]

	return entry, ok
}

func (c *responseCache) set(key string, entry storedResponse) {
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

// freshen replaces a stored entry when the key still maps to the response
// that was validated, so the concurrent store of a newer 200 is never
// overwritten by the freshening of an older one (RFC 9111 §4.3.4 updates
// the stored response, not whichever response landed last).
func (c *responseCache) freshen(key, validatedEtag string, entry storedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, ok := c.entries[key]
	if !ok || current.validator.wire != validatedEtag {
		return
	}

	c.entries[key] = entry
}

// invalidate removes a stored entry entirely: a non-error response to an
// unsafe request method invalidates the target URI (RFC 9111 §4.4), so the
// next GET must fetch and store afresh instead of revalidating stale state.
func (c *responseCache) invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)

	for i, existing := range c.order {
		if existing == key {
			c.order = append(c.order[:i], c.order[i+1:]...)

			break
		}
	}
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

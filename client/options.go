package etagclient

import "net/http"

// Options configures the conditional GET cache of a [Transport]. The zero
// value is safe: every field defaults to a sensible value.
type Options struct {
	// KeyFunc derives the cache key from a request. When nil, the request URL
	// string is used.
	//
	// WARNING: the default key ignores credentials. When the same URL can
	// return different responses for different callers (Authorization header,
	// cookies), supply a KeyFunc that includes a credential fingerprint so a
	// cached response is never replayed across principals.
	KeyFunc func(*http.Request) string

	// MaxEntries bounds the cache. Zero or negative is clamped to 256. When
	// full, the oldest entry is evicted (FIFO).
	MaxEntries int

	// MaxBodyBytes is the largest response body that may be cached. Zero or
	// negative is clamped to 1 MiB. Larger responses pass through uncached
	// with their bodies intact, so a huge payload cannot balloon memory.
	MaxBodyBytes int

	// PreserveOn304 governs which header fields the 304 contributes to the
	// rebuilt 200 and to the stored entry, as RFC 9111 §4.3.4 freshening
	// (via the §3.2 update rules) prescribes.
	//
	// nil (the default) is the RFC behavior: every field provided in the 304
	// replaces the stored value, except fields excepted from storage
	// (Connection and friends), Content-Length, Content-Range, and the
	// Content-Encoding of a transparently decoded body. That includes Age:
	// when a revalidation 304 reports a grown Age, the rebuilt response wears
	// it, so a stale edge cache cannot hide behind the Age the first 200
	// carried.
	//
	// A non-empty list restricts freshening to the named fields (use it for
	// headers whose fresh value matters, like rate limits). An empty non-nil
	// slice disables freshening entirely: the stored values survive every
	// rebuild.
	PreserveOn304 []string

	// FromCacheHeader, when non-empty, is set to "1" on responses rebuilt
	// from cache so tests and diagnostics can distinguish them from network
	// 200s. Empty disables the marker.
	FromCacheHeader string
}

const (
	defaultMaxEntries   = 256
	defaultMaxBodyBytes = 1024 * 1024
)

// normalize applies defaults to a zero-valued or partially filled Options.
func (o Options) normalize() Options {
	if o.MaxEntries <= 0 {
		o.MaxEntries = defaultMaxEntries
	}

	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = defaultMaxBodyBytes
	}

	// PreserveOn304 is deliberately not defaulted: nil selects the RFC
	// 9111 §4.3.4 freshening behavior, while an empty non-nil slice (also
	// valid) disables it.

	if o.KeyFunc == nil {
		o.KeyFunc = defaultKeyFunc
	}

	return o
}

// defaultKeyFunc keys a cache entry by request URL alone.
func defaultKeyFunc(req *http.Request) string {
	return req.URL.String()
}

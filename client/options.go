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

	// PreserveOn304 lists response headers copied from the 304 onto the
	// rebuilt 200, replacing the cached values. It defaults to Date: per RFC
	// 7232 §4.1 a 304 carries a fresh Date that the rebuilt response must
	// wear instead of the stale cached one. Use it for headers whose fresh
	// value matters (rate limits, retry hints). An empty non-nil slice
	// disables merging entirely.
	PreserveOn304 []string

	// FromCacheHeader, when non-empty, is set to "1" on responses rebuilt
	// from cache so tests and diagnostics can distinguish them from network
	// 200s. Empty disables the marker.
	FromCacheHeader string
}

const (
	defaultMaxEntries   = 256
	defaultMaxBodyBytes = 1024 * 1024
	headerDate          = "Date"
)

// normalize applies defaults to a zero-valued or partially filled Options.
func (o Options) normalize() Options {
	if o.MaxEntries <= 0 {
		o.MaxEntries = defaultMaxEntries
	}

	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = defaultMaxBodyBytes
	}

	if o.PreserveOn304 == nil {
		o.PreserveOn304 = []string{headerDate}
	}

	if o.KeyFunc == nil {
		o.KeyFunc = defaultKeyFunc
	}

	return o
}

// defaultKeyFunc keys a cache entry by request URL alone.
func defaultKeyFunc(req *http.Request) string {
	return req.URL.String()
}

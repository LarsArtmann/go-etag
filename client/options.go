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
	//
	// The limit counts the bytes as the transport would store them: when
	// net/http transparently decompresses the body (gzip), it counts the
	// decoded bytes, not the encoded wire bytes, because the decoded form is
	// what a rebuilt response would serve.
	MaxBodyBytes int

	// FreshenOn304 governs which header fields a 304 Not Modified response
	// contributes to the rebuilt 200 and to the stored entry (RFC 9111 §4.3.4
	// freshening, via the §3.2 update rules). The zero value is the RFC
	// behavior; the constructors [FreshenPerRFC], [FreshenFields], and
	// [FreshenNone] select it or restrict it explicitly.
	//
	// Under the RFC behavior every field provided in the 304 replaces the
	// stored value, except fields excepted from storage (Connection and
	// friends), Content-Length, Content-Range, and the Content-Encoding of a
	// transparently decoded body. That includes Age: when a revalidation 304
	// reports a grown Age, the rebuilt response wears it, so a stale edge
	// cache cannot hide behind the Age the first 200 carried.
	FreshenOn304 FreshenPolicy

	// FromCacheHeader, when non-empty, is set to "1" on responses rebuilt
	// from cache so tests and diagnostics can distinguish them from network
	// 200s. Empty disables the marker.
	FromCacheHeader string
}

// FreshenPolicy selects how a 304 Not Modified response updates the rebuilt
// response and the stored entry (RFC 9111 §4.3.4 via the §3.2 update
// rules). The zero value is the RFC behavior, so an untouched [Options]
// field freshens per spec; the constructors make a chosen policy explicit.
type FreshenPolicy struct {
	kind   freshenKind
	fields []string
}

type freshenKind int

const (
	freshenPerRFC freshenKind = iota
	freshenNamedFields
	freshenNothing
)

// FreshenPerRFC returns the default policy: every header field the 304
// provides replaces the stored value, except fields excepted from storage
// (RFC 9111 §3.1), Content-Length and Content-Range (§3.2), and the
// Content-Encoding of a transparently decoded body (§3.2).
func FreshenPerRFC() FreshenPolicy {
	return FreshenPolicy{kind: freshenPerRFC, fields: nil}
}

// FreshenFields restricts freshening to the named header fields; every
// other stored field survives each rebuild untouched. Use it for fields
// whose fresh value matters, like rate limits or Retry-After.
func FreshenFields(fields ...string) FreshenPolicy {
	return FreshenPolicy{kind: freshenNamedFields, fields: fields}
}

// FreshenNone disables freshening entirely: the stored values survive every
// rebuild, and only the validator flows through.
func FreshenNone() FreshenPolicy {
	return FreshenPolicy{kind: freshenNothing, fields: nil}
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

	// FreshenOn304 is deliberately not defaulted: its zero value already
	// selects the RFC 9111 §4.3.4 freshening behavior, so normalize keeps it
	// exactly as provided.

	if o.KeyFunc == nil {
		o.KeyFunc = defaultKeyFunc
	}

	return o
}

// defaultKeyFunc keys a cache entry by request URL alone.
func defaultKeyFunc(req *http.Request) string {
	return req.URL.String()
}

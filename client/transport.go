package etagclient

import (
	"net/http"
	"strconv"
)

const (
	headerETag         = "ETag"
	headerIfNoneMatch  = "If-None-Match"
	headerCacheControl = "Cache-Control"
	headerConnection   = "Connection"

	headerLastModified  = "Last-Modified"
	headerContentLength = "Content-Length"

	statusTextOK = "200 OK"
	markerValue  = "1"

	// oversizedProbeBytes extends the read limit past MaxBodyBytes so a body
	// of exactly MaxBodyBytes is distinguishable from a larger one.
	oversizedProbeBytes = 1
)

// Transport is an http.RoundTripper implementing client-side conditional GET
// caching. GET requests carrying a stored validator are sent with
// If-None-Match; the resulting 304 is rebuilt into the cached 200 the caller
// expects; fresh 200s are stored for the next round trip. Non-GET methods and
// non-200 responses pass through untouched; responses carrying
// Cache-Control: no-store are never stored (RFC 9111 §3); and a non-error
// response to an unsafe method invalidates the stored entry (RFC 9111 §4.4).
type Transport struct {
	next  http.RoundTripper
	cache *responseCache
	opts  Options
}

// NewTransport returns a Transport wrapping next. A nil next falls back to
// http.DefaultTransport. Zero-valued Options fields are defaulted.
func NewTransport(next http.RoundTripper, opts Options) *Transport {
	if next == nil {
		next = http.DefaultTransport
	}

	normalized := opts.normalize()

	return &Transport{
		next:  next,
		cache: newResponseCache(normalized.MaxEntries),
		opts:  normalized,
	}
}

// Stats returns current cache counters.
func (t *Transport) Stats() Stats {
	return t.cache.stats()
}

// RoundTrip implements http.RoundTripper. The cache key is derived only when
// a method can use it — GET for lookups, an invalidating non-GET after its
// response arrives — so passthrough round trips never pay for key derivation.
// A request carrying Cache-Control: no-store bypasses the cache entirely
// (RFC 9111 §5.2.2.5 request directive): nothing about the exchange is
// stored, while an unsafe method still invalidates (RFC 9111 §4.4).
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodHead {
		if hasNoStoreDirective(req.Header) {
			return t.passThrough(req)
		}

		return t.roundTripHead(req)
	}

	if req.Method != http.MethodGet {
		return t.roundTripUnsafe(req)
	}

	if hasNoStoreDirective(req.Header) {
		return t.passThrough(req)
	}

	return t.roundTripGet(req)
}

// roundTripGet serves a GET through the conditional cache: a stored validator
// rides on a cloned request as If-None-Match, a 304 it produces is rebuilt
// from the stored entry, and a fresh 200 with a validator is stored for the
// next round trip.
func (t *Transport) roundTripGet(req *http.Request) (*http.Response, error) {
	key := t.opts.KeyFunc(req)

	entry, cached := t.cache.get(key)

	// The RoundTripper contract forbids mutating the caller's request, so the
	// validator rides on a clone. A caller-supplied If-None-Match is their own
	// conditional (RFC 9110 §13.1.2) and is never clobbered: a 304 answering
	// it belongs to the caller, so it passes through instead of being rebuilt.
	injected := false

	if cached && req.Header.Get(headerIfNoneMatch) == "" {
		req = req.Clone(req.Context())
		req.Header.Set(headerIfNoneMatch, entry.validator.wire)

		injected = true
	}

	resp, err := t.passThrough(req)
	if err != nil || resp == nil {
		return resp, err
	}

	switch {
	case injected && resp.StatusCode == http.StatusNotModified:
		return t.rebuildFromCache(resp, key, entry), nil

	case resp.StatusCode == http.StatusOK &&
		resp.Header.Get(headerETag) != "" &&
		!hasNoStoreDirective(resp.Header):
		t.store(resp, key, resp.Header.Get(headerETag))
	}

	return resp, nil
}

// roundTripUnsafe passes a non-GET request through untouched, invalidating
// the stored entry for its URI when the response is a non-error answer to an
// unsafe method (RFC 9111 §4.4), so a mutation cannot leave a pre-mutation
// body waiting to be rebuilt for the next GET.
func (t *Transport) roundTripUnsafe(req *http.Request) (*http.Response, error) {
	resp, err := t.passThrough(req)
	if err == nil && resp != nil && isUnsafeMethod(req.Method) && isNonErrorStatus(resp.StatusCode) {
		t.cache.invalidate(t.opts.KeyFunc(req))
	}

	return resp, err
}

// passThrough sends the request untouched through the wrapped transport.
func (t *Transport) passThrough(req *http.Request) (*http.Response, error) {
	return t.next.RoundTrip(req) //nolint:wrapcheck // passthrough preserves the underlying error
}

// roundTripHead passes a HEAD request through untouched and uses its 200 to
// freshen or invalidate the stored GET entry for the same key
// (RFC 9111 §4.3.5): a HEAD response whose validators and Content-Length
// match the stored response updates the stored metadata with the fields the
// HEAD provides (§3.2 update rules), while any mismatch — or the absence of
// a comparable validator — marks the stored response stale, so the next GET
// fetches afresh. A no-store response neither updates nor invalidates:
// storing parts of it is forbidden (RFC 9111 §3) and it gives no signal that
// the stored representation changed.
func (t *Transport) roundTripHead(req *http.Request) (*http.Response, error) {
	resp, err := t.passThrough(req)
	if err != nil || resp == nil {
		return resp, err
	}

	if resp.StatusCode != http.StatusOK || hasNoStoreDirective(resp.Header) {
		return resp, nil
	}

	key := t.opts.KeyFunc(req)

	entry, cached := t.cache.get(key)
	if !cached {
		return resp, nil
	}

	if headConfirmsStored(resp.Header, entry) {
		t.freshenFromHead(key, entry, resp.Header)
	} else {
		t.cache.invalidate(key)
	}

	return resp, nil
}

// headConfirmsStored reports whether a HEAD 200 response describes the
// stored response (RFC 9111 §4.3.5): every validator field both responses
// carry must match (ETag weakly, Last-Modified exactly), at least one
// validator must be comparable, and a Content-Length on the HEAD response
// must equal the stored body's length.
func headConfirmsStored(head http.Header, entry storedResponse) bool {
	validatorMatched := false

	if headEtag := head.Get(headerETag); entry.validator.wire != "" && headEtag != "" {
		if !entry.validator.weaklyMatches(headEtag) {
			return false
		}

		validatorMatched = true
	}

	storedLastModified := entry.header.Get(headerLastModified)

	if headLastModified := head.Get(headerLastModified); storedLastModified != "" && headLastModified != "" {
		if headLastModified != storedLastModified {
			return false
		}

		validatorMatched = true
	}

	if !validatorMatched {
		return false
	}

	if headContentLength := head.Get(headerContentLength); headContentLength != "" {
		return headContentLength == strconv.Itoa(len(entry.body))
	}

	return true
}

// freshenFromHead applies a confirming HEAD 200's metadata to the stored
// entry (RFC 9111 §4.3.5 mandates the §3.2 update rules for the fields the
// HEAD provides). The stored body, status, and decompression record are the
// validated ones and never change.
func (t *Transport) freshenFromHead(key string, entry storedResponse, head http.Header) {
	t.persistFreshened(key, entry, freshenedHeader(entry.header, head, entry.uncompressed))
}

func (c *chainedBody) Read(p []byte) (int, error) {
	return c.reader.Read(p) //nolint:wrapcheck // passthrough preserves the underlying error
}

func (c *chainedBody) Close() error {
	return c.body.Close() //nolint:wrapcheck // passthrough preserves the underlying error
}

// isUnsafeMethod reports whether method can change origin state: GET, HEAD,
// OPTIONS, and TRACE are safe per RFC 9110 §9.2.1, and anything else,
// including unrecognized methods, is treated as unsafe.
func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

// isNonErrorStatus reports whether status is in the 2xx or 3xx range whose
// response to an unsafe request method triggers invalidation (RFC 9111 §4.4
// defines a non-error response as 2xx or 3xx).
func isNonErrorStatus(status int) bool {
	return status >= http.StatusOK && status < http.StatusBadRequest
}

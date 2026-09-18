package etagclient

import (
	"bytes"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/larsartmann/go-etag/entitytag"
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
		req.Header.Set(headerIfNoneMatch, entry.etag)

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
func headConfirmsStored(head http.Header, entry cacheEntry) bool {
	validatorMatched := false

	if headEtag := head.Get(headerETag); entry.etag != "" && headEtag != "" {
		if !weaklyMatchesValidator(headEtag, entry.etag) {
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
func (t *Transport) freshenFromHead(key string, entry cacheEntry, head http.Header) {
	t.persistFreshened(key, entry, freshenedHeader(entry.header, head, entry.uncompressed))
}

// rebuildFromCache synthesizes the 200 the caller expects: cached body and
// headers, freshened with the fields the 304 provides (RFC 9111 §4.3.4 via
// the §3.2 update rules, unless FreshenOn304 restricts it), and persists
// the freshened metadata back onto the stored entry.
func (t *Transport) rebuildFromCache(notModified *http.Response, key string, entry cacheEntry) *http.Response {
	t.cache.countHit()

	drainAndClose(notModified)

	header := t.rebuiltHeader(entry, notModified)
	restoreMismatchedValidator(header, notModified, entry)

	if t.opts.FromCacheHeader != "" {
		header.Set(t.opts.FromCacheHeader, markerValue)
	}

	t.persistFreshened(key, entry, header)

	return &http.Response{
		Status:        statusTextOK,
		StatusCode:    http.StatusOK,
		Proto:         notModified.Proto,
		ProtoMajor:    notModified.ProtoMajor,
		ProtoMinor:    notModified.ProtoMinor,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(entry.body)),
		ContentLength: int64(len(entry.body)),
		Request:       notModified.Request,
		Uncompressed:  entry.uncompressed,
	}
}

// rebuiltHeader builds the header set of the synthesized 200 from the stored
// entry and the 304: RFC 9111 §4.3.4 freshening by default, restricted to
// the named fields by FreshenFields, and untouched by FreshenNone.
func (t *Transport) rebuiltHeader(entry cacheEntry, notModified *http.Response) http.Header {
	if t.opts.FreshenOn304.kind == freshenPerRFC {
		return freshenedHeader(entry.header, notModified.Header, entry.uncompressed)
	}

	header := cloneHeader(entry.header)

	if t.opts.FreshenOn304.kind == freshenNamedFields {
		for _, name := range t.opts.FreshenOn304.fields {
			mergeHeader(header, notModified.Header, name)
		}
	}

	return header
}

// restoreMismatchedValidator guards RFC 9111 §4.3.4's validator filtering: a
// 304 declaring a validator that does not weak-match the one it just
// validated names a different representation, so neither the rebuilt
// response nor the store adopts the claim — the stored validator stays and
// the next revalidation presents the validator the server actually answered.
func restoreMismatchedValidator(header http.Header, notModified *http.Response, entry cacheEntry) {
	candidate := notModified.Header.Get(headerETag)
	if candidate != "" && !weaklyMatchesValidator(candidate, entry.etag) {
		header.Set(headerETag, entry.etag)
	}
}

// persistFreshened writes a freshened header back onto the stored entry
// (RFC 9111 §4.3.4 requires updating the stored response, not only the
// synthesized one; §4.3.5 HEAD freshening commits through the same rules),
// so later revalidations present the validator the server returned and
// later rebuilds wear the freshened values. The from-cache marker is
// diagnostic output, never cache state, so it is stripped first.
func (t *Transport) persistFreshened(key string, entry cacheEntry, header http.Header) {
	persisted := header.Clone()

	if t.opts.FromCacheHeader != "" {
		persisted.Del(t.opts.FromCacheHeader)
	}

	// Whatever did not flow from the freshening response keeps the stored
	// validator; a mismatched 304's claim never reaches here (restored before
	// persisting).
	etag := persisted.Get(headerETag)
	if etag == "" {
		etag = entry.etag
	}

	t.cache.freshen(key, entry.etag, cacheEntry{
		etag:         etag,
		status:       entry.status,
		header:       persisted,
		body:         entry.body,
		uncompressed: entry.uncompressed,
	})
}

// store reads the response body and caches it when a validator is present and
// the body fits MaxBodyBytes. Either way the caller receives a re-readable
// body with every original byte; oversized bodies keep streaming their
// remainder, and read failures replay through the restored body instead of
// breaking the response.
func (t *Transport) store(resp *http.Response, key, etag string) {
	limit := int64(t.opts.MaxBodyBytes) + oversizedProbeBytes

	buffered, readErr := io.ReadAll(io.LimitReader(resp.Body, limit))

	switch {
	case readErr != nil:
		resp.Body = newChainedBody(buffered, resp.Body)

		return

	case len(buffered) > t.opts.MaxBodyBytes:
		resp.Body = newChainedBody(buffered, resp.Body)

		return
	}

	closeErr := resp.Body.Close()
	if closeErr != nil {
		resp.Body = io.NopCloser(bytes.NewReader(buffered))
		resp.ContentLength = int64(len(buffered))

		return
	}

	header := resp.Header.Clone()
	if header == nil {
		header = make(http.Header)
	}

	stripHopByHop(header)

	if t.opts.FromCacheHeader != "" {
		header.Del(t.opts.FromCacheHeader)
	}

	t.cache.set(key, cacheEntry{
		etag:         etag,
		status:       resp.StatusCode,
		header:       header,
		body:         buffered,
		uncompressed: resp.Uncompressed,
	})

	resp.Body = io.NopCloser(bytes.NewReader(buffered))
	resp.ContentLength = int64(len(buffered))
}

// chainedBody chains an already-buffered prefix with the unread remainder of
// the original body, so a passthrough response still streams every byte and
// Close still reaches the original body.
type chainedBody struct {
	reader io.Reader
	body   io.ReadCloser
}

func newChainedBody(buffered []byte, body io.ReadCloser) *chainedBody {
	return &chainedBody{
		reader: io.MultiReader(bytes.NewReader(buffered), body),
		body:   body,
	}
}

func (c *chainedBody) Read(p []byte) (int, error) {
	return c.reader.Read(p) //nolint:wrapcheck // passthrough preserves the underlying error
}

func (c *chainedBody) Close() error {
	return c.body.Close() //nolint:wrapcheck // passthrough preserves the underlying error
}

// cloneHeader returns a mutable copy of stored, substituting an empty
// header when stored is nil: http.Header.Clone returns nil for a nil map,
// which would panic on the first freshening write.
func cloneHeader(stored http.Header) http.Header {
	header := stored.Clone()
	if header == nil {
		header = make(http.Header)
	}

	return header
}

// freshenedHeader applies RFC 9111 §4.3.4 (freshening stored responses upon
// validation) using the §3.2 update rules: every field the 304 provides is
// added to the stored header set, replacing any stored value, except fields
// excepted from storage (§3.1), Content-Length and Content-Range (§3.2), and
// Content-Encoding when net/http transparently decoded the stored body (the
// §3.2 integrity allowance for caches storing processed representations:
// the stored bytes are the decoded ones, so the 304's encoding claim would
// describe bytes the rebuilt response does not carry).
func freshenedHeader(stored, notModified http.Header, uncompressed bool) http.Header {
	header := cloneHeader(stored)

	for name, values := range notModified {
		if skippedByFreshening(name, uncompressed) {
			continue
		}

		header[name] = values
	}

	return header
}

// skippedByFreshening reports whether a 304-provided field must not replace
// the stored value: connection-scope fields are excepted from storage
// (RFC 9111 §3.1), Content-Length and Content-Range from updates (§3.2), and
// Content-Encoding when the stored body is the transparently decoded form
// rather than the encoded wire bytes (§3.2).
func skippedByFreshening(name string, uncompressed bool) bool {
	if isHopByHop(name) {
		return true
	}

	switch textproto.CanonicalMIMEHeaderKey(name) {
	case "Content-Length", "Content-Range":
		return true
	case "Content-Encoding":
		return uncompressed
	default:
		return false
	}
}

// isHopByHop reports whether name is a connection-scope field, which a cache
// neither stores nor replays (RFC 9111 §3.1).
func isHopByHop(name string) bool {
	switch textproto.CanonicalMIMEHeaderKey(name) {
	case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authentication-Info",
		"Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade":
		return true
	default:
		return false
	}
}

// stripHopByHop removes fields a stored response must not carry
// (RFC 9111 §3.1): the connection-scope set, plus every field the Connection
// header lists as hop-by-hop for this connection.
func stripHopByHop(header http.Header) {
	for _, value := range header.Values(headerConnection) {
		for name := range strings.SplitSeq(value, ",") {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				header.Del(textproto.CanonicalMIMEHeaderKey(trimmed))
			}
		}
	}

	for name := range header {
		if isHopByHop(name) {
			header.Del(name)
		}
	}
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

// weaklyMatchesValidator reports whether two entity-tag field values carry
// the same opaque tag, ignoring strength (the RFC 9110 §8.8.3.2 weak
// comparison If-None-Match uses). Comparison is delegated to the shared
// entity-tag domain type: a field value that does not parse as an RFC 7232
// §2.3 entity-tag cannot be compared and never matches.
func weaklyMatchesValidator(a, b string) bool {
	parsedA, okA := entitytag.ParseETag(a)
	parsedB, okB := entitytag.ParseETag(b)

	return okA && okB && parsedA.WeakEqual(parsedB)
}

// hasNoStoreDirective reports whether a response carries the no-store cache
// directive, which forbids storing any part of it (RFC 9111 §3, §5.2.2.5).
func hasNoStoreDirective(header http.Header) bool {
	for _, value := range header.Values(headerCacheControl) {
		for _, directive := range cacheControlDirectives(value) {
			name, _, _ := strings.Cut(directive, "=")
			if strings.EqualFold(strings.TrimSpace(name), "no-store") {
				return true
			}
		}
	}

	return false
}

// cacheControlDirectives splits a Cache-Control field value into its
// directives, treating commas inside quoted arguments as part of the
// argument so a quoted "no-store" never matches as a directive name.
func cacheControlDirectives(value string) []string {
	var (
		directives []string
		current    strings.Builder
		quoted     bool
		escaped    bool
	)

	for _, char := range value {
		switch {
		case escaped:
			escaped = false

			current.WriteRune(char)
		case quoted && char == '\\':
			escaped = true

			current.WriteRune(char)
		case char == '"':
			quoted = !quoted

			current.WriteRune(char)
		case char == ',' && !quoted:
			directives = append(directives, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}

	return append(directives, current.String())
}

// mergeHeader copies the header name from src to dst, preferring the exact
// map key and falling back to its canonical form.
func mergeHeader(dst, src http.Header, name string) {
	if values, ok := src[name]; ok {
		dst[name] = values

		return
	}

	canonical := textproto.CanonicalMIMEHeaderKey(name)
	if values, ok := src[canonical]; ok {
		dst[canonical] = values
	}
}

// drainAndClose discards any residual 304 body and releases the connection.
// Both failures are deliberately ignored: the 304 response is discarded
// either way, and net/http closes a connection whose body was not fully
// drained, so the worst case is losing connection reuse.
func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body) //nolint:erraudit // best-effort drain of a body we are discarding
	_ = resp.Body.Close()                 //nolint:erraudit // a failed teardown of a discarded response needs no handling
}

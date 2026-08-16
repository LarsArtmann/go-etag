package etagclient

import (
	"bytes"
	"io"
	"net/http"
	"net/textproto"
)

const (
	headerETag        = "ETag"
	headerIfNoneMatch = "If-None-Match"

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
// non-200 responses pass through untouched.
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

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return t.next.RoundTrip(req) //nolint:wrapcheck // passthrough preserves the underlying error
	}

	key := t.opts.KeyFunc(req)

	entry, cached := t.cache.get(key)
	if cached {
		// The RoundTripper contract forbids mutating the caller's request,
		// so the validator rides on a clone.
		req = req.Clone(req.Context())
		req.Header.Set(headerIfNoneMatch, entry.etag)
	}

	resp, err := t.next.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err //nolint:wrapcheck // passthrough preserves the underlying error
	}

	switch {
	case resp.StatusCode == http.StatusNotModified && cached:
		return t.rebuildFromCache(resp, entry), nil

	case resp.StatusCode == http.StatusOK && resp.Header.Get(headerETag) != "":
		t.store(resp, key, resp.Header.Get(headerETag))
	}

	return resp, nil
}

// rebuildFromCache synthesizes the 200 the caller expects: cached body and
// headers, with the 304's fresh values merged for the PreserveOn304 headers
// (Date by default, per RFC 7232 §4.1).
func (t *Transport) rebuildFromCache(notModified *http.Response, entry cacheEntry) *http.Response {
	t.cache.countHit()

	drainAndClose(notModified)

	header := entry.header.Clone()
	if header == nil {
		header = make(http.Header)
	}

	for _, name := range t.opts.PreserveOn304 {
		mergeHeader(header, notModified.Header, name)
	}

	if t.opts.FromCacheHeader != "" {
		header.Set(t.opts.FromCacheHeader, markerValue)
	}

	return &http.Response{ //nolint:exhaustruct // transport metadata has no meaning on a rebuilt response
		Status:        statusTextOK,
		StatusCode:    http.StatusOK,
		Proto:         notModified.Proto,
		ProtoMajor:    notModified.ProtoMajor,
		ProtoMinor:    notModified.ProtoMinor,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(entry.body)),
		ContentLength: int64(len(entry.body)),
		Request:       notModified.Request,
	}
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

	if t.opts.FromCacheHeader != "" {
		header.Del(t.opts.FromCacheHeader)
	}

	t.cache.set(key, cacheEntry{
		etag:   etag,
		status: resp.StatusCode,
		header: header,
		body:   buffered,
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
func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

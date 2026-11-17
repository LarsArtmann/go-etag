package etagclient

import (
	"bytes"
	"io"
	"net/http"
)

// rebuildFromCache synthesizes the 200 the caller expects: cached body and
// headers, freshened with the fields the 304 provides (RFC 9111 §4.3.4 via
// the §3.2 update rules, unless FreshenOn304 restricts it), and persists
// the freshened metadata back onto the stored entry.
func (t *Transport) rebuildFromCache(notModified *http.Response, key string, entry storedResponse) *http.Response {
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
func (t *Transport) rebuiltHeader(entry storedResponse, notModified *http.Response) http.Header {
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
func restoreMismatchedValidator(header http.Header, notModified *http.Response, entry storedResponse) {
	candidate := notModified.Header.Get(headerETag)
	if candidate != "" && !entry.validator.weaklyMatches(candidate) {
		header.Set(headerETag, entry.validator.wire)
	}
}

// persistFreshened writes a freshened header back onto the stored entry
// (RFC 9111 §4.3.4 requires updating the stored response, not only the
// synthesized one; §4.3.5 HEAD freshening commits through the same rules),
// so later revalidations present the validator the server returned and
// later rebuilds wear the freshened values. The from-cache marker is
// diagnostic output, never cache state, so it is stripped first.
func (t *Transport) persistFreshened(key string, entry storedResponse, header http.Header) {
	persisted := header.Clone()

	if t.opts.FromCacheHeader != "" {
		persisted.Del(t.opts.FromCacheHeader)
	}

	// Whatever did not flow from the freshening response keeps the stored
	// validator; a mismatched 304's claim never reaches here (restored before
	// persisting).
	validatorWire := persisted.Get(headerETag)
	if validatorWire == "" {
		validatorWire = entry.validator.wire
	}

	t.cache.freshen(key, entry.validator.wire, storedResponse{
		validator:    newStoredValidator(validatorWire),
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
func (t *Transport) store(resp *http.Response, key, validatorWire string) {
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

	t.cache.set(key, storedResponse{
		validator:    newStoredValidator(validatorWire),
		status:       resp.StatusCode,
		header:       header,
		body:         buffered,
		uncompressed: resp.Uncompressed,
	})

	resp.Body = io.NopCloser(bytes.NewReader(buffered))
	resp.ContentLength = int64(len(buffered))
}

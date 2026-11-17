package etagclient

import (
	"net/http"
	"net/textproto"
	"strings"
)

// cloneHeader returns a mutable copy of stored, substituting an empty
// header when stored is nil: http.Header.Clone returns nil for a nil map,
// which would panic on the first freshening write. The nil branch is
// unreachable through current production paths — every storedResponse.header
// is non-nil by construction (store nil-guards its Clone explicitly;
// persistFreshened clones a header that already passed through cloneHeader
// or freshenedHeader) — so the guard is defense-in-depth against a future
// store variant, pinned by TestCloneHeaderNilReturnsEmpty.
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

package etag

import "github.com/larsartmann/go-etag/entitytag"

// The entity-tag domain type lives in the shared entitytag package so the
// client cache can use the same typed validator without importing this
// middleware package. These aliases keep the historical server surface
// (`etag.ETag`, `etag.ParseETag`, ...) compiling unchanged.

type (
	// ETag is an RFC 7232 §2.3 entity-tag: an opaque validator with a
	// strength. See [entitytag.ETag].
	ETag = entitytag.ETag

	// Strength classifies entity-tag comparison semantics per RFC 7232 §2.1.
	// See [entitytag.Strength].
	Strength = entitytag.Strength
)

const (
	// Strong means the entity-tag is a strong validator per RFC 7232 §2.1.
	Strong = entitytag.Strong

	// Weak means the entity-tag is a weak validator per RFC 7232 §2.1.
	Weak = entitytag.Weak
)

// NewETag constructs an ETag from its opaque value and strength.
// The opaque value is the unquoted content between the double quotes in the
// wire format. It must not contain double-quote characters.
func NewETag(opaque string, strength Strength) ETag {
	return entitytag.NewETag(opaque, strength)
}

// ParseETag parses a single entity-tag from its wire-format representation.
// Returns the ETag and true on success, or ETag{} and false if the input does
// not match the RFC 7232 §2.3 entity-tag ABNF.
func ParseETag(s string) (ETag, bool) {
	return entitytag.ParseETag(s)
}

// ParseETagList parses a comma-separated list of entity-tags from an HTTP
// conditional request header value (If-Match or If-None-Match).
func ParseETagList(header string) []ETag {
	return entitytag.ParseETagList(header)
}

// MatchesIfNoneMatch reports whether tag matches the If-None-Match header
// value using the RFC 7232 §2.3.2 weak comparison function. The header value
// may be "*" (wildcard, matches any existing representation) or a
// comma-separated list of entity-tags.
func MatchesIfNoneMatch(tag ETag, headerValue string) bool {
	return entitytag.MatchesIfNoneMatch(tag, headerValue)
}

// MatchesIfMatch reports whether tag matches the If-Match header value using
// the RFC 7232 §2.3.2 strong comparison function. Applications should call
// this in their handlers to evaluate If-Match preconditions for unsafe
// methods (PUT, POST, DELETE) to prevent lost updates.
func MatchesIfMatch(tag ETag, headerValue string) bool {
	return entitytag.MatchesIfMatch(tag, headerValue)
}

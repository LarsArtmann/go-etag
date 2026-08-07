package etag

import (
	"slices"
	"strings"
)

// Strength classifies entity-tag comparison semantics per RFC 7232 §2.1.
//
// A Strong validator is guaranteed to change whenever the representation data
// changes. A Weak validator may not change for every representation change,
// making it suitable for cache validation but not for lost-update prevention.
//
// The default FNV-64a hash (see [DefaultETagConfig]) produces Strong entity-tags
// with a practical collision resistance of ~4.3 billion distinct bodies
// (birthday bound). For applications requiring higher guarantees, provide a
// collision-resistant hash function via [ETagConfig.HashFunc].
type Strength int

const (
	// Strong means the entity-tag is a strong validator per RFC 7232 §2.1.
	// Strong validators are usable for all conditional requests: cache
	// validation, range requests, and lost-update prevention.
	Strong Strength = iota

	// Weak means the entity-tag is a weak validator per RFC 7232 §2.1.
	// Weak validators are suitable for cache validation but cannot be used
	// with If-Match for lost-update prevention.
	Weak
)

func (s Strength) valid() bool {
	return s == Strong || s == Weak
}

// ETag is an RFC 7232 §2.3 entity-tag: an opaque validator with a strength.
//
// The zero value ETag{} represents an absent or invalid tag. Use [IsValid] to
// check whether a tag carries a meaningful opaque value.
//
// ETag implements [fmt.Stringer]; its String method returns the wire-format
// representation suitable for the ETag response header (e.g. W/"abc" or "abc").
type ETag struct {
	opaque   string
	strength Strength
}

// NewETag constructs an ETag from its opaque value and strength.
// The opaque value is the unquoted content between the double quotes in the
// wire format. It must not contain double-quote characters.
func NewETag(opaque string, strength Strength) ETag {
	return ETag{
		opaque:   opaque,
		strength: strength,
	}
}

// String returns the wire-format representation of the entity-tag for use in
// HTTP headers: W/"opaque" for weak tags, "opaque" for strong tags.
func (e ETag) String() string {
	if e.strength == Weak {
		return `W/"` + e.opaque + `"`
	}

	return `"` + e.opaque + `"`
}

// OpaqueTag returns the unquoted opaque value of the entity-tag, excluding
// the surrounding double quotes and weakness indicator.
func (e ETag) OpaqueTag() string {
	return e.opaque
}

// IsWeak reports whether the entity-tag is a weak validator (RFC 7232 §2.1).
func (e ETag) IsWeak() bool {
	return e.strength == Weak
}

// IsValid reports whether the entity-tag has a non-empty opaque value.
// The zero value ETag{} is not valid.
func (e ETag) IsValid() bool {
	return e.opaque != ""
}

// StrongEqual reports whether two entity-tags are equivalent using the RFC
// 7232 §2.3.2 strong comparison function: both must be strong validators and
// their opaque-tags must match character-by-character.
//
// Strong comparison is required for If-Match (RFC 7232 §3.1).
func (e ETag) StrongEqual(o ETag) bool {
	return e.strength == Strong && o.strength == Strong && e.opaque == o.opaque
}

// WeakEqual reports whether two entity-tags are equivalent using the RFC 7232
// §2.3.2 weak comparison function: their opaque-tags must match
// character-by-character, regardless of either being tagged as weak.
//
// Weak comparison is used for If-None-Match (RFC 7232 §3.2).
func (e ETag) WeakEqual(o ETag) bool {
	return e.opaque == o.opaque
}

// ParseETag parses a single entity-tag from its wire-format representation.
// Returns the ETag and true on success, or ETag{} and false if the input does
// not match the RFC 7232 §2.3 entity-tag ABNF:
//
//	entity-tag = [ weak ] opaque-tag
//	weak       = %x57.2F ; "W/"
//	opaque-tag = DQUOTE *etagc DQUOTE
func ParseETag(s string) (ETag, bool) {
	s = strings.TrimSpace(s)

	strength := Strong

	if strings.HasPrefix(s, "W/") {
		strength = Weak
		s = s[weakPrefixLen:]
	}

	if len(s) < minETagWireLen || s[0] != '"' || s[len(s)-1] != '"' {
		return ETag{}, false //nolint:exhaustruct // zero value signals parse failure
	}

	opaque := s[1 : len(s)-1]

	if opaque == "" {
		return ETag{}, false //nolint:exhaustruct // zero value signals parse failure
	}

	return ETag{
		opaque:   opaque,
		strength: strength,
	}, true
}

// ParseETagList parses a comma-separated list of entity-tags from an HTTP
// conditional request header value (If-Match or If-None-Match).
//
// Commas inside quoted opaque-tags are respected. Malformed tags are silently
// skipped, matching the lenient parsing posture recommended by RFC 7232.
// Returns an empty slice (not nil) if the header contains no valid tags.
func ParseETagList(header string) []ETag {
	tags := make([]ETag, 0)

	for _, raw := range splitRawETags(header) {
		tag, ok := ParseETag(raw)
		if !ok {
			continue
		}

		tags = append(tags, tag)
	}

	return tags
}

// MatchesIfNoneMatch reports whether tag matches the If-None-Match header
// value using the RFC 7232 §2.3.2 weak comparison function. The header value
// may be "*" (wildcard, matches any existing representation) or a
// comma-separated list of entity-tags.
//
// This is the comparison function used by the middleware for cache validation.
func MatchesIfNoneMatch(tag ETag, headerValue string) bool {
	if strings.TrimSpace(headerValue) == wildcard {
		return true
	}

	return slices.ContainsFunc(ParseETagList(headerValue), tag.WeakEqual)
}

// MatchesIfMatch reports whether tag matches the If-Match header value using
// the RFC 7232 §2.3.2 strong comparison function. The header value may be "*"
// (wildcard, matches any existing representation) or a comma-separated list of
// entity-tags.
//
// Applications should call this in their handlers to evaluate If-Match
// preconditions for unsafe methods (PUT, POST, DELETE) to prevent lost updates.
func MatchesIfMatch(tag ETag, headerValue string) bool {
	if strings.TrimSpace(headerValue) == wildcard {
		return true
	}

	return slices.ContainsFunc(ParseETagList(headerValue), tag.StrongEqual)
}

// splitRawETags splits a comma-separated list of entity-tags, respecting
// commas inside quoted opaque-tags per RFC 7232 §2.3 (etagc permits any VCHAR
// except DQUOTE, which includes comma). Backslash escapes inside quoted
// strings are honored so that an escaped DQUOTE does not toggle the quote
// state.
func splitRawETags(list string) []string {
	var tags []string

	start := 0

	inQuotes := false

	escaped := false

	for i := range list {
		if escaped {
			escaped = false

			continue
		}

		if list[i] == '\\' && inQuotes {
			escaped = true

			continue
		}

		if list[i] == '"' {
			inQuotes = !inQuotes
		}

		if list[i] == ',' && !inQuotes {
			tag := strings.TrimSpace(list[start:i])
			if tag != "" {
				tags = append(tags, tag)
			}

			start = i + 1
		}
	}

	tag := strings.TrimSpace(list[start:])

	if tag != "" {
		tags = append(tags, tag)
	}

	return tags
}

const (
	weakPrefixLen  = 2 // len("W/")
	minETagWireLen = 2 // two double quotes: ""
	wildcard       = "*"
)

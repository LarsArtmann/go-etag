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

var (
	// NewETag constructs an ETag from its opaque value and strength.
	NewETag = entitytag.NewETag

	// ParseETag parses a single entity-tag from its wire-format
	// representation.
	ParseETag = entitytag.ParseETag

	// ParseETagList parses a comma-separated list of entity-tags from an
	// HTTP conditional request header value.
	ParseETagList = entitytag.ParseETagList

	// MatchesIfNoneMatch reports whether tag matches the If-None-Match
	// header value using weak comparison.
	MatchesIfNoneMatch = entitytag.MatchesIfNoneMatch

	// MatchesIfMatch reports whether tag matches the If-Match header value
	// using strong comparison.
	MatchesIfMatch = entitytag.MatchesIfMatch
)

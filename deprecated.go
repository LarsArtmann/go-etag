package etag

import etagserver "github.com/larsartmann/go-etag/server"

// ETag is an RFC 7232 §2.3 entity-tag: an opaque validator with a strength.
//
// Deprecated: import github.com/larsartmann/go-etag/server instead; the package
// name stays etag, so only the import path changes. This shim is removed in v1.0.0.
type ETag = etagserver.ETag

// ETagConfig holds configuration for ETag generation and conditional request
// handling.
//
// Deprecated: import github.com/larsartmann/go-etag/server instead; the package
// name stays etag, so only the import path changes. This shim is removed in v1.0.0.
type ETagConfig = etagserver.ETagConfig

// Middleware wraps an http.Handler to intercept or modify request flow.
//
// Deprecated: import github.com/larsartmann/go-etag/server instead; the package
// name stays etag, so only the import path changes. This shim is removed in v1.0.0.
type Middleware = etagserver.Middleware

// Strength classifies entity-tag comparison semantics per RFC 7232 §2.1.
//
// Deprecated: import github.com/larsartmann/go-etag/server instead; the package
// name stays etag, so only the import path changes. This shim is removed in v1.0.0.
type Strength = etagserver.Strength

const (
	// Strong means the entity-tag is a strong validator per RFC 7232 §2.1.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.Strong.
	Strong = etagserver.Strong

	// Weak means the entity-tag is a weak validator per RFC 7232 §2.1.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.Weak.
	Weak = etagserver.Weak
)

const (
	// ErrCodeETagWriteFailed is returned when the ETag writer fails to write
	// buffered or streamed data.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.ErrCodeETagWriteFailed.
	ErrCodeETagWriteFailed = etagserver.ErrCodeETagWriteFailed

	// ErrCodeHijackUnsupported is returned when the underlying ResponseWriter
	// does not implement http.Hijacker.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.ErrCodeHijackUnsupported.
	ErrCodeHijackUnsupported = etagserver.ErrCodeHijackUnsupported

	// ErrCodeHijackFailed is returned when the underlying Hijack call fails.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.ErrCodeHijackFailed.
	ErrCodeHijackFailed = etagserver.ErrCodeHijackFailed

	// ErrCodeInvalidConfig is returned when ETagConfig validation fails.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.ErrCodeInvalidConfig.
	ErrCodeInvalidConfig = etagserver.ErrCodeInvalidConfig

	// ErrCodeHashWriteFailed is returned when the hash function fails to
	// accept data, violating the hash.Hash contract.
	//
	// Deprecated: import github.com/larsartmann/go-etag/server and use etag.ErrCodeHashWriteFailed.
	ErrCodeHashWriteFailed = etagserver.ErrCodeHashWriteFailed
)

// ErrInvalidConfig is the sentinel error returned by ETagConfig.Validate when
// a field value is invalid.
//
// Deprecated: import github.com/larsartmann/go-etag/server and use
// etag.ErrInvalidConfig there. This shim is removed in v1.0.0.
var ErrInvalidConfig = etagserver.ErrInvalidConfig

// New returns middleware that generates ETag headers based on response body
// content and handles If-None-Match conditional requests with 304 Not Modified.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.New there.
func New(cfg ETagConfig) Middleware { return etagserver.New(cfg) }

// NewETag constructs an ETag from its opaque value and strength.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.NewETag there.
func NewETag(opaque string, strength Strength) ETag { return etagserver.NewETag(opaque, strength) }

// DefaultETagConfig returns an ETagConfig with sensible defaults.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.DefaultETagConfig there.
func DefaultETagConfig() ETagConfig { return etagserver.DefaultETagConfig() }

// ParseETag parses a single entity-tag from its wire-format representation.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.ParseETag there.
func ParseETag(s string) (ETag, bool) { return etagserver.ParseETag(s) }

// ParseETagList parses a comma-separated list of entity-tags from an HTTP
// conditional request header value.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.ParseETagList there.
func ParseETagList(header string) []ETag { return etagserver.ParseETagList(header) }

// MatchesIfNoneMatch reports whether tag matches the If-None-Match header
// value using the RFC 7232 §2.3.2 weak comparison function.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.MatchesIfNoneMatch there.
func MatchesIfNoneMatch(tag ETag, headerValue string) bool {
	return etagserver.MatchesIfNoneMatch(tag, headerValue)
}

// MatchesIfMatch reports whether tag matches the If-Match header value using
// the RFC 7232 §2.3.2 strong comparison function.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.MatchesIfMatch there.
func MatchesIfMatch(tag ETag, headerValue string) bool {
	return etagserver.MatchesIfMatch(tag, headerValue)
}

// RegisterErrorClassifications maps stdlib HTTP sentinel errors to their
// behavioral families and registers error message templates for all etag
// error codes.
//
// Deprecated: import github.com/larsartmann/go-etag/server and call etag.RegisterErrorClassifications there.
func RegisterErrorClassifications() { etagserver.RegisterErrorClassifications() }

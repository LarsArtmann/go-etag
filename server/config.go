package etag

import (
	"hash/fnv"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	headerContentLength = "Content-Length"
	headerETag          = "ETag"
	headerIfNoneMatch   = "If-None-Match"
)

const (
	hashUint64Bytes      = 8
	hashUint64HexChars   = hashUint64Bytes * 2 // 16 hex chars for a uint64
	defaultMaxBufferSize = 1024 * 1024         // 1 MB
)

// ETagConfig holds configuration for ETag generation and conditional request
// handling.
type ETagConfig struct {
	// Strength controls whether generated entity-tags are strong or weak
	// validators per RFC 7232 §2.1. The default ([Strong]) is appropriate for
	// the built-in FNV-64a hash. Use [Weak] if your hash function may produce
	// the same value for semantically equivalent but byte-different
	// representations.
	Strength Strength

	// MaxBufferSize is the maximum bytes buffered for ETag computation before
	// abandoning ETag generation and streaming the response without an ETag.
	// Defaults to 1 MB. A non-positive value is clamped to the default.
	MaxBufferSize int

	// HashFunc computes the opaque-tag value from the response body. If nil,
	// FNV-64a is used (fast, 64-bit, collision-resistant for practical body
	// counts). The returned string is the unquoted opaque content; the
	// middleware wraps it with quotes and the optional weakness indicator.
	HashFunc func([]byte) string

	// SkipIfPresent controls whether the middleware respects an ETag header
	// already set by the handler. When true, the handler's ETag is used for
	// If-None-Match comparison and is not overwritten. This is recommended
	// when handlers provide their own strong validators (e.g. database
	// revision numbers). When false (default), the middleware always
	// overwrites the ETag with a body-content hash.
	SkipIfPresent bool

	// Skip, if non-nil, is called before processing each request. If it
	// returns true, the request is passed through without buffering or ETag
	// generation. Use this to exclude endpoints that are unsuitable for ETag
	// buffering (large file downloads, streaming responses, SSE, etc.).
	Skip func(*http.Request) bool

	// OnError is called when a write to the underlying ResponseWriter fails
	// after the response header has been committed and the error cannot be
	// surfaced to the client or returned from Write. If nil, such errors are
	// silently dropped (matching net/http default behavior).
	OnError func(*errorfamily.Error)

	// OnETagGenerated is called each time the middleware computes a new
	// entity-tag from the buffered body and sets it on the ETag header. It
	// does not fire for handler-provided tags adopted via SkipIfPresent, nor
	// for responses that get no ETag at all (empty body, streaming). Pair it
	// with On304 to derive cache hit ratios. If nil, no action is taken.
	OnETagGenerated func(ETag)

	// On304 is called when a conditional request is answered with 304 Not
	// Modified, after the status line is committed. The argument is the
	// entity-tag that matched If-None-Match, regardless of whether it was
	// computed by the middleware or provided by the handler (SkipIfPresent).
	// On304 fires in addition to OnETagGenerated, which fires first during
	// tag resolution; a consumer wanting only one event per request should
	// prefer On304. If nil, no action is taken.
	On304 func(ETag)

	// OnBufferOverflow is called when a response body exceeds MaxBufferSize
	// and the writer switches to streaming without an ETag. The argument is
	// the configured limit that was exceeded. It fires at most once per
	// response; a handler-initiated Flush does not trigger it. If nil, no
	// action is taken.
	OnBufferOverflow func(int)
}

// DefaultETagConfig returns an ETagConfig with sensible defaults: strong
// ETags via FNV-64a, 1 MB buffer, no skip predicate.
func DefaultETagConfig() ETagConfig {
	return ETagConfig{
		Strength:         Strong,
		MaxBufferSize:    defaultMaxBufferSize,
		HashFunc:         defaultHashFunc,
		SkipIfPresent:    false,
		Skip:             nil,
		OnError:          nil,
		OnETagGenerated:  nil,
		On304:            nil,
		OnBufferOverflow: nil,
	}
}

// defaultHashFunc computes FNV-64a of data and returns the lowercase hex
// encoding. FNV-64a is a non-cryptographic hash with a 64-bit output, making
// accidental collisions astronomically unlikely for practical response-body
// counts (birthday bound: ~4 billion).
//
// The hash.Hash contract guarantees Write never returns an error; if it does,
// that signals a broken implementation and we panic with a classified
// Orchestration error.
func defaultHashFunc(data []byte) string {
	h := fnv.New64a()

	_, err := h.Write(data)
	if err != nil {
		panic(codeHashWriteFailed.Orchestration(
			"fnv hash.Write returned an error, violating the hash.Hash contract that Write never fails",
		))
	}

	return hexEncodeUint64(h.Sum64())
}

// Validate checks the ETagConfig for invalid values.
// Returns a *errorfamily.Error classified as Rejection on failure.
//
// Validate is deliberately stricter than the middleware: New clamps a
// non-positive MaxBufferSize to the 1 MiB default, and an out-of-range
// Strength (anything but Strong or Weak) silently emits strong-form tags
// because only Weak triggers the W/ prefix — while Validate rejects both
// outright. A zero-value ETagConfig{} therefore fails Validate but runs
// fine through New. Call Validate when you want start-up hard failures for
// a config built from user input or a declarative source; never use it as
// a gate for configs you also hand to New.
func (c ETagConfig) Validate() error {
	if c.MaxBufferSize <= 0 {
		return newInvalidConfig().WithContextf("max_buffer_size", "%d", c.MaxBufferSize)
	}

	if !c.Strength.IsValid() {
		return newInvalidConfig().WithContextf("strength", "%d", c.Strength)
	}

	return nil
}

// New returns middleware that generates ETag headers based on response body
// content and handles If-None-Match conditional requests with 304 Not Modified.
//
// For GET and HEAD requests, the middleware buffers the response body,
// computes an ETag, and compares it against the If-None-Match header using
// RFC 7232 §2.3.2 weak comparison. On match, it returns 304 Not Modified with
// no body. On mismatch, it writes the buffered body with the ETag header.
//
// Responses exceeding MaxBufferSize are streamed without an ETag. Hijack and
// Flush calls switch to streaming mode immediately.
func New(cfg ETagConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				next.ServeHTTP(resp, req)

				return
			}

			if cfg.Skip != nil && cfg.Skip(req) {
				next.ServeHTTP(resp, req)

				return
			}

			writer := newETagWriter(resp, cfg)
			next.ServeHTTP(writer, req)

			writer.flush(req)
		})
	}
}

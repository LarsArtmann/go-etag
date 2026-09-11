package etag

import (
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Error codes for classified errors returned by ETag writer operations.
// All codes use the http. namespace and are compatible with go-error-family
// for behavioral classification (Transient vs Infrastructure) and retry decisions.
const (
	// ErrCodeETagWriteFailed is returned when the ETag writer fails to
	// write buffered or streamed data. Classified as Transient (retryable).
	ErrCodeETagWriteFailed = "http.etag_write_failed"

	// ErrCodeHijackUnsupported is returned when the underlying ResponseWriter
	// does not implement http.Hijacker. Classified as Infrastructure (not retryable).
	ErrCodeHijackUnsupported = "http.hijack_unsupported"

	// ErrCodeHijackFailed is returned when the underlying Hijack call fails.
	// Classified as Transient (retryable).
	ErrCodeHijackFailed = "http.hijack_failed"

	// ErrCodeInvalidConfig is returned when ETagConfig validation fails.
	// Classified as Rejection (bad input, not retryable).
	ErrCodeInvalidConfig = "http.etag_config_invalid"

	// ErrCodeHashWriteFailed is returned when the hash function fails to
	// accept data, violating the hash.Hash contract. Classified as
	// Orchestration (internal contract violation, not retryable).
	ErrCodeHashWriteFailed = "http.etag_hash_write_failed"
)

// Typed mirrors of the exported untyped string codes above, used for
// internal construction via the Code constructor and Wrap methods. The
// exported constants remain untyped strings for backward compatibility;
// these typed constants keep internal call sites compile-time grouped.
const (
	codeETagWriteFailed   = Code(ErrCodeETagWriteFailed)
	codeHijackUnsupported = Code(ErrCodeHijackUnsupported)
	codeHijackFailed      = Code(ErrCodeHijackFailed)
	codeInvalidConfig     = Code(ErrCodeInvalidConfig)
	codeHashWriteFailed   = Code(ErrCodeHashWriteFailed)
)

// msgInvalidConfig is the message shared by the ErrInvalidConfig sentinel and
// every context-bearing error Validate derives from it.
const msgInvalidConfig = "ETagConfig has an invalid field value"

// ErrInvalidConfig is the sentinel error returned by Validate when
// ETagConfig has an invalid field value. Declared as the error interface so
// errors.Is(err, ErrInvalidConfig) call sites match the sentinel guard.
// Validate returns a fresh classified error with the same code and family
// carrying context (e.g. the offending field value), so errors.Is matches
// by code and family.
var ErrInvalidConfig error = codeInvalidConfig.Rejection(msgInvalidConfig)

// newInvalidConfig returns a fresh invalid-config error carrying the
// sentinel's code and message; callers attach context with WithContextf.
// Fresh instances keep the package-level sentinel immutable, and errors.Is
// matches them against it by code and family.
func newInvalidConfig() *errorfamily.Error {
	return codeInvalidConfig.Rejection(msgInvalidConfig)
}

const (
	msgRetryMaySucceed           = "This is a Transient error — retrying may succeed."
	msgInfrastructureUnsupported = "This is an Infrastructure error — the runtime environment does not support this operation."
	msgCheckYourConfig           = "Check your ETagConfig values and try again."
	msgReportAsBug               = "This is likely a bug. Please report it if the problem persists."
)

// errorTemplates maps every error code this package can produce to its
// user-facing message template. Templates use {key} placeholders filled
// from the error's context. The completeness test in errors_test.go asserts
// every etag error code has an entry here.
//
//nolint:gochecknoglobals // immutable data table: populated once, never mutated
var errorTemplates = map[string]errorfamily.MessageTemplate{
	ErrCodeETagWriteFailed: {
		What:   "Failed to write ETag-buffered HTTP response",
		Why:    "The underlying ResponseWriter.Write call returned an error while streaming ETag data.",
		Fix:    "Check if the client disconnected or if the response buffer is full.",
		WayOut: msgRetryMaySucceed,
	},
	ErrCodeHijackUnsupported: {
		What:   "HTTP connection hijacking is not supported",
		Why:    "The underlying ResponseWriter does not implement the http.Hijacker interface.",
		Fix:    "Use a ResponseWriter that supports hijacking (e.g., net/http default writer).",
		WayOut: msgInfrastructureUnsupported,
	},
	ErrCodeHijackFailed: {
		What:   "Failed to hijack HTTP connection",
		Why:    "The underlying Hijack() call returned an error.",
		Fix:    "Check if the connection is still active and not already hijacked.",
		WayOut: msgRetryMaySucceed,
	},
	ErrCodeInvalidConfig: {
		What:   "ETag configuration is invalid",
		Why:    "One or more fields of ETagConfig have invalid values.",
		Fix:    "Review the ETagConfig field values and ensure MaxBufferSize is positive.",
		WayOut: msgCheckYourConfig,
	},
	ErrCodeHashWriteFailed: {
		What:   "Hash function failed to accept data",
		Why:    "The hash.Write call returned an error, which violates the hash.Hash contract that Write never fails.",
		Fix:    "This indicates a bug in the hash implementation. Report it to the library author.",
		WayOut: msgReportAsBug,
	},
}

// RegisterErrorClassifications maps stdlib HTTP sentinel errors relevant to
// response-writing middleware to their behavioral families, and registers
// error message templates for all etag error codes. Call once during program
// startup to enable classification of HTTP errors via errorfamily.Classify.
func RegisterErrorClassifications() {
	errorfamily.RegisterClassifications(map[error]errorfamily.Family{
		http.ErrNotSupported: errorfamily.Infrastructure,
		http.ErrAbortHandler: errorfamily.Transient,
	})

	for code, tmpl := range errorTemplates {
		errorfamily.RegisterTemplate(code, tmpl)
	}
}

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

// ErrInvalidConfig is the sentinel error returned by Validate when
// ETagConfig has an invalid field value. The concrete error returned by
// Validate is a clone of this sentinel with context (e.g. the offending
// field value), so errors.Is(err, ErrInvalidConfig) matches by code and family.
var ErrInvalidConfig = errorfamily.NewRejection(
	ErrCodeInvalidConfig,
	"ETagConfig has an invalid field value",
)

const (
	msgRetryMaySucceed           = "This is a Transient error — retrying may succeed."
	msgInfrastructureUnsupported = "This is an Infrastructure error — the runtime environment does not support this operation."
	msgCheckYourConfig           = "Check your ETagConfig values and try again."
	msgReportAsBug               = "This is likely a bug. Please report it if the problem persists."
)

func registerErrorTemplate(code, what, why, fix, wayOut string) {
	errorfamily.RegisterTemplate(code, errorfamily.MessageTemplate{
		What:   what,
		Why:    why,
		Fix:    fix,
		WayOut: wayOut,
	})
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

	registerAllErrorTemplates()
}

func registerAllErrorTemplates() {
	registerErrorTemplate(
		ErrCodeETagWriteFailed,
		"Failed to write ETag-buffered HTTP response",
		"The underlying ResponseWriter.Write call returned an error while streaming ETag data.",
		"Check if the client disconnected or if the response buffer is full.",
		msgRetryMaySucceed,
	)

	registerErrorTemplate(
		ErrCodeHijackUnsupported,
		"HTTP connection hijacking is not supported",
		"The underlying ResponseWriter does not implement the http.Hijacker interface.",
		"Use a ResponseWriter that supports hijacking (e.g., net/http default writer).",
		msgInfrastructureUnsupported,
	)

	registerErrorTemplate(
		ErrCodeHijackFailed,
		"Failed to hijack HTTP connection",
		"The underlying Hijack() call returned an error.",
		"Check if the connection is still active and not already hijacked.",
		msgRetryMaySucceed,
	)

	registerErrorTemplate(
		ErrCodeInvalidConfig,
		"ETag configuration is invalid",
		"One or more fields of ETagConfig have invalid values.",
		"Review the ETagConfig field values and ensure MaxBufferSize is positive.",
		msgCheckYourConfig,
	)

	registerErrorTemplate(
		ErrCodeHashWriteFailed,
		"Hash function failed to accept data",
		"The hash.Write call returned an error, which violates the hash.Hash contract that Write never fails.",
		"This indicates a bug in the hash implementation. Report it to the library author.",
		msgReportAsBug,
	)
}

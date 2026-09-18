package etag

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	errorfamily "github.com/larsartmann/go-error-family"
)

func ExampleNew() {
	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("ETag") != "")

	// Output: true
}

// ExampleNew_observabilityHooks derives a cache hit ratio from the On304 and
// OnETagGenerated hooks. The same pattern feeds OpenTelemetry counters or any
// other metrics backend without the library taking a dependency on one.
func ExampleNew_observabilityHooks() {
	var generated, cacheHits int

	cfg := DefaultETagConfig()
	cfg.OnETagGenerated = func(ETag) { generated++ }
	cfg.On304 = func(ETag) { cacheHits++ }

	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	missReq := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), missReq)

	hitReq := httptest.NewRequest(http.MethodGet, "/", nil)
	hitReq.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)
	handler.ServeHTTP(httptest.NewRecorder(), hitReq)

	fmt.Println("generated:", generated)
	fmt.Println("cache hits:", cacheHits)

	// Output:
	// generated: 2
	// cache hits: 1
}

func ExampleETag() {
	tag := NewETag("abc123", Strong)

	fmt.Println(tag)
	fmt.Println(tag.IsWeak())

	weak := NewETag("abc123", Weak)
	fmt.Println(weak)

	// Output:
	// "abc123"
	// false
	// W/"abc123"
}

// ExampleETagConfig_Validate pins the sentinel contract: Validate returns a
// fresh context-bearing error that errors.Is matches against ErrInvalidConfig
// by code and family, never the sentinel instance itself.
func ExampleETagConfig_Validate() {
	err := ETagConfig{MaxBufferSize: -1}.Validate()

	fmt.Println(errors.Is(err, ErrInvalidConfig))

	// Output: true
}

// ExampleCode shows a family constructor building a classified error from a
// typed code, and how callers read the classification back for retry and
// routing decisions.
func ExampleCode() {
	err := Code("http.rate_limited").Transient("client retries exhausted")

	fmt.Println(err.ErrorCode())
	fmt.Println(err.ErrorFamily() == errorfamily.Transient)
	fmt.Println(err.IsRetryable())

	// Output:
	// http.rate_limited
	// true
	// true
}

// ExampleDomainOf extracts the failing component from a classified error,
// and reports false for errors that carry no machine-readable code.
func ExampleDomainOf() {
	err := Code("http.etag_write_failed").WrapTransient(io.ErrClosedPipe, "stream cut")

	domain, ok := DomainOf(err)
	fmt.Println(domain, ok)

	_, ok = DomainOf(io.EOF)
	fmt.Println(ok)

	// Output:
	// http true
	// false
}

// ExampleInDomain routes errors by failing component without string parsing.
func ExampleInDomain() {
	err := Code("http.hijack_failed").Transient("connection already hijacked")

	if InDomain(err, Domain("http")) {
		fmt.Println("route to http-layer retry logic")
	}

	// Output: route to http-layer retry logic
}

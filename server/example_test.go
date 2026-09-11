package etag

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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

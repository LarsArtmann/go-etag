package etag

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	etagserver "github.com/larsartmann/go-etag/server"
)

// headerETag mirrors the server constant so the shim test does not reach into
// unexported server state.
const headerETag = "ETag"

// Compile-time identity pins: the shim's types must stay true aliases of the
// server package's types, so a v0.1.x import path keeps exchanging values with
// code already migrated to the new path. A change from alias to definition
// breaks these assignments instead of silently splitting the domain.

var (
	_ etagserver.ETag       = ETag{}
	_ etagserver.ETagConfig = ETagConfig{}
	_ etagserver.Middleware = Middleware(nil)
	_ etagserver.Strength   = Strength(0)
)

// TestShim_ConstantsMatchServer pins that every re-exported constant carries
// the same value as its server-package original, so error codes and strength
// markers stay interchangeable across import paths.
func TestShim_ConstantsMatchServer(t *testing.T) {
	t.Parallel()

	if Strong != etagserver.Strong {
		t.Errorf("Strong = %d, want %d", Strong, etagserver.Strong)
	}

	if Weak != etagserver.Weak {
		t.Errorf("Weak = %d, want %d", Weak, etagserver.Weak)
	}

	for _, code := range []struct {
		name     string
		shim     string
		original string
	}{
		{name: "ErrCodeETagWriteFailed", shim: ErrCodeETagWriteFailed, original: etagserver.ErrCodeETagWriteFailed},
		{name: "ErrCodeHijackUnsupported", shim: ErrCodeHijackUnsupported, original: etagserver.ErrCodeHijackUnsupported},
		{name: "ErrCodeHijackFailed", shim: ErrCodeHijackFailed, original: etagserver.ErrCodeHijackFailed},
		{name: "ErrCodeInvalidConfig", shim: ErrCodeInvalidConfig, original: etagserver.ErrCodeInvalidConfig},
		{name: "ErrCodeHashWriteFailed", shim: ErrCodeHashWriteFailed, original: etagserver.ErrCodeHashWriteFailed},
	} {
		t.Run(code.name, func(t *testing.T) {
			t.Parallel()

			if code.shim != code.original {
				t.Errorf("shim %s = %q, want %q", code.name, code.shim, code.original)
			}
		})
	}
}

// TestShim_ErrInvalidConfigIsTheServerSentinel pins that the shim's sentinel
// is the server package's sentinel value, so errors.Is matches across import
// paths.
func TestShim_ErrInvalidConfigIsTheServerSentinel(t *testing.T) {
	t.Parallel()

	if !errors.Is(ErrInvalidConfig, etagserver.ErrInvalidConfig) {
		t.Error("ErrInvalidConfig does not match the server sentinel")
	}

	cfg := DefaultETagConfig()
	cfg.MaxBufferSize = -1

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Validate error = %v, want ErrInvalidConfig through the shim", err)
	}
}

// TestShim_NewMiddlewareBehavesLikeServer pins the shim's primary contract:
// middleware built through the deprecated import path still generates ETags
// and answers conditional requests, exactly like the server package.
func TestShim_NewMiddlewareBehavesLikeServer(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.test/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	tag, ok := ParseETag(rec.Header().Get(headerETag))
	if !ok {
		t.Fatalf("ETag header %q failed to parse", rec.Header().Get(headerETag))
	}

	conditional := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	conditional.Header.Set("If-None-Match", tag.String())

	rec304 := httptest.NewRecorder()
	handler.ServeHTTP(rec304, conditional)

	if rec304.Code != http.StatusNotModified {
		t.Errorf("conditional status = %d, want 304 through the shim middleware", rec304.Code)
	}
}

// TestShim_EntityTagHelpersMatchServer pins the wrapper functions' behavior:
// each delegates to the server implementation with identical results.
func TestShim_EntityTagHelpersMatchServer(t *testing.T) {
	t.Parallel()

	tag := NewETag("abc", Weak)

	if got, want := tag.String(), etagserver.NewETag("abc", Weak).String(); got != want {
		t.Errorf("NewETag().String() = %q, want %q", got, want)
	}

	parsed, ok := ParseETag(`W/"abc"`)
	if !ok {
		t.Fatal("ParseETag(W/\"abc\") failed through the shim")
	}

	if !parsed.WeakEqual(tag) {
		t.Errorf("ParseETag opaque = %q, want %q", parsed.OpaqueTag(), tag.OpaqueTag())
	}

	list := ParseETagList(`"a", W/"b"`)
	if len(list) != 2 {
		t.Fatalf("ParseETagList length = %d, want 2", len(list))
	}

	if !MatchesIfNoneMatch(tag, `W/"abc"`) {
		t.Error("MatchesIfNoneMatch = false, want weak match through the shim")
	}

	if MatchesIfMatch(tag, `W/"abc"`) {
		t.Error("MatchesIfMatch = true, want no strong match through the shim")
	}
}

// TestShim_RegisterErrorClassificationsDoesNotPanic pins that the shim's
// registration entry point stays callable and idempotent.
func TestShim_RegisterErrorClassificationsDoesNotPanic(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()
	RegisterErrorClassifications()
}

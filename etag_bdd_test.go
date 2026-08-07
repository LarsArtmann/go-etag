package etag

import (
	"net/http"
	"testing"
)

// BDD-style specs mapping RFC 7232 sections to behavioral expectations.
// Uses Go's native t.Run naming to produce Describe/Context/It hierarchy
// in test output without third-party BDD dependencies.

func TestSpec_RFC7232_StrongComparison(t *testing.T) {
	t.Parallel()

	// RFC 7232 §2.3.2: "Two entity-tags are equivalent in the strong
	// comparison function only if both are not weak and their opaque-tags
	// match character-by-character."

	t.Run("strong vs strong same opaque", func(t *testing.T) {
		t.Parallel()

		a := NewETag("abc", Strong)
		b := NewETag("abc", Strong)

		if !a.StrongEqual(b) {
			t.Error("expected strong match")
		}
	})

	t.Run("weak vs weak same opaque", func(t *testing.T) {
		t.Parallel()

		a := NewETag("abc", Weak)
		b := NewETag("abc", Weak)

		if a.StrongEqual(b) {
			t.Error("expected no strong match when both are weak")
		}
	})

	t.Run("strong vs weak same opaque", func(t *testing.T) {
		t.Parallel()

		a := NewETag("abc", Strong)
		b := NewETag("abc", Weak)

		if a.StrongEqual(b) {
			t.Error("expected no strong match when one is weak")
		}
	})
}

func TestSpec_RFC7232_WeakComparison(t *testing.T) {
	t.Parallel()

	// RFC 7232 §2.3.2: "Two entity-tags are equivalent in the weak
	// comparison function if their opaque-tags match character-by-character,
	// regardless of either or both being tagged as weak."

	t.Run("strong vs strong same opaque", func(t *testing.T) {
		t.Parallel()

		if !NewETag("abc", Strong).WeakEqual(NewETag("abc", Strong)) {
			t.Error("expected weak match")
		}
	})

	t.Run("weak vs strong same opaque", func(t *testing.T) {
		t.Parallel()

		if !NewETag("abc", Weak).WeakEqual(NewETag("abc", Strong)) {
			t.Error("expected weak match regardless of strength")
		}
	})
}

func TestSpec_RFC7232_IfNoneMatch(t *testing.T) {
	t.Parallel()

	// RFC 7232 §3.2: If-None-Match uses weak comparison.
	// On match for GET/HEAD → 304 Not Modified.

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	t.Run("matching strong ETag returns 304", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusNotModified)
	})

	t.Run("matching weak ETag returns 304", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `W/"779a65e7023cd2e7"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusNotModified)
	})

	t.Run("non-matching ETag returns 200", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"deadbeef"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusOK)
	})

	t.Run("wildcard returns 304 for existing resource", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, "*")

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusNotModified)
	})

	t.Run("list containing matching ETag returns 304", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"other", "779a65e7023cd2e7"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusNotModified)
	})
}

func TestSpec_RFC7232_NotModifiedResponse(t *testing.T) {
	t.Parallel()

	// RFC 7232 §4.1: A 304 response MUST generate ETag and MUST NOT contain
	// a message body. Content-Length must be stripped.

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentLength, "11")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	t.Run("includes ETag header", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Header().Get(headerETag) == "" {
			t.Error("304 response must include ETag header")
		}
	})

	t.Run("excludes Content-Length", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Header().Get(headerContentLength) != "" {
			t.Errorf("304 response must not include Content-Length, got %q", rec.Header().Get(headerContentLength))
		}
	})

	t.Run("body is empty", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

		rec := newRecorder()
		handler.ServeHTTP(rec, req)

		assertBodyEmpty(t, rec, "304 must have no body")
	})
}

func TestSpec_RFC7232_IfMatch(t *testing.T) {
	t.Parallel()

	// RFC 7232 §3.1: If-Match uses strong comparison.
	// Applications use MatchesIfMatch in their handlers for unsafe methods.

	tag := NewETag("abc", Strong)

	t.Run("strong match", func(t *testing.T) {
		t.Parallel()

		if !MatchesIfMatch(tag, `"abc"`) {
			t.Error("expected strong match")
		}
	})

	t.Run("weak tag does not match", func(t *testing.T) {
		t.Parallel()

		if MatchesIfMatch(tag, `W/"abc"`) {
			t.Error("weak tag must not strongly match")
		}
	})

	t.Run("wildcard matches any existing resource", func(t *testing.T) {
		t.Parallel()

		if !MatchesIfMatch(tag, "*") {
			t.Error("wildcard should match")
		}
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()

		if MatchesIfMatch(tag, `"different"`) {
			t.Error("non-matching tag should not match")
		}
	})
}

func TestSpec_RFC7232_EntityTagFormat(t *testing.T) {
	t.Parallel()

	// RFC 7232 §2.3: entity-tag = [ weak ] opaque-tag
	// weak = %x57.2F (W/)
	// opaque-tag = DQUOTE *etagc DQUOTE

	t.Run("strong tag wire format", func(t *testing.T) {
		t.Parallel()

		got := NewETag("xyz", Strong).String()
		want := `"xyz"`

		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("weak tag wire format", func(t *testing.T) {
		t.Parallel()

		got := NewETag("xyz", Weak).String()
		want := `W/"xyz"`

		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("parse round-trips through String", func(t *testing.T) {
		t.Parallel()

		original := NewETag("round-trip-42", Weak)
		parsed, ok := ParseETag(original.String())
		if !ok {
			t.Fatal("ParseETag failed for round-trip")
		}

		if !parsed.WeakEqual(original) {
			t.Error("round-trip parse should produce equal tag")
		}

		if parsed.IsWeak() != original.IsWeak() {
			t.Error("round-trip parse should preserve strength")
		}
	})
}

func TestSpec_RFC7232_HeadRequest(t *testing.T) {
	t.Parallel()

	// RFC 7230 §3.3: HEAD responses MUST NOT include a message body.
	// The middleware should still compute ETag and set Content-Length.

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	t.Run("HEAD has no body", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodHead)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertBodyEmpty(t, rec, "HEAD must not have a body")
	})

	t.Run("HEAD has ETag", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodHead)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Header().Get(headerETag) == "" {
			t.Error("HEAD response should include ETag")
		}
	})

	t.Run("HEAD has Content-Length", func(t *testing.T) {
		t.Parallel()

		req := newTestRequest(http.MethodHead)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Header().Get(headerContentLength) == "" {
			t.Error("HEAD response should include Content-Length")
		}
	})
}

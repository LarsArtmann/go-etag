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

func TestSpec_RFC7232_NotModifiedMetadata(t *testing.T) {
	t.Parallel()

	// RFC 7232 §4.1: "The server generating a 304 response MUST generate any
	// of the following header fields that would have been sent in a 200 (OK)
	// response to the same request: Cache-Control, Content-Location, Date,
	// ETag, Expires, and Vary." Handler-set metadata therefore survives the
	// 304 alongside the mandatory ETag.

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("Content-Location", "/versions/1")
		w.Header().Set("Date", "date-of-representation")
		w.Header().Set("Expires", "expiration-of-representation")
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)

	for name, want := range map[string]string{
		"Cache-Control":    "max-age=60",
		"Content-Location": "/versions/1",
		"Date":             "date-of-representation",
		"ETag":             `"779a65e7023cd2e7"`,
		"Expires":          "expiration-of-representation",
		"Vary":             "Accept-Encoding",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q (RFC 7232 §4.1: 304 must carry the 200's metadata)", name, got, want)
		}
	}
}

// hookCall names used by hookRecorder to record observability event order.
const (
	hookCallGenerated = "generated"
	hookCall304       = "304"
	hookCallOverflow  = "overflow"
)

// overflowSpecLimit is a deliberately tiny MaxBufferSize so overflow specs can
// cross it with short bodies.
const overflowSpecLimit = 8

func (r *hookRecorder) attach(cfg *ETagConfig) {
	cfg.OnETagGenerated = func(tag ETag) {
		r.generated = append(r.generated, tag)
		r.order = append(r.order, hookCallGenerated)
	}

	cfg.On304 = func(tag ETag) {
		r.notModified = append(r.notModified, tag)
		r.order = append(r.order, hookCall304)
	}

	cfg.OnBufferOverflow = func(limit int) {
		r.overflows = append(r.overflows, limit)
		r.order = append(r.order, hookCallOverflow)
	}
}

func (r *hookRecorder) attachToDefault() ETagConfig {
	cfg := DefaultETagConfig()
	r.attach(&cfg)

	return cfg
}

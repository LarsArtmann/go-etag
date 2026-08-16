package etag

import (
	"net/http"
	"slices"
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

// hookCall names used by hookRecorder to record observability event order.
const (
	hookCallGenerated = "generated"
	hookCall304       = "304"
	hookCallOverflow  = "overflow"
)

// overflowSpecLimit is a deliberately tiny MaxBufferSize so overflow specs can
// cross it with short bodies.
const overflowSpecLimit = 8

// hookRecorder captures observability hook invocations in call order so specs
// can assert both counts and event ordering. Hooks fire synchronously in the
// request goroutine and specs serve requests synchronously, so no locking is
// needed.
type hookRecorder struct {
	generated   []ETag
	notModified []ETag
	overflows   []int
	order       []string
}

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

func TestSpec_ObservabilityHookFiring(t *testing.T) {
	t.Parallel()

	// Hook contract: OnETagGenerated fires exactly once per computed tag,
	// On304 fires additionally after a 304 is committed, and OnBufferOverflow
	// fires exactly once when the buffer limit is crossed. Nil hooks must be
	// inert. Handler-provided tags (SkipIfPresent) never count as generated.

	t.Run("nil hooks leave behavior unchanged", func(t *testing.T) {
		t.Parallel()

		handler := New(DefaultETagConfig())(newWriteBodyHandler([]byte("hello world")))

		req := newTestRequest(http.MethodGet)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusOK)
		assertBody(t, rec, "hello world")

		if rec.Header().Get(headerETag) == "" {
			t.Error("ETag must still be set when all hooks are nil")
		}
	})

	t.Run("OnETagGenerated fires once with the computed tag", func(t *testing.T) {
		t.Parallel()

		calls := &hookRecorder{}
		handler := New(calls.attachToDefault())(newWriteBodyHandler([]byte("hello world")))

		req := newTestRequest(http.MethodGet)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if len(calls.generated) != 1 {
			t.Fatalf("generated calls = %d, want 1", len(calls.generated))
		}

		want, ok := ParseETag(rec.Header().Get(headerETag))
		if !ok {
			t.Fatalf("response ETag header %q failed to parse", rec.Header().Get(headerETag))
		}

		if !want.WeakEqual(calls.generated[0]) {
			t.Errorf("generated tag = %v, want header tag %v", calls.generated[0], want)
		}

		if len(calls.order) != 1 || calls.order[0] != hookCallGenerated {
			t.Errorf("order = %v, want [%s]", calls.order, hookCallGenerated)
		}
	})

	t.Run("HEAD request fires OnETagGenerated", func(t *testing.T) {
		t.Parallel()

		calls := &hookRecorder{}
		handler := New(calls.attachToDefault())(newWriteBodyHandler([]byte("hello world")))

		req := newTestRequest(http.MethodHead)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if len(calls.generated) != 1 {
			t.Errorf("generated calls = %d, want 1 for HEAD", len(calls.generated))
		}
	})

	t.Run("On304 fires after OnETagGenerated on a cache hit", func(t *testing.T) {
		t.Parallel()

		calls := &hookRecorder{}
		handler := New(calls.attachToDefault())(newWriteBodyHandler([]byte("hello world")))

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusNotModified)

		wantOrder := []string{hookCallGenerated, hookCall304}
		if !slices.Equal(calls.order, wantOrder) {
			t.Errorf("order = %v, want %v", calls.order, wantOrder)
		}

		if len(calls.notModified) != 1 {
			t.Fatalf("304 calls = %d, want 1", len(calls.notModified))
		}

		if calls.notModified[0].String() != `"779a65e7023cd2e7"` {
			t.Errorf("304 tag = %v, want the matching tag", calls.notModified[0])
		}
	})

	t.Run("SkipIfPresent adoption fires On304 but not OnETagGenerated", func(t *testing.T) {
		t.Parallel()

		cfg := ETagConfig{
			Strength:      Strong,
			MaxBufferSize: defaultMaxBufferSize,
			SkipIfPresent: true,
		}

		calls := &hookRecorder{}
		calls.attach(&cfg)

		handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(headerETag, `"handler-tag"`)
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte("hello world"))
		}))

		req := newTestRequest(http.MethodGet)
		req.Header.Set(headerIfNoneMatch, `"handler-tag"`)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		assertStatus(t, rec, http.StatusNotModified)

		if len(calls.generated) != 0 {
			t.Errorf("generated calls = %d, want 0 for adopted handler tag", len(calls.generated))
		}

		if len(calls.notModified) != 1 || calls.notModified[0].String() != `"handler-tag"` {
			t.Errorf("304 calls = %v, want one with the handler tag", calls.notModified)
		}
	})

	t.Run("OnBufferOverflow fires once with the exceeded limit", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultETagConfig()
		cfg.MaxBufferSize = overflowSpecLimit

		calls := &hookRecorder{}
		calls.attach(&cfg)

		handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte("0123456")) // 7 bytes: still buffered.
			_, _ = w.Write([]byte("89"))      // 9 bytes: crosses the limit.
			_, _ = w.Write([]byte("abcdef"))  // streaming: must not re-fire.
		}))

		req := newTestRequest(http.MethodGet)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if len(calls.overflows) != 1 || calls.overflows[0] != overflowSpecLimit {
			t.Errorf("overflow calls = %v, want [%d] exactly once", calls.overflows, overflowSpecLimit)
		}

		assertBody(t, rec, "012345689abcdef")
		assertETagEmpty(t, rec, "for an overflowed streaming response")
	})
}

func TestSpec_ObservabilityHookSuppression(t *testing.T) {
	t.Parallel()

	// Suppression contract: streaming responses, bypassed methods, and
	// skipped requests must not fire any observability hook.

	t.Run("handler Flush streams without firing OnBufferOverflow", func(t *testing.T) {
		t.Parallel()

		calls := &hookRecorder{}
		handler := New(calls.attachToDefault())(newFlushHandler())

		req := newTestRequest(http.MethodGet)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if len(calls.overflows) != 0 {
			t.Errorf("overflow calls = %d, want 0 for handler-initiated Flush", len(calls.overflows))
		}

		if len(calls.generated) != 0 {
			t.Errorf("generated calls = %d, want 0 for streamed response", len(calls.generated))
		}
	})

	t.Run("POST requests fire no hooks", func(t *testing.T) {
		t.Parallel()

		calls := &hookRecorder{}
		handler := New(calls.attachToDefault())(newWriteBodyHandler([]byte("hello world")))

		req := newTestRequest(http.MethodPost)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if len(calls.order) != 0 {
			t.Errorf("hook calls = %v, want none for POST", calls.order)
		}
	})

	t.Run("Skip predicate requests fire no hooks", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultETagConfig()
		cfg.Skip = func(*http.Request) bool { return true }

		calls := &hookRecorder{}
		calls.attach(&cfg)

		handler := New(cfg)(newWriteBodyHandler([]byte("hello world")))

		req := newTestRequest(http.MethodGet)
		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if len(calls.order) != 0 {
			t.Errorf("hook calls = %v, want none for skipped request", calls.order)
		}
	})
}

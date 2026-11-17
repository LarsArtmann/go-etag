package etag

import (
	"net/http"
	"slices"
	"testing"
)

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

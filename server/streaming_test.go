package etag

import (
	"net/http"
	"testing"
)

func TestNew_Flush(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newFlushHandler())

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETagEmpty(t, rec, "after flush")
	assertBody(t, rec, "partial more")
}

func TestNew_FlushAlreadyFlushed(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("data"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()
		flusher.Flush()
	}))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)
}

func TestNew_Hijack_SetsFlushedMode(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()
	writer := newETagWriter(newHijackRecorder(), cfg)

	_, _, err := writer.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v, want nil", err)
	}

	if !writer.flushed {
		t.Error("Hijack should set flushed mode")
	}
}

func TestNew_Hijack_NoETag(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		hj, ok := rw.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Hijacker")
		}

		_, _, _ = hj.Hijack()
	}))

	rec := newHijackRecorder()
	req := newTestRequest(http.MethodGet)

	handler.ServeHTTP(rec, req)

	assertETagEmpty(t, rec.ResponseRecorder, "after hijack")
}

// --- D1: SkipIfPresent ---

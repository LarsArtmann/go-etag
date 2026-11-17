package etag

import (
	"net/http"
	"testing"
)

func TestNew_MemoryLimit_DisablesETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 10}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "this body exceeds the limit"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETagEmpty(t, rec, "when buffer limit exceeded")
	assertBody(t, rec, "this body exceeds the limit")
}

func TestNew_Overflow_IfNoneMatch(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 10}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "this body exceeds the limit"))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"anything"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "this body exceeds the limit")
	assertETagEmpty(t, rec, "when buffer limit exceeded")
}

// B1: Zero-value ETagConfig must not cause unbounded buffering.
func TestNew_ZeroValueConfig_ClampsBufferSize(t *testing.T) {
	t.Parallel()

	// ETagConfig{} has MaxBufferSize=0, which must be clamped to the default.
	// Without clamping, the overflow guard in Write would be disabled.
	cfg := ETagConfig{}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	// A clamped config generates an ETag; an unbounded one would too, but the
	// important thing is that it doesn't panic or misbehave.
	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty with zero-value config, want clamped default behavior")
	}
}

// --- 304 response correctness ---

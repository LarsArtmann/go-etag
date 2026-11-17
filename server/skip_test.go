package etag

import (
	"net/http"
	"testing"
)

func TestNew_SkipIfPresent_RespectsHandlerETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, `"my-revision-42"`)
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"my-revision-42"`)
}

func TestNew_SkipIfPresent_Allows304WithHandlerETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, `"my-revision-42"`)
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"my-revision-42"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
}

func TestNew_SkipIfPresent_FallsBackWhenNoHandlerETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	// No handler-set ETag, so middleware computes one.
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestNew_SkipIfPresent_MalformedHandlerETag_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, "not-a-valid-etag")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	// Malformed handler ETag cannot be parsed, so middleware computes one.
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestNew_SkipIfPresent_EmptyHandlerETag_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{SkipIfPresent: true}
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerETag, "")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

// --- D6: Skip predicate ---

func TestNew_SkipPredicate(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/stream"
		},
	}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.URL.Path = "/stream"
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETagEmpty(t, rec, "when Skip returns true")
	assertBody(t, rec, "hello world")
}

func TestNew_SkipPredicate_False(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{
		Skip: func(r *http.Request) bool {
			return false
		},
	}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

// --- Validate ---

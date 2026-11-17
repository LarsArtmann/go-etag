package etag

import (
	"net/http"
	"testing"
)

// --- ETag generation ---

func TestNew_GeneratesStrongETag(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestNew_GeneratesWeakETag(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Strength: Weak}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `W/"779a65e7023cd2e7"`)
}

func TestNew_EmptyBody(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, ""))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"cbf29ce484222325"`)
}

// --- If-None-Match ---

func TestNew_NoIfNoneMatchHeader(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "hello world")
	assertETag(t, rec, `"779a65e7023cd2e7"`)
}

func TestNew_NonGetHead(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodPost)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETagEmpty(t, rec, "for POST")
}

func TestNew_HeadRequest_GeneratesETag(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodHead)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for HEAD")
	}
}

// B2: HEAD responses MUST NOT include a message body (RFC 7230 §3.3).
func TestNew_HeadRequest_NoBody(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodHead)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertBodyEmpty(t, rec, "HEAD must not have a body")

	// Content-Length should reflect the representation size.
	if cl := rec.Header().Get(headerContentLength); cl != "11" {
		t.Errorf("Content-Length = %q, want %q for HEAD", cl, "11")
	}
}

func TestNew_HeadRequest_IfNoneMatch(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodHead)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "for HEAD 304")
}

// --- Status codes ---

func TestNew_201Created_IsCacheable(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusCreated, "created"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get(headerETag)
	if etag == "" {
		t.Error("ETag header is empty, want generated ETag for 201")
	}
}

// Non-cacheable status codes (3xx+) must never return 304, even when
// If-None-Match matches the body hash. The ETag is still set for client
// caching, but the full response is always sent.
func TestNew_NonCacheableStatus_NeverReturns304(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
	}{
		{name: "301 Moved Permanently", status: http.StatusMovedPermanently},
		{name: "302 Found", status: http.StatusFound},
		{name: "304 Not Modified", status: http.StatusNotModified},
		{name: "400 Bad Request", status: http.StatusBadRequest},
		{name: "404 Not Found", status: http.StatusNotFound},
		{name: "500 Internal Server Error", status: http.StatusInternalServerError},
		{name: "503 Service Unavailable", status: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := New(DefaultETagConfig())(newWriteStatusHandler(tt.status, "hello world"))

			req := newTestRequest(http.MethodGet)
			req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

			rec := newRecorder()
			handler.ServeHTTP(rec, req)

			assertStatus(t, rec, tt.status)
			assertBody(t, rec, "hello world")
		})
	}
}

// Boundary: 299 is the last cacheable status (200-299 range).
func TestNew_Status299_IsCacheable(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(299, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "299 is cacheable")
}

func TestNew_Status300_NotCacheable(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusMultipleChoices, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusMultipleChoices)
	assertBody(t, rec, "hello world")
}

// --- Buffer overflow ---

func TestNew_EmptyHandler_NoETag(t *testing.T) {
	t.Parallel()

	// A handler that writes nothing and never calls WriteHeader hits the
	// computeETag early return for empty body + no buffered header.
	handler := New(DefaultETagConfig())(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertETagEmpty(t, rec, "for handler that writes nothing")
	assertBodyEmpty(t, rec, "for handler that writes nothing")
}

func TestNew_HandlerWriteHeaderOnly_WithBody(t *testing.T) {
	t.Parallel()

	// A handler that calls WriteHeader but writes no body should still get
	// an ETag (the empty body hash).
	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	// Empty body still produces an ETag (FNV-64a of empty string).
	assertETag(t, rec, `"cbf29ce484222325"`)
}

// --- Fuzz ---

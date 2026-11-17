package etag

import (
	"net/http"
	"testing"
)

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

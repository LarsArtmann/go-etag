package etag

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newWriteStatusHandler returns an http.HandlerFunc that writes status and body.
func newWriteStatusHandler(status int, body string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

// newWriteBodyHandler returns an http.HandlerFunc that writes OK status and body.
func newWriteBodyHandler(body []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// newTestRequest creates an httptest.Request with context and Origin header set.
func newTestRequest(method, path, origin string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	return req
}

// newRecorder creates a new httptest.ResponseRecorder.
func newRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// newFlushHandler returns an http.HandlerFunc that writes "partial", flushes
// if the ResponseWriter implements http.Flusher, then writes " more".
func newFlushHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		_, _ = w.Write([]byte(" more"))
	})
}

// hijackRecorder is an httptest.ResponseRecorder that also implements http.Hijacker.
type hijackRecorder struct {
	*httptest.ResponseRecorder

	hijacked bool
}

func newHijackRecorder() *hijackRecorder {
	return &hijackRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (r *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.hijacked = true

	return nil, nil, nil
}

// errMockWriteFailed is the sentinel returned by failingResponseRecorder.Write.
var errMockWriteFailed = errors.New("mock write failed")

// failingResponseRecorder is an httptest.ResponseRecorder whose Write always
// fails, exercising the streaming-write and overflow-write error branches
// that require the underlying ResponseWriter to reject data.
type failingResponseRecorder struct {
	*httptest.ResponseRecorder
}

func (*failingResponseRecorder) Write([]byte) (int, error) {
	return 0, errMockWriteFailed
}

// assertStatus checks that a response recorder has the expected status code.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rec.Code != want {
		t.Errorf("status = %d, want %d", rec.Code, want)
	}
}

// assertBody checks that a response recorder body matches the expected string.
func assertBody(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := rec.Body.String(); got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// assertBodyEmpty checks that a response recorder has no body, formatted with
// msg to clarify the test intent (e.g. "for 304").
func assertBodyEmpty(t *testing.T, rec *httptest.ResponseRecorder, msg string) {
	t.Helper()

	if rec.Body.Len() != 0 {
		t.Errorf("body length = %d, want 0 %s", rec.Body.Len(), msg)
	}
}

// assertETagEmpty checks that a response recorder has no ETag header,
// formatted with msg to clarify the test intent (e.g. "for POST").
func assertETagEmpty(t *testing.T, rec *httptest.ResponseRecorder, msg string) {
	t.Helper()

	if got := rec.Header().Get(headerETag); got != "" {
		t.Errorf("ETag = %q, want empty %s", got, msg)
	}
}

// assertBodyContains checks that a response recorder body contains substr.
func assertBodyContains(t *testing.T, rec *httptest.ResponseRecorder, substr string) {
	t.Helper()

	if !bytes.Contains(rec.Body.Bytes(), []byte(substr)) {
		t.Errorf("body does not contain %q, got %q", substr, rec.Body.String())
	}
}

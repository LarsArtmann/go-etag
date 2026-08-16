package etag

import (
	"bufio"
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
func newTestRequest(method string) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), method, "/", nil)
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

// failingHijackRecorder is an httptest.ResponseRecorder that implements
// http.Hijacker but always returns an error from Hijack, exercising the
// "hijack failed" error path in hijackDelegate.
type failingHijackRecorder struct {
	*httptest.ResponseRecorder
}

func newFailingHijackRecorder() *failingHijackRecorder {
	return &failingHijackRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (*failingHijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errMockHijackFailed
}

// nonHijackableRecorder is a minimal http.ResponseWriter that does NOT
// implement http.Hijacker, exercising the "hijack unsupported" error path.
type nonHijackableRecorder struct {
	header http.Header
	status int
	body   []byte
}

func newNonHijackableRecorder() *nonHijackableRecorder {
	return &nonHijackableRecorder{header: http.Header{}}
}

func (r *nonHijackableRecorder) Header() http.Header { return r.header }

func (r *nonHijackableRecorder) WriteHeader(code int) { r.status = code }

func (r *nonHijackableRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)

	return len(b), nil
}

// errMockWriteFailed is the sentinel returned by failingResponseRecorder.Write.
var errMockWriteFailed = errors.New("mock write failed")

// errMockHijackFailed is the sentinel returned by failingHijackRecorder.Hijack.
var errMockHijackFailed = errors.New("mock hijack failed")

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

// assertETag checks that a response recorder has the expected ETag header.
func assertETag(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := rec.Header().Get(headerETag); got != want {
		t.Errorf("ETag = %q, want %q", got, want)
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

// serveGetWithIfNoneMatch wraps the default ETag middleware around a body
// handler, issues a GET request with the given If-None-Match header value,
// serves it, and returns the recorder. Used by the If-None-Match cache
// validation tests to factor out the identical request scaffolding.
func serveGetWithIfNoneMatch(t *testing.T, ifNoneMatch string) *httptest.ResponseRecorder {
	t.Helper()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, ifNoneMatch)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	return rec
}

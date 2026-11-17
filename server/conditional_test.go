package etag

import (
	"net/http"
	"testing"
)

func TestNew_IfNoneMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		want304 bool
	}{
		{name: "ExactMatch", header: `"779a65e7023cd2e7"`, want304: true},
		{name: "Wildcard", header: "*", want304: true},
		{name: "ListContainsMatch", header: `"other", "779a65e7023cd2e7", "another"`, want304: true},
		{name: "WeakClientStrongServer", header: `W/"779a65e7023cd2e7"`, want304: true},
		{name: "ListContainsWeakMatch", header: `"other", W/"779a65e7023cd2e7", "another"`, want304: true},
		{name: "StrongClientNoMatch", header: `"different"`, want304: false},
		{name: "WeakClientNoMatch", header: `W/"different"`, want304: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := serveGetWithIfNoneMatch(t, tt.header)

			if tt.want304 {
				assertStatus(t, rec, http.StatusNotModified)
				assertBodyEmpty(t, rec, "for 304")
			} else {
				assertStatus(t, rec, http.StatusOK)
				assertBody(t, rec, "hello world")
			}
		})
	}
}

func TestNew_IfNoneMatch_StrongClientWeakServer(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Strength: Weak}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "for strong If-None-Match against weak server ETag")
}

func TestNew_IfNoneMatch_MultipleHeaders(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.Header.Add(headerIfNoneMatch, `"other"`)
	req.Header.Add(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)
	assertBodyEmpty(t, rec, "for 304 with multiple If-None-Match headers")
}

func TestNew_IfNoneMatch_EscapedQuoteEndToEnd(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"a\"b", "c"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "hello world")
}

// --- HTTP methods ---

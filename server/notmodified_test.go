package etag

import (
	"net/http"
	"testing"
)

// B3: 304 Not Modified MUST NOT include Content-Length.
func TestNew_304_ExcludesContentLength(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerContentLength, "11")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))
	}))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotModified)

	if cl := rec.Header().Get(headerContentLength); cl != "" {
		t.Errorf("Content-Length = %q on 304, want empty", cl)
	}
}

func TestNew_304_IncludesETagHeader(t *testing.T) {
	t.Parallel()

	rec := serveGetWithIfNoneMatch(t, `"779a65e7023cd2e7"`)

	assertStatus(t, rec, http.StatusNotModified)

	if rec.Header().Get(headerETag) == "" {
		t.Error("ETag header is empty on 304, want the generated ETag")
	}
}

// --- Flush / Hijack ---

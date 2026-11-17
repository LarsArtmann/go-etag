package etag

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func FuzzETag(f *testing.F) {
	f.Add([]byte("hello world"), "")
	f.Add([]byte(strings.Repeat("a", 1024)), `"abc123"`)
	f.Add([]byte(""), "*")
	f.Add([]byte("response body"), `W/"779a65e7023cd2e7"`)
	f.Add([]byte("response body"), `"779a65e7023cd2e7", W/"deadbeef"`)
	f.Add([]byte("body"), `"a\"b"`)
	f.Add([]byte("body"), `"a\"b", "c"`)
	f.Add([]byte("body"), `"a,b\"c"`)

	cfg := DefaultETagConfig()
	inner := newWriteStatusHandler(http.StatusOK, "response body")

	f.Fuzz(func(t *testing.T, body []byte, ifNoneMatch string) {
		handler := New(cfg)(inner)

		req := newTestRequest(http.MethodGet)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set(headerIfNoneMatch, ifNoneMatch)

		rec := newRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK && rec.Code != http.StatusNotModified {
			t.Errorf("status = %d, want 200 or 304", rec.Code)
		}
	})
}

// --- Benchmarks ---

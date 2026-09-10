package etag

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// wireResult carries the response facts the wire round trip asserts on, so
// the helper can close the body itself and leak no responses.
type wireResult struct {
	status        int
	header        http.Header
	contentLength int64
	body          string
}

// TestIntegrationRealServerWireRoundTrip runs the middleware behind a real
// net/http server so the wire facts the recorder cannot see are pinned:
// Content-Length framing on 200s, body suppression on HEAD, a bodiless 304
// without Content-Length, and header canonicalization on the live path.
func TestIntegrationRealServerWireRoundTrip(t *testing.T) {
	t.Parallel()

	const payload = "wire payload"

	handler := New(DefaultETagConfig())(newWriteStatusHandler(http.StatusOK, payload))

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := server.Client()

	fetchWire := func(method string, ifNoneMatch string) wireResult {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), method, server.URL, nil)
		if err != nil {
			t.Fatalf("build %s request: %v", method, err)
		}

		if ifNoneMatch != "" {
			req.Header.Set(headerIfNoneMatch, ifNoneMatch)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}

		defer resp.Body.Close()

		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("read %s body: %v", method, readErr)
		}

		return wireResult{
			status:        resp.StatusCode,
			header:        resp.Header,
			contentLength: resp.ContentLength,
			body:          string(data),
		}
	}

	t.Run("200 frames the body with Content-Length", func(t *testing.T) {
		t.Parallel()

		got := fetchWire(http.MethodGet, "")
		if got.status != http.StatusOK || got.body != payload {
			t.Fatalf("response = %d %q, want 200 %q", got.status, got.body, payload)
		}

		if got.contentLength != int64(len(payload)) {
			t.Errorf("ContentLength = %d, want %d framed on the wire", got.contentLength, len(payload))
		}

		if got.header.Get(headerETag) == "" {
			t.Error("ETag missing on the live 200")
		}
	})

	t.Run("conditional GET answers a bodiless 304 without Content-Length", func(t *testing.T) {
		t.Parallel()

		etag := fetchWire(http.MethodGet, "").header.Get(headerETag)

		got := fetchWire(http.MethodGet, etag)

		if got.status != http.StatusNotModified {
			t.Fatalf("status = %d, want 304 over the wire", got.status)
		}

		if got.body != "" {
			t.Errorf("body = %q, want a bodiless 304 (RFC 7232 §4.1)", got.body)
		}

		if got.contentLength != 0 {
			t.Errorf("ContentLength = %d, want 0 (length must not advertise the hidden body)", got.contentLength)
		}

		if name := got.header.Get("Content-Length"); name != "" {
			t.Errorf("Content-Length header = %q, want absent on the 304", name)
		}

		if name := got.header.Get(headerETag); name != etag {
			t.Errorf("ETag = %q, want %q on the 304 (RFC 7232 §4.1)", name, etag)
		}
	})

	t.Run("HEAD suppresses the body but frames the length", func(t *testing.T) {
		t.Parallel()

		got := fetchWire(http.MethodHead, "")

		if got.status != http.StatusOK {
			t.Fatalf("status = %d, want 200", got.status)
		}

		if got.body != "" {
			t.Errorf("body = %q, want empty (HEAD has no body on the wire)", got.body)
		}

		if got.header.Get("Content-Length") == "" {
			t.Error("Content-Length missing on HEAD, want the representation size advertised")
		}

		if got.header.Get(headerETag) == "" {
			t.Error("ETag missing on HEAD")
		}
	})

	t.Run("conditional HEAD also answers 304", func(t *testing.T) {
		t.Parallel()

		etag := fetchWire(http.MethodGet, "").header.Get(headerETag)

		got := fetchWire(http.MethodHead, etag)

		if got.status != http.StatusNotModified {
			t.Fatalf("status = %d, want 304 for a conditional HEAD", got.status)
		}

		if got.body != "" {
			t.Errorf("body = %q, want empty", got.body)
		}
	})
}

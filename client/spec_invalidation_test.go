package etagclient

import (
	"net/http"
	"testing"
)

// TestSpecUnsafeMethodInvalidatesEntry pins RFC 9111 §4.4: "A cache MUST
// invalidate the target URI ... when it receives a non-error status code in
// response to an unsafe request method" — a mutation cannot leave a
// pre-mutation body waiting to be rebuilt. Non-error means 2xx/3xx; error
// responses and safe methods (GET/OPTIONS/TRACE) leave the entry alone.
// HEAD is safe too, but it has its own freshening rules (RFC 9111 §4.3.5,
// pinned by TestSpecHeadFreshening).
func TestSpecUnsafeMethodInvalidatesEntry(t *testing.T) {
	t.Parallel()

	for _, spec := range []struct {
		name        string
		method      string
		status      int
		invalidated bool
	}{
		{name: "2xx after DELETE invalidates", method: http.MethodDelete, status: http.StatusNoContent, invalidated: true},
		{name: "3xx after POST invalidates", method: http.MethodPost, status: http.StatusFound, invalidated: true},
		{name: "error after POST keeps the entry", method: http.MethodPost, status: http.StatusInternalServerError, invalidated: false},
		{name: "safe OPTIONS keeps the entry", method: http.MethodOptions, status: http.StatusOK, invalidated: false},
	} {
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()

			next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("If-None-Match") == `"v"` {
					return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
				}

				if req.Method != http.MethodGet {
					return stubResponse(spec.status, stubHeader(), ""), nil
				}

				return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), "payload"), nil
			})

			transport := NewTransport(next, Options{})
			url := "https://example.test/thing"

			fetch(t, transport, newGetRequest(t, url))

			status, _, _ := fetch(t, transport, newSpecRequest(t, spec.method, url))
			if status != spec.status {
				t.Fatalf("%s status = %d, want passthrough %d", spec.method, status, spec.status)
			}

			wantEntries := 1
			if spec.invalidated {
				wantEntries = 0
			}

			if got := transport.Stats().Entries; got != wantEntries {
				t.Fatalf("entries after %s = %d, want %d (RFC 9111 §4.4)", spec.method, got, wantEntries)
			}

			// After invalidation the next GET refetches (a 200); with the entry
			// kept it revalidates (a 304 rebuilt from cache).
			status, _, _ = fetch(t, transport, newGetRequest(t, url))

			wantHits := int64(1)
			if spec.invalidated {
				wantHits = 0
			}

			if got := transport.Stats().Hits; got != wantHits {
				t.Errorf("hits after re-GET = %d, want %d", got, wantHits)
			}

			if status != http.StatusOK {
				t.Errorf("re-GET status = %d, want 200", status)
			}
		})
	}
}

// TestSpecInvalidationIsScopedToTheTargetURI pins RFC 9111 §4.4's scoping: a
// cache MUST invalidate the TARGET URI of the unsafe request — and that URI
// only — so a mutation of one resource can never discard another resource's
// stored entry. After the unrelated mutation, the untouched entry still
// revalidates with a conditional GET instead of refetching.
func TestSpecInvalidationIsScopedToTheTargetURI(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"kept"` {
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"kept"`}), ""), nil
		}

		if req.URL.Path == "/kept" {
			return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"kept"`}), "kept payload"), nil
		}

		return stubResponse(http.StatusNoContent, stubHeader(), ""), nil
	})

	transport := NewTransport(next, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/kept"))

	status, _, _ := fetch(t, transport, newSpecRequest(t, http.MethodDelete, "https://example.test/other"))
	if status != http.StatusNoContent {
		t.Fatalf("DELETE /other status = %d, want passthrough 204", status)
	}

	status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/kept"))
	if status != http.StatusOK || body != "kept payload" {
		t.Fatalf("revalidated /kept = %d %q, want the cached 200 payload", status, body)
	}

	if got := transport.Stats().Hits; got != 1 {
		t.Errorf("hits = %d, want 1 (the entry survived an unrelated mutation)", got)
	}
}

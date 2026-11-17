package etagclient

import (
	"net/http"
	"testing"
)

// TestSpecHeadErrorAndCacheMissesPassThrough pins the §4.3.5 preconditions:
// only a 200 with a stored entry under the same key can update or
// invalidate; error responses and uncached URIs pass through with the store
// untouched, and a HEAD response itself is never stored.
func TestSpecHeadErrorAndCacheMissesPassThrough(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusInternalServerError, header: stubHeader(), body: ""},
		{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"head-only"`}), body: ""},
		{status: http.StatusNotFound, header: stubHeader(), body: ""},
	}}

	transport := NewTransport(stub, Options{})
	url := "https://api.dev.test/articles"

	head := newSpecRequest(t, http.MethodHead, url)

	status, _, _ := fetch(t, transport, head)
	if status != http.StatusInternalServerError {
		t.Fatalf("first HEAD status = %d, want passthrough 500", status)
	}

	status, _, _ = fetch(t, transport, head)
	if status != http.StatusOK {
		t.Fatalf("second HEAD status = %d, want passthrough 200", status)
	}

	status, _, _ = fetch(t, transport, newGetRequest(t, url))
	if status != http.StatusNotFound {
		t.Fatalf("GET status = %d, want passthrough 404", status)
	}

	stats := transport.Stats()

	if stats.Stored != 0 || stats.Entries != 0 || stats.Hits != 0 {
		t.Errorf("stats = %+v, want all zero (HEAD responses are never stored)", stats)
	}
}

// TestSpecHeadTransportErrorPassesThrough pins that a transport failure under
// a HEAD never touches the cache: the error is the caller's to handle, and
// neither update nor invalidation may run on a response that never arrived.
func TestSpecHeadTransportErrorPassesThrough(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodHead {
			return nil, errCloseBoom
		}

		header := stubHeader(headerPair{"ETag", `"v1"`})

		return stubResponse(http.StatusOK, header, "payload"), nil
	})

	transport := NewTransport(next, Options{})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))

	head := newSpecRequest(t, http.MethodHead, url)

	// The failure belongs to the caller, so this round trip bypasses the
	// fetch helper, which treats transport errors as fatal.
	resp, headErr := transport.RoundTrip(head)
	if headErr == nil {
		_ = resp.Body.Close()

		t.Fatal("HEAD error = nil, want the transport failure passed through")
	}

	stats := transport.Stats()
	if stats.Entries != 1 || stats.Hits != 0 {
		t.Errorf("stats after failed HEAD = %+v, want the stored entry untouched", stats)
	}
}

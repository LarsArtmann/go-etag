package etagclient

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// TestSpecNoCacheResponseServesViaRevalidation pins the interplay with
// RFC 9111 §5.2.2.4: no-cache forbids reuse of a stored response "without
// successful validation". The transport only ever serves a stored body
// after the server answers 304, so a no-cache response remains fully
// usable: it is stored and revalidated on every use.
func TestSpecNoCacheResponseServesViaRevalidation(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"v"` {
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
		}

		header := stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"Cache-Control", "no-cache"},
		)

		return stubResponse(http.StatusOK, header, "payload"), nil
	})

	transport := NewTransport(next, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if status != http.StatusOK || body != "payload" {
		t.Fatalf("revalidated response = %d %q, want the cached 200 payload", status, body)
	}

	got := transport.Stats()
	want := Stats{Hits: 1, Stored: 1, Entries: 1}
	if got != want {
		t.Errorf("stats = %+v, want %+v (no-cache is storable; reuse just requires validation)", got, want)
	}
}

// TestSpecHeadIsNeverConditional pins the caller-facing side of the
// transport's HEAD handling: HEAD's headers-only semantics are the server's
// to answer, so a HEAD never receives the stored validator and is never
// rebuilt from cache. (Behind the scenes a matching HEAD 200 still freshens
// the stored metadata — RFC 9111 §4.3.5, pinned by TestSpecHeadFreshening.)
func TestSpecHeadIsNeverConditional(t *testing.T) {
	t.Parallel()

	stub := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("If-None-Match"); got != "" {
			t.Errorf("If-None-Match = %q on %s, want none (HEAD must not carry the stored validator)", got, req.Method)
		}

		return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
	})

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodHead, "https://example.test/data", nil)
	if err != nil {
		t.Fatalf("build HEAD request: %v", err)
	}

	status, _, _ := fetch(t, transport, req)

	if status != http.StatusOK {
		t.Errorf("HEAD status = %d, want passthrough 200", status)
	}

	if got := transport.Stats().Hits; got != 0 {
		t.Errorf("hits = %d, HEAD must never be rebuilt from cache", got)
	}
}

// TestSpecCallerSuppliedIfNoneMatchIsHonored pins the ownership rule behind
// conditional requests (RFC 9110 §13.1.2): a validator the caller already
// set is the caller's own conditional. The transport must not clobber it
// with its stored validator, and a 304 answering the caller's validator
// passes through for the caller to interpret, instead of being rebuilt from
// the transport's entry.
func TestSpecCallerSuppliedIfNoneMatchIsHonored(t *testing.T) {
	t.Parallel()

	var (
		callMutex sync.Mutex
		received  []string
	)

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callMutex.Lock()
		received = append(received, req.Header.Get("If-None-Match"))
		callMutex.Unlock()

		switch req.Header.Get("If-None-Match") {
		case `"other"`:
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"other"`}), ""), nil
		case `"v2"`:
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v2"`}), ""), nil
		case `"v1"`:
			header := stubHeader(headerPair{"ETag", `"v2"`}, headerPair{"Age", "0"})

			return stubResponse(http.StatusOK, header, "fresh payload"), nil
		default:
			return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v1"`}), "payload"), nil
		}
	})

	transport := NewTransport(next, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	callerConditional := newGetRequest(t, "https://example.test/data")
	callerConditional.Header.Set("If-None-Match", `"other"`)

	status, _, _ := fetch(t, transport, callerConditional)
	if status != http.StatusNotModified {
		t.Fatalf("caller conditional status = %d, want 304 passthrough (not rebuilt)", status)
	}

	if got := transport.Stats().Hits; got != 0 {
		t.Errorf("hits = %d, a caller-owned 304 is not a cache hit", got)
	}

	if got := callerConditional.Header.Get("If-None-Match"); got != `"other"` {
		t.Errorf("caller If-None-Match = %q, must not be clobbered", got)
	}

	_, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	if body != "fresh payload" {
		t.Fatalf("body = %q, want the fresh payload replacing the entry", body)
	}

	callMutex.Lock()
	seenValidators := append([]string(nil), received...)
	callMutex.Unlock()

	want := []string{"", `"other"`, `"v1"`}
	if strings.Join(seenValidators, "|") != strings.Join(want, "|") {
		t.Errorf("validators seen = %v, want %v", seenValidators, want)
	}
}

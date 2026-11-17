package etagclient

import (
	"net/http"
	"testing"
)

// TestSpecWeakFormOfStoredValidatorIsAdopted pins the comparison function of
// the RFC 9111 §4.3.4 validator filter: weak comparison (RFC 9110 §8.8.3.2).
// A 304 answering stored "v1" with the weak form W/"v1" names the same
// representation, so its claim is adopted and re-presented — only a
// mismatching opaque tag is filtered.
func TestSpecWeakFormOfStoredValidatorIsAdopted(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"v1"`}), body: "payload"},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `W/"v1"`})},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `W/"v1"`})},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := header.Get("ETag"); got != `W/"v1"` {
		t.Errorf("rebuilt ETag = %q, want the 304's weak form (weak comparison adopts it)", got)
	}

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := stub.lastValidator(); got != `W/"v1"` {
		t.Errorf("revalidation validator = %q, want the adopted weak form (RFC 9111 §4.3.4)", got)
	}
}

// TestSpecOnly200WithValidatorIsStored pins the storage gate the transport
// applies ahead of RFC 9111 §3: only a 200 carrying a validator is stored,
// because §4 reuse requires successful validation and §4.3 validation
// requires a validator. Other final statuses pass through untouched.
func TestSpecOnly200WithValidatorIsStored(t *testing.T) {
	t.Parallel()

	for _, spec := range []struct {
		name   string
		status int
	}{
		{name: "201 Created", status: http.StatusCreated},
		{name: "206 Partial Content", status: http.StatusPartialContent},
		{name: "304 Not Modified", status: http.StatusNotModified},
		{name: "500 Internal Server Error", status: http.StatusInternalServerError},
	} {
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()

			next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("If-None-Match"); got != "" {
					t.Errorf("If-None-Match = %q on second request, want none (nothing stored)", got)
				}

				return stubResponse(spec.status, stubHeader(headerPair{"ETag", `"v"`}), "body"), nil
			})

			transport := NewTransport(next, Options{})

			for range 2 {
				status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
				if status != spec.status || body != "body" {
					t.Fatalf("response = %d %q, want passthrough %d", status, body, spec.status)
				}
			}

			if got := transport.Stats(); got.Stored != 0 || got.Entries != 0 {
				t.Errorf("stats = %+v, a %d response must never be stored", got, spec.status)
			}
		})
	}
}

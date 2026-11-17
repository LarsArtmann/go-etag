package etagclient

import (
	"net/http"
	"testing"
)

// TestSpecFresheningSkipsExceptedFields pins the exceptions of the RFC 9111
// update rules: connection-scope fields are excepted from storage (§3.1),
// Content-Length and Content-Range from updates (§3.2), and Content-Encoding
// when net/http transparently decoded the stored body, because the §3.2
// integrity allowance for caches storing processed representations applies:
// the stored bytes are the decoded ones, so the 304's encoding claim would
// describe bytes the rebuilt response does not carry.
func TestSpecFresheningSkipsExceptedFields(t *testing.T) {
	t.Parallel()

	t.Run("hop-by-hop and length metadata never replace stored fields", func(t *testing.T) {
		t.Parallel()

		stored := stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"Date", "stale-date"},
			headerPair{"Content-Type", "text/plain"},
		)

		notModified := stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"Date", "fresh-date"},
			headerPair{"Content-Length", "999"},
			headerPair{"Content-Range", "bytes 0-4/9"},
			headerPair{"Connection", "close"},
			headerPair{"Keep-Alive", "timeout=5"},
		)

		stub := &recordedStub{steps: []stubStep{
			{status: http.StatusOK, header: stored, body: "payload"},
			{status: http.StatusNotModified, header: notModified},
		}}

		transport := NewTransport(stub, Options{})

		fetch(t, transport, newGetRequest(t, "https://example.test/data"))

		resp, err := transport.RoundTrip(newGetRequest(t, "https://example.test/data"))
		if err != nil {
			t.Fatalf("revalidation: %v", err)
		}

		defer func() { _ = resp.Body.Close() }()

		if got := resp.Header.Get("Date"); got != "fresh-date" {
			t.Errorf("Date = %q, want fresh-date (freshening must still run)", got)
		}

		for _, name := range []string{"Connection", "Keep-Alive", "Content-Range"} {
			if got := resp.Header.Get(name); got != "" {
				t.Errorf("%s = %q, want empty (RFC 9111 §3.1/§3.2 excepted from update)", name, got)
			}
		}

		if got := resp.Header.Get("Content-Length"); got == "999" {
			t.Error("Content-Length = 999, the 304's claim must not replace the stored metadata")
		}

		if resp.ContentLength != int64(len("payload")) {
			t.Errorf("ContentLength = %d, want the cached body length", resp.ContentLength)
		}
	})

	t.Run("content-encoding is not claimed for a decoded body", func(t *testing.T) {
		t.Parallel()

		stub := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("If-None-Match") != "" {
				notModified := stubHeader(
					headerPair{"ETag", `"v"`},
					headerPair{"Content-Encoding", "gzip"},
				)

				return stubResponse(http.StatusNotModified, notModified, ""), nil
			}

			// net/http transparently decompresses such responses and strips
			// Content-Encoding; resp.Uncompressed records that the stored
			// bytes are the decoded form.
			resp := stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), "decoded body")
			resp.Uncompressed = true

			return resp, nil
		})

		transport := NewTransport(stub, Options{})

		fetch(t, transport, newGetRequest(t, "https://example.test/data"))

		resp, err := transport.RoundTrip(newGetRequest(t, "https://example.test/data"))
		if err != nil {
			t.Fatalf("revalidation: %v", err)
		}

		defer func() { _ = resp.Body.Close() }()

		if got := resp.Header.Get("Content-Encoding"); got != "" {
			t.Errorf("Content-Encoding = %q, want empty (RFC 9111 §3.2: the stored body is decoded)", got)
		}

		if !resp.Uncompressed {
			t.Error("Uncompressed = false, want true (the rebuilt body is the stored decoded form)")
		}
	})

	t.Run("content-encoding replaces for a verbatim stored body", func(t *testing.T) {
		t.Parallel()

		stored := stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"Content-Encoding", "br"},
		)

		notModified := stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"Content-Encoding", "gzip"},
		)

		stub := &recordedStub{steps: []stubStep{
			{status: http.StatusOK, header: stored, body: "encoded body"},
			{status: http.StatusNotModified, header: notModified},
		}}

		transport := NewTransport(stub, Options{})

		fetch(t, transport, newGetRequest(t, "https://example.test/data"))
		_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

		if got := header.Get("Content-Encoding"); got != "gzip" {
			t.Errorf("Content-Encoding = %q, want gzip from the 304 (stored bytes are verbatim)", got)
		}
	})
}

// TestSpecMismatched304ValidatorIsNotAdopted pins the validator filtering
// of RFC 9111 §4.3.4: a 304 declaring a validator that does not weak-match
// the one it just validated names a different representation, so neither
// the rebuilt response nor the store adopts the claim — the next
// revalidation presents the validator the server actually answered.
func TestSpecMismatched304ValidatorIsNotAdopted(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"v1"`}), body: "payload"},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `"v2"`})},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `"v2"`})},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := header.Get("ETag"); got != `"v1"` {
		t.Errorf("rebuilt ETag = %q, want the stored v1 (mismatched claims are not adopted)", got)
	}

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := stub.lastValidator(); got != `"v1"` {
		t.Errorf("revalidation validator = %q, want the stored v1 (RFC 9111 §4.3.4 filtering)", got)
	}
}

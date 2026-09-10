package etagclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestIntegrationRealServerRoundTrip runs the transport against a real
// net/http server and client, exercising the canonical header forms, the
// bodiless 304, and Age freshening exactly as a production stack produces
// them. The stubbed spec tests verify spec decisions; this one verifies the
// wire reality those decisions ride on.
func TestIntegrationRealServerRoundTrip(t *testing.T) {
	t.Parallel()

	const (
		validator      = `"real-v1"`
		firstAge       = "137882"
		revalidatedAge = "138100"
		payload        = "real payload"
	)

	var revalidations atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == validator {
			revalidations.Add(1)

			w.Header().Set("ETag", validator)
			w.Header().Set("Age", revalidatedAge)
			w.WriteHeader(http.StatusNotModified)

			return
		}

		w.Header().Set("ETag", validator)
		w.Header().Set("Age", firstAge)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: NewTransport(http.DefaultTransport, Options{})}

	read := func() (int, http.Header, string) {
		t.Helper()

		resp, err := client.Get(server.URL + "/articles")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}

		defer func() { _ = resp.Body.Close() }()

		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("read body: %v", readErr)
		}

		return resp.StatusCode, resp.Header, string(data)
	}

	status, header, body := read()
	if status != http.StatusOK || body != payload {
		t.Fatalf("first response = %d %q, want 200 %q", status, body, payload)
	}

	if got := header.Get("Age"); got != firstAge {
		t.Errorf("Age = %q, want %q surfaced on the network 200 (RFC 9111 §5.1)", got, firstAge)
	}

	status, header, body = read()
	if status != http.StatusOK || body != payload {
		t.Fatalf("rebuilt response = %d %q, want the cached 200 %q", status, body, payload)
	}

	if got := header.Get("Age"); got != revalidatedAge {
		t.Errorf("Age = %q, want the 304's %q freshened onto the rebuild (RFC 9111 §4.3.4)", got, revalidatedAge)
	}

	if got := header.Get("ETag"); got != validator {
		t.Errorf("Etag = %q, want %q through real canonicalization", got, validator)
	}

	if got := revalidations.Load(); got != 1 {
		t.Errorf("revalidations = %d, want exactly one conditional round trip", got)
	}
}

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

// TestIntegrationUnsafeMethodInvalidatesThroughRealServer runs the RFC 9111
// §4.4 invalidation through a real net/http server and client: a 204 answer
// to PUT must invalidate the stored GET entry, so the next GET refetches
// unconditionally and receives the mutated representation instead of
// revalidating the pre-mutation body.
func TestIntegrationUnsafeMethodInvalidatesThroughRealServer(t *testing.T) {
	t.Parallel()

	var (
		mutated      atomic.Bool
		conditionals atomic.Int64
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			mutated.Store(true)

			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			if r.Header.Get("If-None-Match") != "" {
				conditionals.Add(1)
			}

			if mutated.Load() {
				w.Header().Set("ETag", `"mutated-v2"`)
				_, _ = w.Write([]byte("mutated payload"))

				return
			}

			w.Header().Set("ETag", `"v1"`)
			_, _ = w.Write([]byte("original payload"))
		}
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: NewTransport(http.DefaultTransport, Options{})}

	read := func() (int, string) {
		t.Helper()

		resp, err := client.Get(server.URL + "/thing")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}

		defer func() { _ = resp.Body.Close() }()

		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("read body: %v", readErr)
		}

		return resp.StatusCode, string(data)
	}

	status, body := read()
	if status != http.StatusOK || body != "original payload" {
		t.Fatalf("first GET = %d %q, want the original payload", status, body)
	}

	put, err := http.NewRequestWithContext(t.Context(), http.MethodPut, server.URL+"/thing", nil)
	if err != nil {
		t.Fatalf("build PUT: %v", err)
	}

	resp, err := client.Do(put)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}

	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 204", resp.StatusCode)
	}

	status, body = read()
	if status != http.StatusOK || body != "mutated payload" {
		t.Fatalf("post-PUT GET = %d %q, want the mutated payload (RFC 9111 §4.4)", status, body)
	}

	if got := conditionals.Load(); got != 0 {
		t.Errorf("conditional GETs = %d, want 0 (the entry was invalidated, not revalidated)", got)
	}
}

// TestIntegrationHeadFreshensStoredEntryThroughRealServer runs RFC 9111
// §4.3.5 through a real net/http server and client: a HEAD 200 with the
// stored validator (and matching Content-Length) must freshen the stored
// metadata, and the next conditional GET must rebuild wearing the HEAD's
// fields — with Go's canonical "Etag" key on both sides of the wire.
func TestIntegrationHeadFreshensStoredEntryThroughRealServer(t *testing.T) {
	t.Parallel()

	const (
		validator = `"head-v1"`
		payload   = "head payload"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", validator)

		if r.Method == http.MethodHead {
			w.Header().Set("X-Ratelimit-Remaining", "41")
			w.WriteHeader(http.StatusOK)

			return
		}

		if r.Header.Get("If-None-Match") == validator {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(server.Close)

	client := &http.Client{Transport: NewTransport(http.DefaultTransport, Options{})}

	first, err := client.Get(server.URL + "/articles")
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}

	_, _ = io.Copy(io.Discard, first.Body)
	_ = first.Body.Close()

	if first.StatusCode != http.StatusOK {
		t.Fatalf("first GET status = %d, want the stored 200", first.StatusCode)
	}

	head, err := http.NewRequestWithContext(t.Context(), http.MethodHead, server.URL+"/articles", nil)
	if err != nil {
		t.Fatalf("build HEAD: %v", err)
	}

	headResp, err := client.Do(head)
	if err != nil {
		t.Fatalf("HEAD: %v", err)
	}

	_ = headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", headResp.StatusCode)
	}

	getResp, err := client.Get(server.URL + "/articles")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}

	defer func() { _ = getResp.Body.Close() }()

	data, readErr := io.ReadAll(getResp.Body)
	if readErr != nil {
		t.Fatalf("read body: %v", readErr)
	}

	if getResp.StatusCode != http.StatusOK || string(data) != payload {
		t.Fatalf("rebuilt GET = %d %q, want the stored 200 %q", getResp.StatusCode, string(data), payload)
	}

	if got := getResp.Header.Get("X-Ratelimit-Remaining"); got != "41" {
		t.Errorf("X-Ratelimit-Remaining = %q, want the HEAD-freshened 41 (RFC 9111 §4.3.5)", got)
	}

	if got := getResp.Header.Get("ETag"); got != validator {
		t.Errorf("Etag = %q, want %q through real canonicalization", got, validator)
	}
}

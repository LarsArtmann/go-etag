package etagclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestRoundTripEvictsOldestEntry(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		header := stubHeader(headerPair{"ETag", `"tag-` + req.URL.Path + `"`})

		return stubResponse(http.StatusOK, header, "body of "+req.URL.Path), nil
	})

	transport := NewTransport(next, Options{MaxEntries: 1})

	fetch(t, transport, newGetRequest(t, "https://example.test/first"))
	fetch(t, transport, newGetRequest(t, "https://example.test/second"))

	// "first" was evicted: its revalidation carries no If-None-Match, so the
	// stub answers with a fresh 200 instead of a 304.
	status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/first"))
	if status != http.StatusOK || body != "body of /first" {
		t.Errorf("evicted entry re-request = %d %q, want fresh 200", status, body)
	}

	if got := transport.Stats().Hits; got != 0 {
		t.Errorf("hits = %d, want 0 (no entry was cached for revalidation)", got)
	}
}

func TestRoundTripSkipsNonGET(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		header := stubHeader(headerPair{"ETag", `"should-not-cache"`})

		return stubResponse(http.StatusCreated, header, "created"), nil
	})

	transport := NewTransport(next, Options{})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.test/issues", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201 passthrough", resp.StatusCode)
	}

	if got := transport.Stats(); got.Stored != 0 || got.Entries != 0 {
		t.Errorf("stats = %+v, POST must not populate the cache", got)
	}
}

func TestRoundTripMaxBodyBytes(t *testing.T) {
	t.Parallel()

	exact := strings.Repeat("x", 8)
	oversized := strings.Repeat("y", 9)

	t.Run("body at the limit is cached", func(t *testing.T) {
		t.Parallel()

		next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("If-None-Match") == `"v"` {
				return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
			}

			return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), exact), nil
		})

		transport := NewTransport(next, Options{MaxBodyBytes: len(exact)})

		_, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
		if body != exact {
			t.Fatalf("first body = %q, want %d bytes", body, len(exact))
		}

		_, _, body = fetch(t, transport, newGetRequest(t, "https://example.test/data"))
		if body != exact {
			t.Fatalf("rebuilt body = %q, want cached %d bytes", body, len(exact))
		}

		if got := transport.Stats().Hits; got != 1 {
			t.Errorf("hits = %d, want 1 (boundary body must be cached)", got)
		}
	})

	t.Run("oversized body passes through uncached and intact", func(t *testing.T) {
		t.Parallel()

		next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("If-None-Match") != "" {
				t.Errorf(
					"oversized responses must not be revalidated, got If-None-Match %q",
					req.Header.Get("If-None-Match"),
				)
			}

			return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), oversized), nil
		})

		transport := NewTransport(next, Options{MaxBodyBytes: len(exact)})

		for range 2 {
			_, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
			if body != oversized {
				t.Fatalf("body = %d bytes, want the full %d-byte passthrough body", len(body), len(oversized))
			}
		}

		if got := transport.Stats(); got.Stored != 0 || got.Entries != 0 {
			t.Errorf("stats = %+v, oversized responses must not be cached", got)
		}
	})
}

func TestRoundTrip304WithoutCachedEntryPassesThrough(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
	})

	transport := NewTransport(next, Options{})

	status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	if status != http.StatusNotModified || body != "" {
		t.Fatalf("response = %d %q, want 304 passthrough", status, body)
	}

	if got := transport.Stats().Hits; got != 0 {
		t.Errorf("hits = %d, an uncached 304 is not a hit", got)
	}
}

func TestRoundTripStoreFailurePassesThrough(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		header := stubHeader(headerPair{"ETag", `"v"`})

		return &http.Response{
			Status:        "200 OK",
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        header,
			Body:          &failingBody{data: []byte("partial")},
			ContentLength: -1,
			Request:       nil,
		}, nil
	})

	transport := NewTransport(next, Options{})

	resp, err := transport.RoundTrip(newGetRequest(t, "https://example.test/data"))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	data, readErr := io.ReadAll(resp.Body)

	if string(data) != "partial" {
		t.Errorf("body = %q, want the buffered prefix", data)
	}

	if !errors.Is(readErr, errBodyBoom) {
		t.Errorf("read error = %v, want the original body error to replay", readErr)
	}

	if got := transport.Stats(); got.Stored != 0 {
		t.Errorf("stored = %d, a failing body must not be cached", got.Stored)
	}
}

func TestRoundTripConcurrent(t *testing.T) {
	t.Parallel()

	const (
		goroutines   = 8
		perGoroutine = 25
	)

	next, _ := conditionalStub(t, `"v"`, "shared-body")

	transport := NewTransport(next, Options{})

	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			for range perGoroutine {
				req := newGetRequest(t, "https://example.test/data")

				resp, err := transport.RoundTrip(req)
				if err != nil {
					t.Errorf("GET: %v", err)

					return
				}

				data, readErr := io.ReadAll(resp.Body)
				_ = resp.Body.Close()

				if readErr != nil {
					t.Errorf("read body: %v", readErr)

					return
				}

				if string(data) != "shared-body" {
					t.Errorf("body = %q, want shared-body", data)

					return
				}
			}
		})
	}

	wg.Wait()

	got := transport.Stats()

	if got.Entries != 1 {
		t.Errorf("entries = %d, want 1", got.Entries)
	}

	if total := int64(goroutines * perGoroutine); got.Hits+got.Stored != total {
		t.Errorf("hits+stored = %d, want %d (every request is a store or a hit)", got.Hits+got.Stored, total)
	}
}

// TestStoreCloseFailureKeepsResponseReadable pins the store path when the
// original body refuses to close: the caller still receives a fully
// re-readable body with accurate Content-Length, and the unreadable-to-close
// response is never stored.
func TestStoreCloseFailureKeepsResponseReadable(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:        "200 OK",
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        stubHeader(headerPair{"ETag", `"v"`}),
			Body:          &closeFailingBody{data: []byte("payload")},
			ContentLength: -1,
			Request:       nil,
		}, nil
	})

	transport := NewTransport(next, Options{})

	resp, err := transport.RoundTrip(newGetRequest(t, "https://example.test/data"))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read body: %v", readErr)
	}

	if string(data) != "payload" {
		t.Errorf("body = %q, want the full payload", data)
	}

	if resp.ContentLength != int64(len("payload")) {
		t.Errorf("ContentLength = %d, want %d", resp.ContentLength, len("payload"))
	}

	if got := transport.Stats(); got.Stored != 0 || got.Entries != 0 {
		t.Errorf("stats = %+v, want nothing stored when Close fails", got)
	}
}

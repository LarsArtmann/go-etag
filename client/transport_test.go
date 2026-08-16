package etagclient

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// errBodyBoom is the sentinel a failingBody replays once its data is
// exhausted.
var errBodyBoom = errors.New("body boom")

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// failingBody yields its data once, then fails with errBodyBoom forever.
type failingBody struct {
	data []byte
	read int
}

func (b *failingBody) Read(p []byte) (int, error) {
	if b.read >= len(b.data) {
		return 0, errBodyBoom
	}

	n := copy(p, b.data[b.read:])
	b.read += n

	return n, nil
}

func (b *failingBody) Close() error {
	return nil
}

// headerPair is one name/value entry handed to stubHeader.
type headerPair struct {
	name  string
	value string
}

// stubHeader builds a header map through Set, so keys land in the canonical
// form a real transport would use; map literals bypass canonicalization.
func stubHeader(pairs ...headerPair) http.Header {
	header := http.Header{}

	for _, pair := range pairs {
		header.Set(pair.name, pair.value)
	}

	return header
}

// stubResponse builds a minimal HTTP/1.1 response with a string body.
func stubResponse(status int, header http.Header, body string) *http.Response {
	if header == nil {
		header = http.Header{}
	}

	return &http.Response{
		Status:        http.StatusText(status),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       nil,
	}
}

// newGetRequest builds a GET request bound to the test context.
func newGetRequest(t *testing.T, url string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	return req
}

// fetch runs a GET through the transport and returns status, header, and
// fully-read body, closing the response.
func fetch(t *testing.T, transport *Transport, req *http.Request) (int, http.Header, string) {
	t.Helper()

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("GET %s: %v", req.URL, err)
	}

	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body of %s: %v", req.URL, err)
	}

	return resp.StatusCode, resp.Header, string(data)
}

// conditionalStub simulates an origin server: the first GET for a validator
// returns 200 with body; revalidations carrying If-None-Match get 304 with
// fresh Date and rate-limit headers.
func conditionalStub(t *testing.T, etag, body string) (roundTripperFunc, func() int) {
	t.Helper()

	var (
		callMutex sync.Mutex
		calls     int
	)

	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		callMutex.Lock()
		calls++
		callMutex.Unlock()

		if req.Header.Get("If-None-Match") == etag {
			header := stubHeader(
				headerPair{"ETag", etag},
				headerPair{"Date", "fresh-date"},
				headerPair{"X-Ratelimit-Remaining", "42"},
			)

			return stubResponse(http.StatusNotModified, header, ""), nil
		}

		header := stubHeader(
			headerPair{"ETag", etag},
			headerPair{"Date", "stale-date"},
			headerPair{"Content-Type", "text/plain"},
			headerPair{"X-Ratelimit-Remaining", "5000"},
		)

		return stubResponse(http.StatusOK, header, body), nil
	})

	return transport, func() int {
		callMutex.Lock()
		defer callMutex.Unlock()

		return calls
	}
}

func TestRoundTripRebuilds304FromCache(t *testing.T) {
	t.Parallel()

	next, calls := conditionalStub(t, `"v1"`, "payload")

	transport := NewTransport(next, Options{FromCacheHeader: "X-From-Cache"})

	status, header, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	if status != http.StatusOK || body != "payload" {
		t.Fatalf("first response = %d %q, want 200 payload", status, body)
	}

	if header.Get("X-From-Cache") != "" {
		t.Error("network 200 must not carry the from-cache marker")
	}

	status, header, body = fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	if status != http.StatusOK || body != "payload" {
		t.Fatalf("rebuilt response = %d %q, want 200 with cached body", status, body)
	}

	if got := header.Get("X-From-Cache"); got != "1" {
		t.Errorf("from-cache marker = %q, want 1", got)
	}

	if got := header.Get("Date"); got != "fresh-date" {
		t.Errorf("Date = %q, want the 304's fresh value (default PreserveOn304)", got)
	}

	if got := header.Get("Content-Type"); got != "text/plain" {
		t.Errorf("Content-Type = %q, want the cached value", got)
	}

	if got := calls(); got != 2 {
		t.Errorf("network calls = %d, want 2 (one fresh, one revalidation)", got)
	}

	got := transport.Stats()
	want := Stats{Hits: 1, Stored: 1, Entries: 1}
	if got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

func TestRoundTripStoreStripsStaleMarker(t *testing.T) {
	t.Parallel()

	// The origin sets the marker itself; the rebuilt response must wear the
	// transport's own value, never the stored one.
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"v"` {
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
		}

		header := stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"X-From-Cache", "9"},
		)

		return stubResponse(http.StatusOK, header, "body"), nil
	})

	transport := NewTransport(next, Options{FromCacheHeader: "X-From-Cache"})

	fetch(t, transport, newGetRequest(t, "https://example.test/x"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/x"))

	if got := header.Get("X-From-Cache"); got != "1" {
		t.Errorf("from-cache marker = %q, want 1 (stored marker must be replaced)", got)
	}
}

func TestRoundTripSeparatesCredentialsViaKeyFunc(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := "body-for-" + strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")

		header := stubHeader(headerPair{"ETag", `"same"`})

		return stubResponse(http.StatusOK, header, body), nil
	})

	transport := NewTransport(next, Options{
		KeyFunc: func(req *http.Request) string {
			return req.Header.Get("Authorization") + "|" + req.URL.String()
		},
	})

	get := func(token string) string {
		req := newGetRequest(t, "https://example.test/things")
		req.Header.Set("Authorization", "Bearer "+token)

		_, _, body := fetch(t, transport, req)

		return body
	}

	if got := get("token-a"); got != "body-for-token-a" {
		t.Fatalf("first token-a body = %q", got)
	}

	if got := get("token-b"); got != "body-for-token-b" {
		t.Errorf("token-b body = %q; cached cross-credential replay detected", got)
	}

	if got := get("token-a"); got != "body-for-token-a" {
		t.Errorf("second token-a body = %q, want cached body-for-token-a", got)
	}
}

func TestRoundTripDefaultKeyIgnoresCredentials(t *testing.T) {
	t.Parallel()

	// Documents the sharp edge that motivates Options.KeyFunc: with the
	// default URL-only key, a second credential revalidates the first
	// credential's entry and receives its cached body.
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"v"` {
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
		}

		header := stubHeader(headerPair{"ETag", `"v"`})

		return stubResponse(http.StatusOK, header, "body-for-a"), nil
	})

	transport := NewTransport(next, Options{})

	first := newGetRequest(t, "https://example.test/things")
	first.Header.Set("Authorization", "Bearer token-a")
	fetch(t, transport, first)

	second := newGetRequest(t, "https://example.test/things")
	second.Header.Set("Authorization", "Bearer token-b")
	_, _, body := fetch(t, transport, second)

	if body != "body-for-a" {
		t.Fatalf("body = %q, want the first credential's cached body (default key ignores auth)", body)
	}
}

func TestRoundTripPreserveOn304(t *testing.T) {
	t.Parallel()

	t.Run("custom headers replace cached values", func(t *testing.T) {
		t.Parallel()

		next, _ := conditionalStub(t, `"v"`, "body")

		transport := NewTransport(next, Options{
			PreserveOn304: []string{"Date", "X-Ratelimit-Remaining", "Retry-After"},
		})

		fetch(t, transport, newGetRequest(t, "https://example.test/rate"))
		_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/rate"))

		if got := header.Get("X-Ratelimit-Remaining"); got != "42" {
			t.Errorf("X-Ratelimit-Remaining = %q, want 42 from the 304", got)
		}

		if got := header.Get("Date"); got != "fresh-date" {
			t.Errorf("Date = %q, want fresh-date from the 304", got)
		}
	})

	t.Run("empty slice disables merging", func(t *testing.T) {
		t.Parallel()

		next, _ := conditionalStub(t, `"v"`, "body")

		transport := NewTransport(next, Options{PreserveOn304: []string{}})

		fetch(t, transport, newGetRequest(t, "https://example.test/rate"))
		_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/rate"))

		if got := header.Get("Date"); got != "stale-date" {
			t.Errorf("Date = %q, want the cached value when merging is disabled", got)
		}
	})
}

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

func TestRoundTripDoesNotMutateRequest(t *testing.T) {
	t.Parallel()

	next, _ := conditionalStub(t, `"v"`, "body")

	transport := NewTransport(next, Options{})

	req := newGetRequest(t, "https://example.test/data")

	fetch(t, transport, req)

	if got := req.Header.Get("If-None-Match"); got != "" {
		t.Errorf("If-None-Match = %q on the caller's request, RoundTrip must not mutate it", got)
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

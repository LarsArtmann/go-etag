package etagclient

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
		t.Errorf("Date = %q, want the 304's fresh value (default FreshenOn304)", got)
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

func TestRoundTripDerivesCacheKeyLazily(t *testing.T) {
	t.Parallel()

	// The cache key is derived only when a method can use it: GET needs it
	// for the lookup up front, while a non-GET needs it only when its
	// response actually invalidates (RFC 9111 §4.4). Passthrough round trips
	// must never pay for key derivation.
	var keyCalls int

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			header := stubHeader(headerPair{"ETag", `"v"`})

			return stubResponse(http.StatusOK, header, "body"), nil
		}

		return stubResponse(http.StatusInternalServerError, nil, ""), nil
	})

	transport := NewTransport(next, Options{
		KeyFunc: func(req *http.Request) string {
			keyCalls++

			return req.URL.String()
		},
	})

	post := newSpecRequest(t, http.MethodPost, "https://example.test/things")
	fetch(t, transport, post)

	if keyCalls != 0 {
		t.Fatalf("KeyFunc called %d times for a non-invalidating POST, want 0", keyCalls)
	}

	get := newGetRequest(t, "https://example.test/things")
	fetch(t, transport, get)

	if keyCalls != 1 {
		t.Fatalf("KeyFunc called %d times for one GET, want 1", keyCalls)
	}

	next = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return stubResponse(http.StatusNoContent, nil, ""), nil
	})

	transport = NewTransport(next, Options{
		KeyFunc: func(req *http.Request) string {
			keyCalls++

			return req.URL.String()
		},
	})

	fetch(t, transport, post)

	if keyCalls != 2 {
		t.Fatalf("KeyFunc called %d times total, want 2 (one invalidating POST)", keyCalls)
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

func TestRoundTripFreshenOn304(t *testing.T) {
	t.Parallel()

	t.Run("named fields replace cached values", func(t *testing.T) {
		t.Parallel()

		next, _ := conditionalStub(t, `"v"`, "body")

		transport := NewTransport(next, Options{
			FreshenOn304: FreshenFields("Date", "X-Ratelimit-Remaining", "Retry-After"),
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

	t.Run("FreshenNone disables freshening", func(t *testing.T) {
		t.Parallel()

		next, _ := conditionalStub(t, `"v"`, "body")

		transport := NewTransport(next, Options{FreshenOn304: FreshenNone()})

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

func (b *closeFailingBody) Read(p []byte) (int, error) {
	if b.read >= len(b.data) {
		return 0, io.EOF
	}

	n := copy(p, b.data[b.read:])
	b.read += n

	return n, nil
}

func (b *closeFailingBody) Close() error {
	return errCloseBoom
}

// TestRoundTripNilNilFromNextPassesThrough pins the guard for a broken next
// RoundTripper that violates its contract by returning (nil, nil): the
// transport forwards the pair untouched instead of dereferencing a nil
// response.
func TestRoundTripNilNilFromNextPassesThrough(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, nil //nolint:nilnil // the contract-violating pair is the subject under test
	})

	transport := NewTransport(next, Options{})

	resp, err := transport.RoundTrip(newGetRequest(t, "https://example.test/data"))
	if err != nil {
		t.Fatalf("RoundTrip error = %v, want nil", err)
	}

	if resp != nil {
		_ = resp.Body.Close()

		t.Errorf("RoundTrip = %v, want nil, nil passed through", resp)
	}
}

package etagclient

import (
	"net/http"
	"testing"
)

// TestSpecNoStoreResponseIsNeverCached pins RFC 9111 §3: "A cache MUST NOT
// store a response ... [unless] the no-store cache directive is not present
// in the response", per the directive's own definition in §5.2.2.5. Bodies
// still stream through intact; only storage is forbidden.
func TestSpecNoStoreResponseIsNeverCached(t *testing.T) {
	t.Parallel()

	t.Run("no-store passes through uncached", func(t *testing.T) {
		t.Parallel()

		stub := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("If-None-Match"); got != "" {
				t.Errorf("If-None-Match = %q, want none (nothing was stored)", got)
			}

			header := stubHeader(
				headerPair{"ETag", `"v"`},
				headerPair{"Cache-Control", "no-store"},
			)

			return stubResponse(http.StatusOK, header, "secret payload"), nil
		})

		transport := NewTransport(stub, Options{})

		for range 2 {
			_, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/private"))
			if body != "secret payload" {
				t.Fatalf("body = %q, want the full passthrough body", body)
			}
		}

		if got := transport.Stats(); got.Stored != 0 || got.Entries != 0 {
			t.Errorf("stats = %+v, a no-store response must never be cached (RFC 9111 §3)", got)
		}
	})

	t.Run("no-store inside a quoted argument does not match", func(t *testing.T) {
		t.Parallel()

		// The escaped quote and the comma sit inside the quoted argument, so
		// the value is one private directive with a multi-name argument, not
		// a no-store directive; the response stays cacheable.
		next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("If-None-Match") == `"v"` {
				return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
			}

			header := stubHeader(
				headerPair{"ETag", `"v"`},
				headerPair{"Cache-Control", "max-age=60, private=\"a\\\"b, no-store\""},
			)

			return stubResponse(http.StatusOK, header, "payload"), nil
		})

		transport := NewTransport(next, Options{})

		fetch(t, transport, newGetRequest(t, "https://example.test/data"))
		_, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

		if body != "payload" {
			t.Fatalf("body = %q, want the cached payload", body)
		}

		if got := transport.Stats().Stored; got != 1 {
			t.Errorf("stored = %d, want 1 (a quoted argument is not a directive)", got)
		}
	})

	t.Run("directive matching is case-insensitive", func(t *testing.T) {
		t.Parallel()

		next := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			header := stubHeader(
				headerPair{"ETag", `"v"`},
				headerPair{"Cache-Control", "NO-STORE"},
			)

			return stubResponse(http.StatusOK, header, "payload"), nil
		})

		transport := NewTransport(next, Options{})

		fetch(t, transport, newGetRequest(t, "https://example.test/data"))

		if got := transport.Stats().Stored; got != 0 {
			t.Errorf("stored = %d, want 0 (directive names are case-insensitive)", got)
		}
	})
}

// TestSpecFresh200ReplacesStaleEntry pins the field case's resolution: when
// the edge finally refreshes, the 200 arrives with a new validator and a
// small Age; it replaces the stale entry, and later revalidations present
// the new validator and rebuild the new body.
func TestSpecFresh200ReplacesStaleEntry(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"ETag", `"stale-v1"`},
				headerPair{"Age", "137882"},
			),
			body: "stale payload",
		},
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"ETag", `"fresh-v2"`},
				headerPair{"Age", "0"},
			),
			body: "fresh payload",
		},
		{
			status: http.StatusNotModified,
			header: stubHeader(
				headerPair{"ETag", `"fresh-v2"`},
				headerPair{"Age", "3"},
			),
		},
	}}

	transport := NewTransport(stub, Options{})

	_, _, body := fetch(t, transport, newGetRequest(t, "https://api.dev.test/articles"))
	if body != "stale payload" {
		t.Fatalf("first body = %q, want stale payload", body)
	}

	_, header, body := fetch(t, transport, newGetRequest(t, "https://api.dev.test/articles"))
	if body != "fresh payload" {
		t.Fatalf("second body = %q, want the fresh replacement payload", body)
	}

	if got := header.Get("Age"); got != "0" {
		t.Errorf("Age = %q, want 0 from the fresh 200", got)
	}

	if got := stub.lastValidator(); got != `"stale-v1"` {
		t.Fatalf("revalidation validator = %q, want the stale entry's (that is what was stored)", got)
	}

	_, header, _ = fetch(t, transport, newGetRequest(t, "https://api.dev.test/articles"))

	if got := stub.lastValidator(); got != `"fresh-v2"` {
		t.Errorf("revalidation validator = %q, want fresh-v2 (the 200 replaced the entry)", got)
	}

	if got := header.Get("Age"); got != "3" {
		t.Errorf("Age = %q, want 3 from the 304 (RFC 9111 §4.3.4 freshening)", got)
	}

	got := transport.Stats()
	want := Stats{Hits: 1, Stored: 2, Entries: 1}
	if got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

// TestSpecStoredHeaderShedsHopByHopFields pins RFC 9111 §3.1: fields whose
// semantics require removal before forwarding, and the Proxy-* family a
// cache MUST NOT store, never enter the stored entry, so no later rebuild
// can resurrect connection-scope state from a previous response.
func TestSpecStoredHeaderShedsHopByHopFields(t *testing.T) {
	t.Parallel()

	stored := stubHeader(
		headerPair{"ETag", `"v"`},
		headerPair{"Connection", "close"},
		headerPair{"X-Edge-Hint", "drop-me"},
		headerPair{"Keep-Alive", "timeout=5"},
		headerPair{"Proxy-Authenticate", `Basic realm="edge"`},
	)

	// Connection also lists X-Edge-Hint by name, making it hop-by-hop for
	// this response even though the fixed set would keep it.
	stored.Set("Connection", "X-Edge-Hint")

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stored, body: "payload"},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `"v"`})},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	for _, name := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "X-Edge-Hint"} {
		if got := header.Get(name); got != "" {
			t.Errorf("%s = %q, want empty (RFC 9111 §3.1 forbids storing it)", name, got)
		}
	}
}

// TestSpec200WithoutValidatorIsNeverStored pins RFC 9111 §4 via §4.3: reuse
// requires successful validation, and validation requires a validator, so a
// 200 without an ETag can never legally be served from cache. The transport
// stores only what it can revalidate; the validator-less response keeps
// passing through untouched.
func TestSpec200WithoutValidatorIsNeverStored(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("If-None-Match"); got != "" {
			t.Errorf("If-None-Match = %q, want none (nothing was stored)", got)
		}

		return stubResponse(
			http.StatusOK,
			stubHeader(headerPair{"Content-Type", "text/plain"}),
			"validator-less payload",
		), nil
	})

	transport := NewTransport(next, Options{})

	for range 2 {
		status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))
		if status != http.StatusOK || body != "validator-less payload" {
			t.Fatalf("response = %d %q, want the passthrough 200 body", status, body)
		}
	}

	if got := transport.Stats(); got.Stored != 0 || got.Entries != 0 {
		t.Errorf("stats = %+v, a 200 without a validator must never be stored", got)
	}
}

// TestSpecRequestNoStoreBypassesTheCache pins RFC 9111 §5.2.2.5 (request
// directive): "A cache MUST NOT store any part of either this request or any
// response to it." A GET carrying Cache-Control: no-store must neither
// consult nor populate the store — no validator injection, no rebuild, no
// storage — and a HEAD carrying it must not freshen stored metadata either.
// Deleting on an unsafe method is still permitted: invalidation stores
// nothing.
func TestSpecRequestNoStoreBypassesTheCache(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{
			status: http.StatusOK,
			header: stubHeader(headerPair{"ETag", `"v1"`}, headerPair{"Date", "date-one"}),
			body:   "payload",
		},
		{
			status: http.StatusOK,
			header: stubHeader(headerPair{"ETag", `"v2"`}, headerPair{"Date", "date-two"}),
			body:   "fresh payload",
		},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `"v1"`}), body: ""},
		{
			status: http.StatusOK,
			header: stubHeader(headerPair{"ETag", `"v9"`}, headerPair{"Date", "date-nine"}),
			body:   "",
		},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `"v1"`}), body: ""},
	}}

	transport := NewTransport(stub, Options{})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))

	noStoreGet := newGetRequest(t, url)
	noStoreGet.Header.Set("Cache-Control", "no-store")

	status, header, body := fetch(t, transport, noStoreGet)
	if status != http.StatusOK || body != "fresh payload" || header.Get("ETag") != `"v2"` {
		t.Fatalf(
			"no-store GET = %d %q ETag %q, want the network answer passed through",
			status,
			body,
			header.Get("ETag"),
		)
	}

	if got := stub.lastValidator(); got != "" {
		t.Errorf("no-store GET validator = %q, want none (the cache must not be consulted)", got)
	}

	stats := transport.Stats()
	if stats.Stored != 1 || stats.Entries != 1 || stats.Hits != 0 {
		t.Fatalf("stats after no-store GET = %+v, want only the original entry stored", stats)
	}

	status, _, body = fetch(t, transport, newGetRequest(t, url))
	if status != http.StatusOK || body != "payload" {
		t.Fatalf("next GET = %d %q, want the original entry rebuilt untouched", status, body)
	}

	noStoreHead := newSpecRequest(t, http.MethodHead, url)
	noStoreHead.Header.Set("Cache-Control", "no-store")

	fetch(t, transport, noStoreHead)

	_, header, _ = fetch(t, transport, newGetRequest(t, url))

	if got := stub.lastValidator(); got != `"v1"` {
		t.Errorf("final GET validator = %q, want the original v1 (no-store HEAD must not touch the store)", got)
	}

	if got := header.Get("Date"); got != "date-one" {
		t.Errorf("Date = %q, want the stored date-one (no-store HEAD must not freshen)", got)
	}
}

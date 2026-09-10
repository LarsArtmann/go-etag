package etagclient

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// Tests in this file pin the transport to HTTP caching spec requirements
// (RFC 9111, the successor of RFC 7234) rather than to stub conveniences.
// They were motivated by a field report: an API behind a CDN answered 200
// with a two-day-stale body whose ETag described that stale entity
// faithfully (Age: 137882, x-cache: HIT). A conditional-GET client cannot
// detect that case through validators alone; the spec's signals are Age and
// the no-store directive, so both must survive the transport untouched.

// recordedStub answers from a scripted sequence of status/header/body steps
// in order, recording the If-None-Match of every request. Responses are
// built per call so each step owns a fresh body.
type recordedStub struct {
	mutex sync.Mutex
	steps []stubStep
	seen  []string
}

// stubStep is one scripted response of a recordedStub.
type stubStep struct {
	status int
	header http.Header
	body   string
}

func (s *recordedStub) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.seen = append(s.seen, req.Header.Get("If-None-Match"))

	step := s.steps[0]
	s.steps = s.steps[1:]

	return stubResponse(step.status, step.header, step.body), nil
}

func (s *recordedStub) lastValidator() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.seen) == 0 {
		return ""
	}

	return s.seen[len(s.seen)-1]
}

// TestSpecStale200SurfacesAgeToCaller pins RFC 9111 §5.1: "The presence of
// an Age header field implies that the response was not generated or
// validated by the origin server for this request." A 200 whose entity is a
// CDN's hours-old copy is indistinguishable from a fresh origin response by
// ETag alone, so the transport must hand Age to the caller verbatim: it is
// the only staleness signal a conditional-GET client has (field case: Age
// 137882 with content two days behind).
func TestSpecStale200SurfacesAgeToCaller(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		header := stubHeader(
			headerPair{"ETag", `"stale-v1"`},
			headerPair{"Age", "137882"},
			headerPair{"X-Cache", "HIT"},
			headerPair{"Content-Type", "application/json"},
		)

		return stubResponse(http.StatusOK, header, "two-day-old payload"), nil
	})

	transport := NewTransport(next, Options{})

	status, header, body := fetch(t, transport, newGetRequest(t, "https://api.dev.test/articles?per_page=30"))
	if status != http.StatusOK || body != "two-day-old payload" {
		t.Fatalf("response = %d %q, want the 200 body intact", status, body)
	}

	if got := header.Get("Age"); got != "137882" {
		t.Errorf("Age = %q, want 137882 surfaced to the caller (RFC 9111 §5.1)", got)
	}

	if got := header.Get("X-Cache"); got != "HIT" {
		t.Errorf("X-Cache = %q, want HIT surfaced to the caller", got)
	}

	if got := transport.Stats().Stored; got != 1 {
		t.Errorf("stored = %d, want 1 (the response itself is cacheable)", got)
	}
}

// TestSpecFresheningWearsThe304sValues pins RFC 9111 §4.3.4 ("the cache MUST
// update its header fields with the header fields provided in the 304") via
// the §3.2 update rules ("MUST add each header field in the provided
// response to the stored response, replacing field values that are already
// present"). Every field the 304 provides replaces the stored value on the
// rebuilt 200, including a grown Age: the age the first 200 carried must not
// survive revalidation and run backwards (the field case's masked symptom).
func TestSpecFresheningWearsThe304sValues(t *testing.T) {
	t.Parallel()

	stored := stubHeader(
		headerPair{"ETag", `"v1"`},
		headerPair{"Age", "137882"},
		headerPair{"Cache-Control", "max-age=0"},
		headerPair{"Expires", "stale-expires"},
		headerPair{"Date", "date-one"},
		headerPair{"X-Cache", "MISS"},
		headerPair{"Content-Type", "application/json"},
	)

	notModified := stubHeader(
		headerPair{"ETag", `"v1"`},
		headerPair{"Age", "138100"},
		headerPair{"Cache-Control", "max-age=60"},
		headerPair{"Expires", "fresh-expires"},
		headerPair{"Date", "date-two"},
		headerPair{"X-Cache", "HIT"},
	)

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stored, body: "payload"},
		{status: http.StatusNotModified, header: notModified},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	for name, want := range map[string]string{
		"Age":           "138100",
		"Cache-Control": "max-age=60",
		"Expires":       "fresh-expires",
		"Date":          "date-two",
		"X-Cache":       "HIT",
	} {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q, want the 304's %q (RFC 9111 §4.3.4 freshening)", name, got, want)
		}
	}

	if got := header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want the stored value (the 304 did not provide one)", got)
	}
}

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

// TestSpecFresheningUpdatesStoredValidator pins the store-side half of
// RFC 9111 §4.3.4: the 304's metadata updates the STORED response, not only
// the synthesized one. A validator the 304 returns replaces the stored one,
// so subsequent revalidations present the newest validator (field case: an
// edge that answers with the strong form of a weak validator).
func TestSpecFresheningUpdatesStoredValidator(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"ETag", `W/"v1"`},
				headerPair{"Date", "date-one"},
			),
			body: "payload",
		},
		{
			status: http.StatusNotModified,
			header: stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Date", "date-two"},
			),
		},
		{
			status: http.StatusNotModified,
			header: stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Date", "date-three"},
			),
		},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := header.Get("Date"); got != "date-two" {
		t.Fatalf("first rebuild Date = %q, want date-two from the 304", got)
	}

	_, header, _ = fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := stub.lastValidator(); got != `"v1"` {
		t.Fatalf("revalidation validator = %q, want the 304's strong form (RFC 9111 §4.3.4)", got)
	}

	if got := header.Get("Date"); got != "date-three" {
		t.Errorf("second rebuild Date = %q, want date-three (freshening must persist across rebuilds)", got)
	}

	got := transport.Stats()
	want := Stats{Hits: 2, Stored: 1, Entries: 1}
	if got != want {
		t.Errorf("stats = %+v, want %+v (freshening must not count as storing)", got, want)
	}
}

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

// TestSpecHEADRequestsBypassTheCache documents the transport's opt-out from
// RFC 9111 §4.3.5 (freshening with HEAD): the stored body belongs to GET
// responses, and synthesizing a body-carrying 200 for HEAD would violate
// HEAD's headers-only semantics. HEAD therefore never receives a stored
// validator and is never answered from the cache.
func TestSpecHEADRequestsBypassTheCache(t *testing.T) {
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

// TestSpecRestrictedPreserveListLeavesOtherFields documents the explicit
// opt-out from RFC 9111 §4.3.4 freshening: a non-empty PreserveOn304 list
// restricts the 304's contribution to the named fields, so every other
// stored value (including a stale Age) survives the rebuild. This is the
// sharp edge the nil default exists to avoid.
func TestSpecRestrictedPreserveListLeavesOtherFields(t *testing.T) {
	t.Parallel()

	stored := stubHeader(
		headerPair{"ETag", `"v"`},
		headerPair{"Date", "date-one"},
		headerPair{"Age", "137882"},
		headerPair{"X-Cache", "MISS"},
	)

	notModified := stubHeader(
		headerPair{"ETag", `"v"`},
		headerPair{"Date", "date-two"},
		headerPair{"Age", "138100"},
		headerPair{"X-Cache", "HIT"},
	)

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stored, body: "payload"},
		{status: http.StatusNotModified, header: notModified},
	}}

	transport := NewTransport(stub, Options{PreserveOn304: []string{"Date"}})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := header.Get("Date"); got != "date-two" {
		t.Errorf("Date = %q, want date-two (Date is listed)", got)
	}

	if got := header.Get("Age"); got != "137882" {
		t.Errorf("Age = %q, want the stored value (Age is not listed)", got)
	}

	if got := header.Get("X-Cache"); got != "MISS" {
		t.Errorf("X-Cache = %q, want the stored value (X-Cache is not listed)", got)
	}
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

// TestSpecUnsafeMethodInvalidatesEntry pins RFC 9111 §4.4: "A cache MUST
// invalidate the target URI ... when it receives a non-error status code in
// response to an unsafe request method" — a mutation cannot leave a
// pre-mutation body waiting to be rebuilt. Non-error means 2xx/3xx; error
// responses and safe methods (GET/OPTIONS/TRACE) leave the entry alone.
// HEAD is safe too, but it has its own freshening rules (RFC 9111 §4.3.5,
// pinned by TestSpecHeadFreshening).
func TestSpecUnsafeMethodInvalidatesEntry(t *testing.T) {
	t.Parallel()

	for _, spec := range []struct {
		name        string
		method      string
		status      int
		invalidated bool
	}{
		{name: "2xx after DELETE invalidates", method: http.MethodDelete, status: http.StatusNoContent, invalidated: true},
		{name: "3xx after POST invalidates", method: http.MethodPost, status: http.StatusFound, invalidated: true},
		{name: "error after POST keeps the entry", method: http.MethodPost, status: http.StatusInternalServerError, invalidated: false},
		{name: "safe OPTIONS keeps the entry", method: http.MethodOptions, status: http.StatusOK, invalidated: false},
	} {
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()

			next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("If-None-Match") == `"v"` {
					return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
				}

				if req.Method != http.MethodGet {
					return stubResponse(spec.status, stubHeader(), ""), nil
				}

				return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), "payload"), nil
			})

			transport := NewTransport(next, Options{})
			url := "https://example.test/thing"

			fetch(t, transport, newGetRequest(t, url))

			status, _, _ := fetch(t, transport, newSpecRequest(t, spec.method, url))
			if status != spec.status {
				t.Fatalf("%s status = %d, want passthrough %d", spec.method, status, spec.status)
			}

			wantEntries := 1
			if spec.invalidated {
				wantEntries = 0
			}

			if got := transport.Stats().Entries; got != wantEntries {
				t.Fatalf("entries after %s = %d, want %d (RFC 9111 §4.4)", spec.method, got, wantEntries)
			}

			// After invalidation the next GET refetches (a 200); with the entry
			// kept it revalidates (a 304 rebuilt from cache).
			status, _, _ = fetch(t, transport, newGetRequest(t, url))

			wantHits := int64(1)
			if spec.invalidated {
				wantHits = 0
			}

			if got := transport.Stats().Hits; got != wantHits {
				t.Errorf("hits after re-GET = %d, want %d", got, wantHits)
			}

			if status != http.StatusOK {
				t.Errorf("re-GET status = %d, want 200", status)
			}
		})
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

// TestSpecInvalidationIsScopedToTheTargetURI pins RFC 9111 §4.4's scoping: a
// cache MUST invalidate the TARGET URI of the unsafe request — and that URI
// only — so a mutation of one resource can never discard another resource's
// stored entry. After the unrelated mutation, the untouched entry still
// revalidates with a conditional GET instead of refetching.
func TestSpecInvalidationIsScopedToTheTargetURI(t *testing.T) {
	t.Parallel()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"kept"` {
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"kept"`}), ""), nil
		}

		if req.URL.Path == "/kept" {
			return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"kept"`}), "kept payload"), nil
		}

		return stubResponse(http.StatusNoContent, stubHeader(), ""), nil
	})

	transport := NewTransport(next, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/kept"))

	status, _, _ := fetch(t, transport, newSpecRequest(t, http.MethodDelete, "https://example.test/other"))
	if status != http.StatusNoContent {
		t.Fatalf("DELETE /other status = %d, want passthrough 204", status)
	}

	status, _, body := fetch(t, transport, newGetRequest(t, "https://example.test/kept"))
	if status != http.StatusOK || body != "kept payload" {
		t.Fatalf("revalidated /kept = %d %q, want the cached 200 payload", status, body)
	}

	if got := transport.Stats().Hits; got != 1 {
		t.Errorf("hits = %d, want 1 (the entry survived an unrelated mutation)", got)
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

// TestSpecNoStoreOn304DoesNotBlockRebuild pins the reading that RFC 9111
// §5.2.2.5 governs STORAGE of the response carrying it: the 304 itself is
// never stored (only 200s are), so its no-store directive cannot veto
// updating and rebuilding from the already-stored 200 (§4.3.4).
func TestSpecNoStoreOn304DoesNotBlockRebuild(t *testing.T) {
	t.Parallel()

	notModified := stubHeader(
		headerPair{"ETag", `"v"`},
		headerPair{"Cache-Control", "no-store"},
		headerPair{"Date", "fresh-date"},
	)

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"v"`}), body: "payload"},
		{status: http.StatusNotModified, header: notModified},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))
	status, header, body := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if status != http.StatusOK || body != "payload" {
		t.Fatalf("rebuilt response = %d %q, want the cached 200 despite the 304's no-store", status, body)
	}

	if got := header.Get("Date"); got != "fresh-date" {
		t.Errorf("Date = %q, want fresh-date (the 304 still freshens the stored entry)", got)
	}

	if got := transport.Stats().Hits; got != 1 {
		t.Errorf("hits = %d, want 1 (revalidation proceeded normally)", got)
	}
}

// TestSpecFresheningAddsFieldsTheStoredResponseLacks pins the additive half
// of the §3.2 update rules behind §4.3.4: the cache MUST add each header
// field in the 304, including fields the stored response never had, and the
// addition persists to the store for later rebuilds.
func TestSpecFresheningAddsFieldsTheStoredResponseLacks(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"v"`}), body: "payload"},
		{status: http.StatusNotModified, header: stubHeader(
			headerPair{"ETag", `"v"`},
			headerPair{"X-Ratelimit-Limit", "5000"},
		)},
		{status: http.StatusNotModified, header: stubHeader(headerPair{"ETag", `"v"`})},
	}}

	transport := NewTransport(stub, Options{})

	fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	_, header, _ := fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := header.Get("X-Ratelimit-Limit"); got != "5000" {
		t.Fatalf("X-Ratelimit-Limit = %q, want 5000 added by the 304 (§3.2 add rule)", got)
	}

	_, header, _ = fetch(t, transport, newGetRequest(t, "https://example.test/data"))

	if got := header.Get("X-Ratelimit-Limit"); got != "5000" {
		t.Errorf("X-Ratelimit-Limit = %q, want 5000 persisted to the store", got)
	}
}

// TestSpecAgeNeverRunsBackwardsAcrossRevalidations pins the field case's
// masked symptom over a longer conversation: every revalidation's Age
// replaces the stored one, so the rebuilt response's Age must equal the
// freshest 304's value and never decrease.
func TestSpecAgeNeverRunsBackwardsAcrossRevalidations(t *testing.T) {
	t.Parallel()

	ages := []string{"137882", "138000", "138100", "139000"}

	steps := make([]stubStep, 0, len(ages))
	steps = append(
		steps,
		stubStep{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"v"`}), body: "payload"},
	)

	for _, age := range ages[1:] {
		steps = append(steps, stubStep{
			status: http.StatusNotModified,
			header: stubHeader(headerPair{"ETag", `"v"`}, headerPair{"Age", age}),
		})
	}

	stub := &recordedStub{steps: steps}
	transport := NewTransport(stub, Options{})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))

	previous := ages[0]

	for _, wantAge := range ages[1:] {
		_, header, _ := fetch(t, transport, newGetRequest(t, url))

		age := header.Get("Age")

		if age != wantAge {
			t.Errorf("Age = %q, want the 304's %q", age, wantAge)
		}

		if age < previous {
			t.Errorf("Age = %q, ran backwards from %q", age, previous)
		}

		previous = wantAge
	}

	if got := transport.Stats().Hits; got != int64(len(ages)-1) {
		t.Errorf("hits = %d, want %d (every revalidation rebuilt from cache)", got, len(ages)-1)
	}
}

package etagclient

import (
	"net/http"
	"testing"
)

// TestSpecHeadFreshening pins RFC 9111 §4.3.5: a HEAD 200 whose validators
// and Content-Length match a stored response "SHOULD update" it ("the cache
// MUST use the header fields provided in the HEAD response to update the
// stored response", via the §3.2 update rules); "otherwise, the cache SHOULD
// consider the stored response to be stale", so the next GET fetches afresh.
// A HEAD 200 whose validators cannot be compared to the stored response at
// all counts as "otherwise": identity is unproven.
func TestSpecHeadFreshening(t *testing.T) {
	t.Parallel()

	for _, spec := range []struct {
		name          string
		headHeader    []headerPair
		freshened     bool
		wantValidator string
	}{
		{
			name:          "matching ETag and Content-Length updates the stored metadata",
			headHeader:    []headerPair{{"ETag", `"v1"`}, {"Content-Length", "7"}, {"Date", "date-two"}, {"X-RateLimit-Remaining", "41"}},
			freshened:     true,
			wantValidator: `"v1"`,
		},
		{
			name:          "weak form of the stored ETag matches",
			headHeader:    []headerPair{{"ETag", `W/"v1"`}, {"Content-Length", "7"}, {"Date", "date-two"}},
			freshened:     true,
			wantValidator: `W/"v1"`,
		},
		{
			name:          "matching Last-Modified alone is comparable when stored",
			headHeader:    []headerPair{{"Last-Modified", "tue, 01 jan 2025 00:00:00 gmt"}, {"Date", "date-two"}},
			freshened:     true,
			wantValidator: `"v1"`,
		},
		{name: "different ETag is stale", headHeader: []headerPair{{"ETag", `"v2"`}}, freshened: false},
		{
			name:          "Content-Length mismatch is stale",
			headHeader:    []headerPair{{"ETag", `"v1"`}, {"Content-Length", "999"}},
			freshened:     false,
			wantValidator: `"v1"`,
		},
		{name: "no comparable validator is stale", headHeader: []headerPair{{"Date", "date-two"}}, freshened: false},
		{
			name:          "differing Last-Modified is stale",
			headHeader:    []headerPair{{"Last-Modified", "wed, 01 jan 2026 00:00:00 gmt"}},
			freshened:     false,
			wantValidator: `"v1"`,
		},
	} {
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()

			storedHeader := stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Last-Modified", "tue, 01 jan 2025 00:00:00 gmt"},
				headerPair{"Date", "date-one"},
			)

			notModifiedHeader := stubHeader(headerPair{"ETag", `"v1"`})

			headStepHeader := stubHeader(spec.headHeader...)

			finalStep := stubStep{status: http.StatusNotModified, header: notModifiedHeader, body: ""}
			if !spec.freshened {
				finalStep = stubStep{status: http.StatusOK, header: storedHeader, body: "payload"}
			}

			stub := &recordedStub{steps: []stubStep{
				{status: http.StatusOK, header: storedHeader, body: "payload"},
				{status: http.StatusOK, header: headStepHeader, body: ""},
				finalStep,
			}}

			transport := NewTransport(stub, Options{})
			url := "https://api.dev.test/articles"

			fetch(t, transport, newGetRequest(t, url))

			head := newSpecRequest(t, http.MethodHead, url)
			headStatus, _, _ := fetch(t, transport, head)
			if headStatus != http.StatusOK {
				t.Fatalf("HEAD status = %d, want passthrough 200", headStatus)
			}

			if spec.freshened {
				if got := transport.Stats().Entries; got != 1 {
					t.Fatalf("entries after confirming HEAD = %d, want the freshened entry kept", got)
				}
			} else if got := transport.Stats().Entries; got != 0 {
				t.Fatalf("entries after non-confirming HEAD = %d, want 0 (stale, RFC 9111 §4.3.5)", got)
			}

			status, header, body := fetch(t, transport, newGetRequest(t, url))

			if status != http.StatusOK || body != "payload" {
				t.Fatalf("final GET = %d %q, want the stored body rebuilt", status, body)
			}

			if !spec.freshened {
				if got := stub.lastValidator(); got != "" {
					t.Errorf("final GET validator = %q, want none (the entry was invalidated)", got)
				}

				return
			}

			if got := stub.lastValidator(); got != spec.wantValidator {
				t.Errorf("final GET validator = %q, want the stored %s", got, spec.wantValidator)
			}

			if got := header.Get("Date"); got != "date-two" {
				t.Errorf("Date = %q, want the HEAD's date-two (§4.3.5 update)", got)
			}
		})
	}
}

// TestSpecHeadFresheningPersistsToTheStore pins that the §4.3.5 update lands
// on the stored entry, not only on a response in flight: the HEAD's fields
// must still be visible on a much later rebuild.
func TestSpecHeadFresheningPersistsToTheStore(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{
			status: http.StatusOK,
			header: stubHeader(headerPair{"ETag", `"v1"`}, headerPair{"Date", "date-one"}),
			body:   "payload",
		},
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Content-Length", "7"},
				headerPair{"Date", "date-two"},
			),
			body: "",
		},
		{status: http.StatusNotModified, header: stubHeader(), body: ""},
	}}

	transport := NewTransport(stub, Options{})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))
	fetch(t, transport, newSpecRequest(t, http.MethodHead, url))

	_, header, _ := fetch(t, transport, newGetRequest(t, url))

	if got := header.Get("Date"); got != "date-two" {
		t.Errorf("Date = %q, want the HEAD-freshened date-two on a later rebuild", got)
	}
}

// TestSpecHeadFresheningDoesNotPersistFromCacheMarker pins that the from-cache
// marker is diagnostic output, never cache state: HEAD freshening (RFC 9111
// §4.3.5 committing through the shared persistFreshened path) must strip the
// marker from the stored entry even when the stored header carried one. The
// entry is deliberately primed with a marker (a legacy or drifted store) so
// the test proves active stripping, not merely that the HEAD path never sets
// it; the rebuilt GET response still wears the marker, proving the strip did
// not disable the response-side feature.
func TestSpecHeadFresheningDoesNotPersistFromCacheMarker(t *testing.T) {
	t.Parallel()

	const fromCache = "X-From-Cache"

	stub := &recordedStub{steps: []stubStep{
		{
			status: http.StatusOK,
			header: stubHeader(headerPair{"ETag", `"v1"`}, headerPair{"Date", "date-one"}),
			body:   "payload",
		},
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Content-Length", "7"},
				headerPair{"Date", "date-two"},
			),
			body: "",
		},
		{status: http.StatusNotModified, header: stubHeader(), body: ""},
	}}

	transport := NewTransport(stub, Options{FromCacheHeader: fromCache})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))

	entry, ok := transport.cache.get(url)
	if !ok {
		t.Fatalf("priming GET stored no entry")
	}

	marked := entry
	marked.header = entry.header.Clone()
	marked.header.Set(fromCache, markerValue)
	transport.cache.set(url, marked)

	headStatus, headHeader, _ := fetch(t, transport, newSpecRequest(t, http.MethodHead, url))
	if headStatus != http.StatusOK {
		t.Fatalf("HEAD status = %d, want passthrough 200", headStatus)
	}

	if got := headHeader.Get(fromCache); got != "" {
		t.Errorf("HEAD response carries the from-cache marker; HEAD is passthrough and never rebuilt")
	}

	freshened, ok := transport.cache.get(url)
	if !ok {
		t.Fatalf("confirming HEAD dropped the stored entry")
	}

	if got := freshened.header.Get(fromCache); got != "" {
		t.Errorf(
			"stored entry carries from-cache marker %q after HEAD freshening; the marker is diagnostic output, never cache state",
			got,
		)
	}

	if got := freshened.header.Get("Date"); got != "date-two" {
		t.Errorf("stored Date = %q, want the HEAD's date-two (the freshening ran; the strip is not vacuous)", got)
	}

	_, header, _ := fetch(t, transport, newGetRequest(t, url))

	if got := header.Get(fromCache); got != markerValue {
		t.Errorf("rebuilt GET from-cache marker = %q, want %q (response-side marking still works)", got, markerValue)
	}
}

// TestSpecHeadNoStoreLeavesTheEntryAlone pins that a no-store HEAD 200
// neither updates nor invalidates the stored entry (RFC 9111 §3 forbids
// storing parts of the response; §4.3.5 gives no staleness signal either).
func TestSpecHeadNoStoreLeavesTheEntryAlone(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{status: http.StatusOK, header: stubHeader(headerPair{"ETag", `"v1"`}), body: "payload"},
		{
			status: http.StatusOK,
			header: stubHeader(headerPair{"ETag", `"v1"`}, headerPair{"Cache-Control", "no-store"}),
			body:   "",
		},
		{status: http.StatusNotModified, header: stubHeader(), body: ""},
	}}

	transport := NewTransport(stub, Options{})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))
	fetch(t, transport, newSpecRequest(t, http.MethodHead, url))

	if got := transport.Stats().Entries; got != 1 {
		t.Fatalf("entries after no-store HEAD = %d, want the entry untouched", got)
	}

	status, header, body := fetch(t, transport, newGetRequest(t, url))

	if status != http.StatusOK || body != "payload" || header.Get("ETag") != `"v1"` {
		t.Fatalf("final GET = %d %q ETag %q, want the untouched entry rebuilt", status, body, header.Get("ETag"))
	}

	if got := transport.Stats().Hits; got != 1 {
		t.Errorf("hits = %d, want 1 (the entry still revalidates)", got)
	}
}

package etagclient

import (
	"net/http"
	"testing"
)

// TestStoredValidatorWeaklyMatches pins the comparison the §4.3.4 validator
// filter relies on: the same opaque tag matches regardless of the W/ marker,
// in either argument, while a lowercase w/ is not a weakness marker and an
// unparseable value on either side never matches.
func TestStoredValidatorWeaklyMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stored    string
		candidate string
		want      bool
	}{
		{name: "identical strong forms", stored: `"v"`, candidate: `"v"`, want: true},
		{name: "weak vs strong", stored: `W/"v"`, candidate: `"v"`, want: true},
		{name: "strong vs weak", stored: `"v"`, candidate: `W/"v"`, want: true},
		{name: "both weak", stored: `W/"v"`, candidate: `W/"v"`, want: true},
		{name: "different opaque tags", stored: `"a"`, candidate: `"b"`, want: false},
		{name: "lowercase w is not a marker", stored: `w/"v"`, candidate: `"v"`, want: false},
		{name: "unparseable stored validator", stored: "", candidate: `"v"`, want: false},
		{name: "unparseable candidate", stored: `"v"`, candidate: "", want: false},
		{name: "unclosed quote never matches", stored: `"v`, candidate: `"v"`, want: false},
		{name: "wildcard is not a comparable validator", stored: "*", candidate: "*", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			validator := newStoredValidator(tt.stored)
			if got := validator.weaklyMatches(tt.candidate); got != tt.want {
				t.Errorf(
					"newStoredValidator(%q).weaklyMatches(%q) = %v, want %v",
					tt.stored,
					tt.candidate,
					got,
					tt.want,
				)
			}
		})
	}
}

// TestNewStoredValidatorReplaysWireVerbatim pins that the stored identity
// keeps the exact field value the server sent — parse state never rewrites
// it — so If-None-Match replays a validator the server can actually compare,
// including values it sent outside the RFC grammar.
func TestNewStoredValidatorReplaysWireVerbatim(t *testing.T) {
	t.Parallel()

	for _, wire := range []string{`"v"`, `W/"v"`, `w/"v"`, "*", `"unclosed`, ""} {
		t.Run(wire, func(t *testing.T) {
			t.Parallel()

			if got := newStoredValidator(wire).wire; got != wire {
				t.Errorf("wire = %q, want verbatim %q", got, wire)
			}
		})
	}
}

// TestFreshenFromHeadKeepsValidatorWhenHeaderLacksETag pins the
// persistFreshened validator fallback: a HEAD-freshened entry whose header
// set carries no ETag field still revalidates later with the stored
// validator wire instead of orphaning the entry. The HEAD confirms via
// Last-Modified alone (the RFC 9111 §4.3.5 comparable-validator rule), so
// no ETag ever flows into the freshened header and only the fallback keeps
// the validator alive.
func TestFreshenFromHeadKeepsValidatorWhenHeaderLacksETag(t *testing.T) {
	t.Parallel()

	stub := &recordedStub{steps: []stubStep{
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Last-Modified", "tue, 01 jan 2025 00:00:00 gmt"},
				headerPair{"Date", "date-one"},
			),
			body: "payload",
		},
		{
			status: http.StatusOK,
			header: stubHeader(
				headerPair{"Last-Modified", "tue, 01 jan 2025 00:00:00 gmt"},
				headerPair{"Date", "date-two"},
			),
			body: "",
		},
		{status: http.StatusNotModified, header: stubHeader(), body: ""},
	}}

	transport := NewTransport(stub, Options{})
	url := "https://api.dev.test/articles"

	fetch(t, transport, newGetRequest(t, url))

	entry, ok := transport.cache.get(url)
	if !ok {
		t.Fatalf("priming GET stored no entry")
	}

	etagless := entry
	etagless.header = entry.header.Clone()
	etagless.header.Del("ETag")
	transport.cache.set(url, etagless)

	fetch(t, transport, newSpecRequest(t, http.MethodHead, url))

	if got := transport.Stats().Entries; got != 1 {
		t.Fatalf("entries after confirming HEAD = %d, want the freshened entry kept", got)
	}

	_, header, _ := fetch(t, transport, newGetRequest(t, url))

	if got := stub.lastValidator(); got != `"v1"` {
		t.Errorf("revalidation validator = %q, want the stored %q (the fallback kept it)", got, `"v1"`)
	}

	if got := header.Get("Date"); got != "date-two" {
		t.Errorf("Date = %q, want the HEAD's date-two (the freshening ran)", got)
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

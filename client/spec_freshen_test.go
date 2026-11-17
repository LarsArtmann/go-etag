package etagclient

import (
	"net/http"
	"testing"
)

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

// TestSpecRestrictedFreshenListLeavesOtherFields documents the explicit
// opt-out from RFC 9111 §4.3.4 freshening: a FreshenFields policy restricts
// the 304's contribution to the named fields, so every other stored value
// (including a stale Age) survives the rebuild.
func TestSpecRestrictedFreshenListLeavesOtherFields(t *testing.T) {
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

	transport := NewTransport(stub, Options{FreshenOn304: FreshenFields("Date")})

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

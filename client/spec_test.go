package etagclient

import (
	"net/http"
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

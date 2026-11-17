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

// failingBody yields its data once, then fails with errBodyBoom forever.
type failingBody struct {
	data []byte
	read int
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

// newSpecRequest builds a request of any method bound to the test context.
func newSpecRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
	if err != nil {
		t.Fatalf("build %s request: %v", method, err)
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

// errCloseBoom is the sentinel closeFailingBody replays from Close.
var errCloseBoom = errors.New("close boom")

// closeFailingBody yields its data normally but fails on Close, exercising
// the store path that must keep the response re-readable when the original
// body cannot be released.
type closeFailingBody struct {
	data []byte
	read int
}

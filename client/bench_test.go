package etagclient

import (
	"io"
	"net/http"
	"testing"
)

func benchmarkTransport() *Transport {
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"v"` {
			return stubResponse(http.StatusNotModified, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
		}

		return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), "benchmark body"), nil
	})

	return NewTransport(next, Options{})
}

func benchmarkRoundTrip(b *testing.B, warm bool) {
	b.Helper()

	transport := benchmarkTransport()
	req := newBenchRequest(b)

	if warm {
		warmResp, err := transport.RoundTrip(req)
		if err != nil {
			b.Fatalf("warm-up: %v", err)
		}

		_ = warmResp.Body.Close()
	}

	b.ResetTimer()

	for b.Loop() {
		resp, err := transport.RoundTrip(req)
		if err != nil {
			b.Fatalf("GET: %v", err)
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

func newBenchRequest(b *testing.B) *http.Request {
	b.Helper()

	req, err := http.NewRequestWithContext(b.Context(), http.MethodGet, "https://example.test/bench", nil)
	if err != nil {
		b.Fatalf("build request: %v", err)
	}

	return req
}

func BenchmarkTransportFresh200(b *testing.B) {
	benchmarkRoundTrip(b, false)
}

func BenchmarkTransport304Rebuild(b *testing.B) {
	benchmarkRoundTrip(b, true)
}

func benchmarkUnsafeRoundTrip(b *testing.B, statusCode int) {
	b.Helper()

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return stubResponse(statusCode, stubHeader(headerPair{"ETag", `"v"`}), ""), nil
	})

	transport := NewTransport(next, Options{})

	req, err := http.NewRequestWithContext(b.Context(), http.MethodPost, "https://example.test/bench", nil)
	if err != nil {
		b.Fatalf("build request: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		resp, roundTripErr := transport.RoundTrip(req)
		if roundTripErr != nil {
			b.Fatalf("POST: %v", roundTripErr)
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// BenchmarkTransportUnsafeErrorPassthrough isolates the fixed cost every
// non-GET round trip pays before any invalidation is decided (a 500 never
// invalidates), so regressions here are pure per-request overhead.
func BenchmarkTransportUnsafeErrorPassthrough(b *testing.B) {
	benchmarkUnsafeRoundTrip(b, http.StatusInternalServerError)
}

// BenchmarkTransportUnsafeInvalidation covers the full non-GET path including
// the RFC 9111 §4.4 invalidation of the stored entry.
func BenchmarkTransportUnsafeInvalidation(b *testing.B) {
	benchmarkUnsafeRoundTrip(b, http.StatusNoContent)
}

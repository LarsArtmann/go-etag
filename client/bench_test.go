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

// BenchmarkTransportHeadFreshening isolates the RFC 9111 §4.3.5 path: every
// iteration is a confirming HEAD 200 that freshens the stored entry. The
// 2026-09-11 dedup routed freshenFromHead through persistFreshened, adding
// one header.Clone() per freshening HEAD; this benchmark pairs the pre-dedup
// baseline against the routed implementation
// (reports/bench/2026-09-23_before-head-freshening.txt and its after twin).
func BenchmarkTransportHeadFreshening(b *testing.B) {
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodHead {
			header := stubHeader(
				headerPair{"ETag", `"v"`},
				headerPair{"Content-Length", "14"},
				headerPair{"Date", "fresh-date"},
			)

			return stubResponse(http.StatusOK, header, ""), nil
		}

		return stubResponse(http.StatusOK, stubHeader(headerPair{"ETag", `"v"`}), "benchmark body"), nil
	})

	transport := NewTransport(next, Options{})

	getReq, err := http.NewRequestWithContext(b.Context(), http.MethodGet, "https://example.test/bench", nil)
	if err != nil {
		b.Fatalf("build GET: %v", err)
	}

	headReq, err := http.NewRequestWithContext(b.Context(), http.MethodHead, "https://example.test/bench", nil)
	if err != nil {
		b.Fatalf("build HEAD: %v", err)
	}

	primeResp, primeErr := transport.RoundTrip(getReq)
	if primeErr != nil {
		b.Fatalf("prime GET: %v", primeErr)
	}

	_, _ = io.Copy(io.Discard, primeResp.Body)
	_ = primeResp.Body.Close()

	b.ResetTimer()

	for b.Loop() {
		resp, roundTripErr := transport.RoundTrip(headReq)
		if roundTripErr != nil {
			b.Fatalf("HEAD: %v", roundTripErr)
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

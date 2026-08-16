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

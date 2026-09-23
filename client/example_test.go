package etagclient

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// ExampleNewTransport wraps a stubbed origin with a conditional GET cache and
// fetches the same URL twice: the second GET revalidates via If-None-Match
// and is rebuilt from cache.
func ExampleNewTransport() {
	var calls int

	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls++

		if req.Header.Get("If-None-Match") == `"v1"` {
			header := stubHeader(headerPair{"ETag", `"v1"`}, headerPair{"Date", "now"})

			return stubResponse(http.StatusNotModified, header, ""), nil
		}

		header := stubHeader(headerPair{"ETag", `"v1"`}, headerPair{"Date", "then"})

		return stubResponse(http.StatusOK, header, "hello"), nil
	})

	transport := NewTransport(next, Options{FromCacheHeader: "X-From-Cache"})
	client := &http.Client{Transport: transport}

	for range 2 {
		resp, err := client.Get("https://example.test/greeting")
		if err != nil {
			fmt.Println("error:", err)

			return
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if readErr != nil {
			fmt.Println("error:", readErr)

			return
		}

		fmt.Println(resp.StatusCode, string(body), "from-cache="+resp.Header.Get("X-From-Cache"))
	}

	fmt.Println("network calls:", calls)
	fmt.Printf("stats: %+v\n", transport.Stats())

	// Output:
	// 200 hello from-cache=
	// 200 hello from-cache=1
	// network calls: 2
	// stats: {Hits:1 Stored:1 Entries:1}
}

// Example_ageAwareStalenessCheck demonstrates the field case that motivated
// Age surfacing: an edge cache can answer 200 with a faithful ETag attached
// to a days-old entity. A conditional-GET client that cares about freshness
// reads the Age the transport surfaces and rejects responses older than its
// tolerance, because ETag alone cannot detect that case.
func Example_ageAwareStalenessCheck() {
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"stale-v1"` {
			header := stubHeader(headerPair{"ETag", `"stale-v1"`}, headerPair{"Age", "137900"})

			return stubResponse(http.StatusNotModified, header, ""), nil
		}

		header := stubHeader(headerPair{"ETag", `"stale-v1"`}, headerPair{"Age", "137882"})

		return stubResponse(http.StatusOK, header, "two-day-old payload"), nil
	})

	client := &http.Client{Transport: NewTransport(next, Options{})}

	resp, err := client.Get("https://api.dev.test/articles")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if readErr != nil {
		fmt.Println("error:", readErr)

		return
	}

	age, parseErr := strconv.Atoi(resp.Header.Get("Age"))
	if parseErr != nil {
		fmt.Println("error:", parseErr)

		return
	}

	const staleAfterSeconds = 24 * 60 * 60

	if age > staleAfterSeconds {
		fmt.Printf(
			"rejecting %q: Age %d exceeds %d seconds, the edge may hold stale content\n",
			string(body),
			age,
			staleAfterSeconds,
		)

		return
	}

	fmt.Println("fresh:", string(body))

	// Output:
	// rejecting "two-day-old payload": Age 137882 exceeds 86400 seconds, the edge may hold stale content
}

// ExampleFreshenPolicy restricts 304 freshening to one field: the rebuilt
// 200 wears the fresh Retry-After the revalidation provides, while Date
// keeps the stored value and the validator flows through whatever the
// policy is.
func ExampleFreshenPolicy() {
	next := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"v1"` {
			header := stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Date", "later"},
				headerPair{"Retry-After", "30"},
			)

			return stubResponse(http.StatusNotModified, header, ""), nil
		}

		header := stubHeader(
			headerPair{"ETag", `"v1"`},
			headerPair{"Date", "then"},
			headerPair{"Retry-After", "120"},
		)

		return stubResponse(http.StatusOK, header, "job status"), nil
	})

	transport := NewTransport(next, Options{FreshenOn304: FreshenFields("Retry-After")})
	client := &http.Client{Transport: transport}

	for range 2 {
		resp, err := client.Get("https://api.test/jobs/42")
		if err != nil {
			fmt.Println("error:", err)

			return
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if readErr != nil {
			fmt.Println("error:", readErr)

			return
		}

		fmt.Println(
			resp.StatusCode,
			string(body),
			"date="+resp.Header.Get("Date"),
			"retry-after="+resp.Header.Get("Retry-After"),
			"etag="+resp.Header.Get("ETag"),
		)
	}

	// Output:
	// 200 job status date=then retry-after=120 etag="v1"
	// 200 job status date=then retry-after=30 etag="v1"
}

// fresheningStub answers a conditional-GET sequence: a first 200 carrying
// Date=then / Retry-After=120, then a 304 providing Date=later /
// Retry-After=30. The freshening examples differ only in policy.
func fresheningStub() roundTripperFunc {
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("If-None-Match") == `"v1"` {
			header := stubHeader(
				headerPair{"ETag", `"v1"`},
				headerPair{"Date", "later"},
				headerPair{"Retry-After", "30"},
			)

			return stubResponse(http.StatusNotModified, header, ""), nil
		}

		header := stubHeader(
			headerPair{"ETag", `"v1"`},
			headerPair{"Date", "then"},
			headerPair{"Retry-After", "120"},
		)

		return stubResponse(http.StatusOK, header, "job status"), nil
	})
}

// ExampleFreshenPerRFC shows the default 304 freshening (RFC 9111 §4.3.4,
// the zero value): every field the revalidation provides replaces the
// stored value, so the rebuilt 200 wears the fresh Date and Retry-After
// while body and validator flow through.
func ExampleFreshenPerRFC() {
	client := &http.Client{Transport: NewTransport(fresheningStub(), Options{FreshenOn304: FreshenPerRFC()})}

	for range 2 {
		resp, err := client.Get("https://api.test/jobs/42")
		if err != nil {
			fmt.Println("error:", err)

			return
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if readErr != nil {
			fmt.Println("error:", readErr)

			return
		}

		fmt.Println(
			resp.StatusCode,
			string(body),
			"date="+resp.Header.Get("Date"),
			"retry-after="+resp.Header.Get("Retry-After"),
		)
	}

	// Output:
	// 200 job status date=then retry-after=120
	// 200 job status date=later retry-after=30
}

// ExampleFreshenNone disables 304 freshening: the rebuilt 200 keeps every
// stored field and only the body replay and the validator revalidation
// remain — opting out of the RFC 9111 §4.3.4 update is the policy's job.
func ExampleFreshenNone() {
	client := &http.Client{Transport: NewTransport(fresheningStub(), Options{FreshenOn304: FreshenNone()})}

	for range 2 {
		resp, err := client.Get("https://api.test/jobs/42")
		if err != nil {
			fmt.Println("error:", err)

			return
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if readErr != nil {
			fmt.Println("error:", readErr)

			return
		}

		fmt.Println(
			resp.StatusCode,
			string(body),
			"date="+resp.Header.Get("Date"),
			"retry-after="+resp.Header.Get("Retry-After"),
		)
	}

	// Output:
	// 200 job status date=then retry-after=120
	// 200 job status date=then retry-after=120
}

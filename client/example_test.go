package etagclient

import (
	"fmt"
	"io"
	"net/http"
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

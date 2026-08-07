package etag

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func ExampleNew() {
	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("ETag") != "")

	// Output: true
}

func ExampleETag() {
	tag := NewETag("abc123", Strong)

	fmt.Println(tag)
	fmt.Println(tag.IsWeak())

	weak := NewETag("abc123", Weak)
	fmt.Println(weak)

	// Output:
	// "abc123"
	// false
	// W/"abc123"
}

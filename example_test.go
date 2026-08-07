package etag

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func ExampleETag() {
	cfg := DefaultETagConfig()
	handler := ETag(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("ETag") != "")

	// Output: true
}

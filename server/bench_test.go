package etag

import (
	"net/http"
	"testing"
)

func BenchmarkETag(b *testing.B) {
	handler := New(DefaultETagConfig())(newWriteBodyHandler([]byte("hello world benchmark test data")))

	req := newTestRequest(http.MethodGet)

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkETag_IfNoneMatch(b *testing.B) {
	handler := New(DefaultETagConfig())(newWriteBodyHandler([]byte("hello world benchmark test data")))

	req := newTestRequest(http.MethodGet)
	req.Header.Set(headerIfNoneMatch, `"779a65e7023cd2e7"`)

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkMatchesIfNoneMatch(b *testing.B) {
	single := `"779a65e7023cd2e7"`
	multi := `"abc123", W/"def456", "779a65e7023cd2e7"`
	weak := `W/"779a65e7023cd2e7"`
	tag := NewETag("779a65e7023cd2e7", Strong)

	b.Run("single", func(b *testing.B) {
		for b.Loop() {
			_ = MatchesIfNoneMatch(tag, single)
		}
	})

	b.Run("multi", func(b *testing.B) {
		for b.Loop() {
			_ = MatchesIfNoneMatch(tag, multi)
		}
	})

	b.Run("weak", func(b *testing.B) {
		for b.Loop() {
			_ = MatchesIfNoneMatch(tag, weak)
		}
	})
}

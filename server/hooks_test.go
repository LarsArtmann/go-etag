package etag

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNew_OnError_StreamingWriteFailure(t *testing.T) {
	t.Parallel()

	var capturedErr *errorfamily.Error

	cfg := ETagConfig{
		OnError: func(e *errorfamily.Error) {
			capturedErr = e
		},
	}

	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("first chunk"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()

		_, _ = w.Write([]byte("second chunk"))
	}))

	req := newTestRequest(http.MethodGet)
	rec := &failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)

	if capturedErr == nil {
		t.Fatal("OnError was not called on streaming write failure")
	}

	if capturedErr.ErrorFamily() != errorfamily.Transient {
		t.Errorf("OnError error family = %s, want Transient", capturedErr.ErrorFamily())
	}
}

func TestNew_OverflowWriteError(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 10}

	var writeErr error
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, writeErr = w.Write([]byte(strings.Repeat("x", 100)))
	}))

	req := newTestRequest(http.MethodGet)
	rec := &failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)

	if writeErr == nil {
		t.Fatal("Write returned nil error for overflow to failing writer")
	}

	classified, ok := errors.AsType[*errorfamily.Error](writeErr)
	if !ok {
		t.Fatalf("Write error is %T, want *errorfamily.Error", writeErr)
	}

	if classified.Code() != ErrCodeETagWriteFailed {
		t.Errorf("Write error code = %q, want %q", classified.Code(), ErrCodeETagWriteFailed)
	}
}

func TestNew_OverflowWriteError_NilOnError_DoesNotPanic(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 10}

	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	}))

	req := newTestRequest(http.MethodGet)
	rec := &failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)
}

func TestNew_FlushWriteError_NilOnError_DoesNotPanic(t *testing.T) {
	t.Parallel()

	handler := New(DefaultETagConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("hello world"))

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not implement http.Flusher")
		}

		flusher.Flush()

		_, _ = w.Write([]byte("after flush"))
	}))

	req := newTestRequest(http.MethodGet)
	rec := &failingResponseRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.ServeHTTP(rec, req)
}

// --- Custom HashFunc (D3) ---

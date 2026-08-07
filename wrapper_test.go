package etag

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestHijackDelegate_UnsupportedWriter(t *testing.T) {
	t.Parallel()

	rec := newNonHijackableRecorder()

	_, _, err := hijackDelegate(rec)

	if err == nil {
		t.Fatal("hijackDelegate returned nil error for non-hijackable writer")
	}

	var classified *errorfamily.Error
	if !errors.As(err, &classified) {
		t.Fatalf("error is %T, want *errorfamily.Error", err)
	}

	if classified.ErrorFamily() != errorfamily.Infrastructure {
		t.Errorf("family = %s, want Infrastructure", classified.ErrorFamily())
	}

	if classified.Code() != ErrCodeHijackUnsupported {
		t.Errorf("code = %q, want %q", classified.Code(), ErrCodeHijackUnsupported)
	}
}

func TestHijackDelegate_HijackFails(t *testing.T) {
	t.Parallel()

	rec := newFailingHijackRecorder()

	_, _, err := hijackDelegate(rec)

	if err == nil {
		t.Fatal("hijackDelegate returned nil error for failing hijack writer")
	}

	var classified *errorfamily.Error
	if !errors.As(err, &classified) {
		t.Fatalf("error is %T, want *errorfamily.Error", err)
	}

	if classified.ErrorFamily() != errorfamily.Transient {
		t.Errorf("family = %s, want Transient", classified.ErrorFamily())
	}

	if classified.Code() != ErrCodeHijackFailed {
		t.Errorf("code = %q, want %q", classified.Code(), ErrCodeHijackFailed)
	}

	if !errors.Is(classified, errMockHijackFailed) {
		t.Errorf("error does not wrap the underlying cause %v", errMockHijackFailed)
	}
}

func TestHijackDelegate_Success(t *testing.T) {
	t.Parallel()

	rec := newHijackRecorder()

	conn, rw, err := hijackDelegate(rec)
	if err != nil {
		t.Fatalf("hijackDelegate returned unexpected error: %v", err)
	}

	if conn != nil {
		t.Error("conn = non-nil, want nil from test double")
	}

	if rw != nil {
		t.Error("rw = non-nil, want nil from test double")
	}

	if !rec.hijacked {
		t.Error("underlying Hijack was not called")
	}
}

func TestResponseWrapper_Flush_NonFlusher(t *testing.T) {
	t.Parallel()

	// flushDelegate on a writer that does not implement http.Flusher must
	// be a no-op, not a panic.
	rec := newNonHijackableRecorder()

	flushDelegate(rec)
}

func TestResponseWrapper_WriteHeader_CalledTwice_KeepsFirstStatus(t *testing.T) {
	t.Parallel()

	w := newResponseWrapper(httptest.NewRecorder())

	w.WriteHeader(http.StatusNotFound)
	w.WriteHeader(http.StatusOK)

	w.writeHeaderToUnderlying()

	if w.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d (first WriteHeader wins)", w.status, http.StatusNotFound)
	}
}

func TestResponseWrapper_WriteDefaultOK_WhenNoHeaderWritten(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	w := newResponseWrapper(rec)

	w.writeDefaultOK()
	w.writeHeaderToUnderlying()

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (implicit default)", rec.Code)
	}
}

func TestResponseWrapper_WriteDefaultOK_WhenHeaderAlreadyWritten(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	w := newResponseWrapper(rec)

	w.WriteHeader(http.StatusTeapot)
	w.writeDefaultOK()
	w.writeHeaderToUnderlying()

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d (explicit header preserved)", rec.Code, http.StatusTeapot)
	}
}

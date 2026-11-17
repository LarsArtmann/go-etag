package etag

import (
	"errors"
	"net/http"
	"testing"
)

func TestETagConfig_Validate_ValidDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultETagConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestETagConfig_Validate_ZeroMaxBufferSize(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: 0}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for zero MaxBufferSize")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Validate() error = %v, want ErrInvalidConfig", err)
	}
}

func TestETagConfig_Validate_NegativeMaxBufferSize(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{MaxBufferSize: -1}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative MaxBufferSize")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Validate() error = %v, want ErrInvalidConfig", err)
	}
}

func TestETagConfig_Validate_InvalidStrength(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{Strength: Strength(42)}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for invalid Strength")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Validate() error = %v, want ErrInvalidConfig", err)
	}
}

func TestErrInvalidConfig_MatchesDerivedErrors(t *testing.T) {
	t.Parallel()

	if ErrInvalidConfig == nil {
		t.Fatal("ErrInvalidConfig sentinel is nil")
	}

	if !errors.Is(ErrInvalidConfig, ErrInvalidConfig) {
		t.Error("errors.Is(ErrInvalidConfig, ErrInvalidConfig) = false, want true")
	}

	derived := newInvalidConfig().WithContextf("max_buffer_size", "%d", -1)

	if !errors.Is(derived, ErrInvalidConfig) {
		t.Errorf("errors.Is(derived, ErrInvalidConfig) = false for derived error %v, want true", derived)
	}
}

// --- Error handling ---

func TestNew_CustomHashFunc(t *testing.T) {
	t.Parallel()

	cfg := ETagConfig{
		HashFunc: func(_ []byte) string {
			return "custom-hash"
		},
	}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "hello world"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertETag(t, rec, `"custom-hash"`)
}

func TestNew_CustomHashFunc_ReceivesBody(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	cfg := ETagConfig{
		HashFunc: func(b []byte) string {
			receivedBody = b

			return "from-body"
		},
	}
	handler := New(cfg)(newWriteStatusHandler(http.StatusOK, "the actual body"))

	req := newTestRequest(http.MethodGet)
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if string(receivedBody) != "the actual body" {
		t.Errorf("HashFunc received %q, want %q", string(receivedBody), "the actual body")
	}
}

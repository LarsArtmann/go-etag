package etag

import (
	"net/http"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestRegisterErrorClassifications_DoesNotPanic(t *testing.T) {
	t.Parallel()

	// RegisterErrorClassifications is idempotent; calling it multiple times
	// must not panic or corrupt state.
	RegisterErrorClassifications()
	RegisterErrorClassifications()
}

func TestRegisterErrorClassifications_ClassifiesHTTPErrors(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	tests := []struct {
		name   string
		err    error
		family errorfamily.Family
	}{
		{name: "ErrNotSupported is Infrastructure", err: http.ErrNotSupported, family: errorfamily.Infrastructure},
		{name: "ErrAbortHandler is Transient", err: http.ErrAbortHandler, family: errorfamily.Transient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errorfamily.Classify(tt.err)
			if got != tt.family {
				t.Errorf("Classify(%v) = %s, want %s", tt.err, got, tt.family)
			}
		})
	}
}

func TestRegisterErrorClassifications_RegistersAllTemplates(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	codes := []string{
		ErrCodeETagWriteFailed,
		ErrCodeHijackUnsupported,
		ErrCodeHijackFailed,
		ErrCodeInvalidConfig,
		ErrCodeHashWriteFailed,
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			t.Parallel()

			tmpl, ok := errorfamily.TemplateForCode(code)
			if !ok {
				t.Fatalf("TemplateForCode(%q) returned ok=false, want registered template", code)
			}

			if tmpl.What == "" {
				t.Errorf("template for %q has empty What field", code)
			}

			if tmpl.Why == "" {
				t.Errorf("template for %q has empty Why field", code)
			}

			if tmpl.Fix == "" {
				t.Errorf("template for %q has empty Fix field", code)
			}

			if tmpl.WayOut == "" {
				t.Errorf("template for %q has empty WayOut field", code)
			}
		})
	}
}

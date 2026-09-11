package etag

import (
	"errors"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errTestCause = errors.New("cause")

var errTestPlain = errors.New("plain")

func TestCodeDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code Code
		want Domain
	}{
		{name: "etag write code has http domain", code: codeETagWriteFailed, want: Domain("http")},
		{name: "code without dot is its own domain", code: Code("bare"), want: Domain("bare")},
		{name: "empty code is empty domain", code: Code(""), want: Domain("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.code.Domain()
			if got != tt.want {
				t.Errorf("Domain() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCodeConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		got    *errorfamily.Error
		family errorfamily.Family
	}{
		{name: "Rejection", got: codeInvalidConfig.Rejection("msg"), family: errorfamily.Rejection},
		{name: "Conflict", got: codeInvalidConfig.Conflict("msg"), family: errorfamily.Conflict},
		{name: "Transient", got: codeInvalidConfig.Transient("msg"), family: errorfamily.Transient},
		{name: "Corruption", got: codeInvalidConfig.Corruption("msg"), family: errorfamily.Corruption},
		{name: "Infrastructure", got: codeInvalidConfig.Infrastructure("msg"), family: errorfamily.Infrastructure},
		{name: "Orchestration", got: codeInvalidConfig.Orchestration("msg"), family: errorfamily.Orchestration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got.ErrorFamily() != tt.family {
				t.Errorf("family = %v, want %v", tt.got.ErrorFamily(), tt.family)
			}

			if tt.got.ErrorCode() != string(codeInvalidConfig) {
				t.Errorf("code = %q, want %q", tt.got.ErrorCode(), string(codeInvalidConfig))
			}

			if !strings.Contains(tt.got.Error(), "msg") {
				t.Errorf("message = %q, want it to contain %q", tt.got.Error(), "msg")
			}
		})
	}
}

func TestCodeWrapMethods(t *testing.T) {
	t.Parallel()

	cause := errTestCause

	tests := []struct {
		name   string
		got    *errorfamily.Error
		family errorfamily.Family
	}{
		{name: "WrapRejection", got: codeInvalidConfig.WrapRejection(cause, "msg"), family: errorfamily.Rejection},
		{name: "WrapConflict", got: codeInvalidConfig.WrapConflict(cause, "msg"), family: errorfamily.Conflict},
		{name: "WrapTransient", got: codeInvalidConfig.WrapTransient(cause, "msg"), family: errorfamily.Transient},
		{name: "WrapCorruption", got: codeInvalidConfig.WrapCorruption(cause, "msg"), family: errorfamily.Corruption},
		{
			name:   "WrapInfrastructure",
			got:    codeInvalidConfig.WrapInfrastructure(cause, "msg"),
			family: errorfamily.Infrastructure,
		},
		{
			name:   "WrapOrchestration",
			got:    codeInvalidConfig.WrapOrchestration(cause, "msg"),
			family: errorfamily.Orchestration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got.ErrorFamily() != tt.family {
				t.Errorf("family = %v, want %v", tt.got.ErrorFamily(), tt.family)
			}

			if !errors.Is(tt.got, cause) {
				t.Errorf("wrapped error does not match its cause")
			}
		})
	}
}

func TestDomainOf(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	t.Run("classified error yields its domain", func(t *testing.T) {
		t.Parallel()

		domain, ok := DomainOf(newInvalidConfig())
		if !ok {
			t.Fatalf("DomainOf returned ok=false for a classified error")
		}

		if domain != Domain("http") {
			t.Errorf("domain = %q, want %q", domain, Domain("http"))
		}
	})

	t.Run("uncoded error reports false", func(t *testing.T) {
		t.Parallel()

		domain, ok := DomainOf(errTestPlain)
		if ok {
			t.Fatalf("DomainOf returned ok=%v for an uncoded error", ok)
		}

		if domain != "" {
			t.Errorf("domain = %q, want empty", domain)
		}
	})
}

func TestInDomain(t *testing.T) {
	t.Parallel()

	RegisterErrorClassifications()

	if !InDomain(newInvalidConfig(), Domain("http")) {
		t.Errorf("InDomain = false, want true for an http-domain error")
	}

	if InDomain(newInvalidConfig(), Domain("other")) {
		t.Errorf("InDomain = true, want false for a different domain")
	}

	if InDomain(errTestPlain, Domain("http")) {
		t.Errorf("InDomain = true, want false for an uncoded error")
	}
}

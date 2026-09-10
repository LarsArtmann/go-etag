package etagclient

import (
	"testing"
)

func TestOptionsNormalizeDefaults(t *testing.T) {
	t.Parallel()

	normalized := Options{}.normalize()

	if normalized.MaxEntries != defaultMaxEntries {
		t.Errorf("MaxEntries = %d, want %d", normalized.MaxEntries, defaultMaxEntries)
	}

	if normalized.MaxBodyBytes != defaultMaxBodyBytes {
		t.Errorf("MaxBodyBytes = %d, want %d", normalized.MaxBodyBytes, defaultMaxBodyBytes)
	}

	// The zero FreshenPolicy is meaningful: it selects RFC 9111 §4.3.4
	// freshening wholesale, so normalize must keep it untouched.
	if normalized.FreshenOn304 != (FreshenPolicy{}) {
		t.Errorf("FreshenOn304 = %+v, want the zero policy (RFC 9111 §4.3.4 freshening)", normalized.FreshenOn304)
	}

	if normalized.KeyFunc == nil {
		t.Fatal("KeyFunc must default to a non-nil function")
	}
}

func TestOptionsNormalizeKeepsExplicitValues(t *testing.T) {
	t.Parallel()

	normalized := Options{
		MaxEntries:   3,
		MaxBodyBytes: 512,
		FreshenOn304: FreshenFields("Date"),
	}.normalize()

	if normalized.MaxEntries != 3 {
		t.Errorf("MaxEntries = %d, want 3", normalized.MaxEntries)
	}

	if normalized.MaxBodyBytes != 512 {
		t.Errorf("MaxBodyBytes = %d, want 512", normalized.MaxBodyBytes)
	}

	if normalized.FreshenOn304 != FreshenFields("Date") {
		t.Errorf("FreshenOn304 = %+v, want the provided policy untouched", normalized.FreshenOn304)
	}
}

func TestDefaultKeyFuncUsesURL(t *testing.T) {
	t.Parallel()

	req := newGetRequest(t, "https://example.test/things?a=1")

	if got := defaultKeyFunc(req); got != "https://example.test/things?a=1" {
		t.Errorf("defaultKeyFunc = %q, want the request URL", got)
	}
}

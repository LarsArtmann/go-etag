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

	if len(normalized.PreserveOn304) != 1 || normalized.PreserveOn304[0] != "Date" {
		t.Errorf("PreserveOn304 = %v, want [Date]", normalized.PreserveOn304)
	}

	if normalized.KeyFunc == nil {
		t.Fatal("KeyFunc must default to a non-nil function")
	}
}

func TestOptionsNormalizeKeepsExplicitValues(t *testing.T) {
	t.Parallel()

	normalized := Options{
		MaxEntries:    3,
		MaxBodyBytes:  512,
		PreserveOn304: []string{},
	}.normalize()

	if normalized.MaxEntries != 3 {
		t.Errorf("MaxEntries = %d, want 3", normalized.MaxEntries)
	}

	if normalized.MaxBodyBytes != 512 {
		t.Errorf("MaxBodyBytes = %d, want 512", normalized.MaxBodyBytes)
	}

	if normalized.PreserveOn304 == nil || len(normalized.PreserveOn304) != 0 {
		t.Errorf("PreserveOn304 = %v, want empty non-nil slice (merge disabled)", normalized.PreserveOn304)
	}
}

func TestDefaultKeyFuncUsesURL(t *testing.T) {
	t.Parallel()

	req := newGetRequest(t, "https://example.test/things?a=1")

	if got := defaultKeyFunc(req); got != "https://example.test/things?a=1" {
		t.Errorf("defaultKeyFunc = %q, want the request URL", got)
	}
}

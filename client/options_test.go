package etagclient

import (
	"slices"
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
	if normalized.FreshenOn304.kind != freshenPerRFC || normalized.FreshenOn304.fields != nil {
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

	switch {
	case normalized.FreshenOn304.kind != freshenNamedFields:
		t.Errorf("FreshenOn304 kind = %d, want named-fields mode preserved", normalized.FreshenOn304.kind)
	case !slices.Equal(normalized.FreshenOn304.fields, []string{"Date"}):
		t.Errorf("FreshenOn304 fields = %v, want [Date] preserved", normalized.FreshenOn304.fields)
	}
}

func TestDefaultKeyFuncUsesURL(t *testing.T) {
	t.Parallel()

	req := newGetRequest(t, "https://example.test/things?a=1")

	if got := defaultKeyFunc(req); got != "https://example.test/things?a=1" {
		t.Errorf("defaultKeyFunc = %q, want the request URL", got)
	}
}

func TestFreshenPolicyConstructors(t *testing.T) {
	t.Parallel()

	perRFC := FreshenPerRFC()
	if perRFC.kind != freshenPerRFC || perRFC.fields != nil {
		t.Errorf("FreshenPerRFC() = %+v, want the zero policy (RFC behavior)", perRFC)
	}

	none := FreshenNone()
	if none.kind != freshenNothing || none.fields != nil {
		t.Errorf("FreshenNone() = %+v, want the disabled policy", none)
	}

	fields := FreshenFields("A", "B")
	if fields.kind != freshenNamedFields {
		t.Errorf("FreshenFields kind = %d, want named-fields mode", fields.kind)
	}

	if !slices.Equal(fields.fields, []string{"A", "B"}) {
		t.Errorf("FreshenFields fields = %v, want [A B]", fields.fields)
	}

	// Naming zero fields means zero fields are freshened: the constructor
	// normalizes to FreshenNone instead of an empty named-fields mode.
	empty := FreshenFields()
	if empty.kind != freshenNothing || empty.fields != nil {
		t.Errorf("FreshenFields() = %+v, want the disabled policy (FreshenNone)", empty)
	}
}

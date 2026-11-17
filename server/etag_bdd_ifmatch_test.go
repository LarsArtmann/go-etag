package etag

import (
	"testing"
)

func TestSpec_RFC7232_IfMatch(t *testing.T) {
	t.Parallel()

	// RFC 7232 §3.1: If-Match uses strong comparison.
	// Applications use MatchesIfMatch in their handlers for unsafe methods.

	tag := NewETag("abc", Strong)

	t.Run("strong match", func(t *testing.T) {
		t.Parallel()

		if !MatchesIfMatch(tag, `"abc"`) {
			t.Error("expected strong match")
		}
	})

	t.Run("weak tag does not match", func(t *testing.T) {
		t.Parallel()

		if MatchesIfMatch(tag, `W/"abc"`) {
			t.Error("weak tag must not strongly match")
		}
	})

	t.Run("wildcard matches any existing resource", func(t *testing.T) {
		t.Parallel()

		if !MatchesIfMatch(tag, "*") {
			t.Error("wildcard should match")
		}
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()

		if MatchesIfMatch(tag, `"different"`) {
			t.Error("non-matching tag should not match")
		}
	})
}

func TestSpec_RFC7232_EntityTagFormat(t *testing.T) {
	t.Parallel()

	// RFC 7232 §2.3: entity-tag = [ weak ] opaque-tag
	// weak = %x57.2F (W/)
	// opaque-tag = DQUOTE *etagc DQUOTE

	t.Run("strong tag wire format", func(t *testing.T) {
		t.Parallel()

		got := NewETag("xyz", Strong).String()
		want := `"xyz"`

		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("weak tag wire format", func(t *testing.T) {
		t.Parallel()

		got := NewETag("xyz", Weak).String()
		want := `W/"xyz"`

		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("parse round-trips through String", func(t *testing.T) {
		t.Parallel()

		original := NewETag("round-trip-42", Weak)
		parsed, ok := ParseETag(original.String())
		if !ok {
			t.Fatal("ParseETag failed for round-trip")
		}

		if !parsed.WeakEqual(original) {
			t.Error("round-trip parse should produce equal tag")
		}

		if parsed.IsWeak() != original.IsWeak() {
			t.Error("round-trip parse should preserve strength")
		}
	})
}

package etagclient

import (
	"net/http"
	"testing"
)

// TestMergeHeaderPrefersExactThenCanonical pins the dual lookup: a literal
// non-canonical key in the source map wins, and a canonical name falls back
// to the source's canonical entry, so restricted FreshenFields lists work
// regardless of how the 304's header map was built.
func TestMergeHeaderPrefersExactThenCanonical(t *testing.T) {
	t.Parallel()

	t.Run("exact non-canonical key wins", func(t *testing.T) {
		t.Parallel()

		dst := stubHeader(headerPair{"X-A", "dst"})
		src := http.Header{"x-b": {"src"}}

		mergeHeader(dst, src, "x-b")

		// The merged value lands under the literal key, invisible to Get's
		// canonicalized lookup: the dual-key sharp edge for restricted
		// FreshenFields lists built from non-canonical names.
		//nolint:staticcheck // SA1008: the non-canonical key is the subject under test
		if got := dst["x-b"]; len(got) != 1 || got[0] != "src" {
			t.Errorf("dst[x-b] = %v, want [src] via the exact key", got)
		}
	})

	t.Run("canonical fallback for lowercase name", func(t *testing.T) {
		t.Parallel()

		dst := stubHeader(headerPair{"X-A", "dst"})
		src := stubHeader(headerPair{"Date", "fresh"})

		mergeHeader(dst, src, "date")

		if got := dst.Get("Date"); got != "fresh" {
			t.Errorf("Date = %q, want fresh via the canonical fallback", got)
		}
	})

	t.Run("absent name leaves destination untouched", func(t *testing.T) {
		t.Parallel()

		dst := stubHeader(headerPair{"Date", "stale"})
		src := stubHeader(headerPair{"ETag", `"v"`})

		mergeHeader(dst, src, "Age")

		if got := dst.Get("Age"); got != "" {
			t.Errorf("Age = %q, want empty (source has no Age)", got)
		}

		if got := dst.Get("Date"); got != "stale" {
			t.Errorf("Date = %q, want the stored value", got)
		}
	})
}

// TestIsUnsafeMethodTable pins the full safe-method set of RFC 9110 §9.2.1
// that §4.4 invalidation respects: GET, HEAD, OPTIONS, and TRACE are safe;
// everything else, including CONNECT and methods the spec does not know, is
// treated as unsafe.
func TestIsUnsafeMethodTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method string
		unsafe bool
	}{
		{http.MethodGet, false},
		{http.MethodHead, false},
		{http.MethodOptions, false},
		{http.MethodTrace, false},
		{http.MethodPost, true},
		{http.MethodPut, true},
		{http.MethodDelete, true},
		{http.MethodPatch, true},
		{http.MethodConnect, true},
		{"BREW", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			if got := isUnsafeMethod(tt.method); got != tt.unsafe {
				t.Errorf("isUnsafeMethod(%q) = %v, want %v", tt.method, got, tt.unsafe)
			}
		})
	}
}

// TestCloneHeaderNilReturnsEmpty pins cloneHeader's nil contract: a nil
// stored header must yield a usable empty header, not the nil map that
// http.Header.Clone returns — the first freshening write would panic. The
// branch is unreachable through current production paths (see cloneHeader's
// doc comment for the proof); this test keeps the defense-in-depth honest.
func TestCloneHeaderNilReturnsEmpty(t *testing.T) {
	t.Parallel()

	header := cloneHeader(nil)

	if header == nil {
		t.Fatalf("cloneHeader(nil) = nil, want an empty usable header")
	}

	if got := len(header); got != 0 {
		t.Errorf("cloneHeader(nil) has %d fields, want 0", got)
	}

	header.Set("X-Probe", "writable")

	if got := header.Get("X-Probe"); got != "writable" {
		t.Errorf("cloneHeader(nil) result is not writable: Get = %q", got)
	}
}

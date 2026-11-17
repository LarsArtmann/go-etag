package entitytag

import (
	"testing"
)

func TestETag_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  ETag
		want string
	}{
		{
			name: "strong",
			tag:  NewETag("abc123", Strong),
			want: `"abc123"`,
		},
		{
			name: "weak",
			tag:  NewETag("abc123", Weak),
			want: `W/"abc123"`,
		},
		{
			name: "empty opaque strong",
			tag:  NewETag("", Strong),
			want: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.tag.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestETag_OpaqueTag(t *testing.T) {
	t.Parallel()

	tag := NewETag("my-value", Strong)

	if got := tag.OpaqueTag(); got != "my-value" {
		t.Errorf("OpaqueTag() = %q, want %q", got, "my-value")
	}
}

func TestETag_IsWeak(t *testing.T) {
	t.Parallel()

	if NewETag("x", Strong).IsWeak() {
		t.Error("Strong tag reports IsWeak() = true")
	}

	if !NewETag("x", Weak).IsWeak() {
		t.Error("Weak tag reports IsWeak() = false")
	}
}

func TestETag_IsValid(t *testing.T) {
	t.Parallel()

	var empty ETag
	if empty.IsValid() {
		t.Error("zero-value ETag reports IsValid() = true")
	}

	if !NewETag("x", Strong).IsValid() {
		t.Error("ETag with opaque 'x' reports IsValid() = false")
	}
}

func TestStrength_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    Strength
		want bool
	}{
		{name: "Strong is valid", s: Strong, want: true},
		{name: "Weak is valid", s: Weak, want: true},
		{name: "out-of-range strength is invalid", s: Strength(42), want: false},
		{name: "negative strength is invalid", s: Strength(-1), want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.s.IsValid(); got != tc.want {
				t.Errorf("Strength(%d).IsValid() = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestETag_Comparison(t *testing.T) {
	t.Parallel()

	// RFC 7232 §2.3.2 comparison table: all four combinations of
	// strong/weak x matching/non-matching opaque-tags.
	tests := []struct {
		name   string
		a      ETag
		b      ETag
		strong bool
		weak   bool
	}{
		{name: "both weak same opaque", a: NewETag("1", Weak), b: NewETag("1", Weak), strong: false, weak: true},
		{name: "weak vs strong same opaque", a: NewETag("1", Weak), b: NewETag("1", Strong), strong: false, weak: true},
		{name: "both strong same opaque", a: NewETag("1", Strong), b: NewETag("1", Strong), strong: true, weak: true},
		{
			name:   "both strong different opaque",
			a:      NewETag("1", Strong),
			b:      NewETag("2", Strong),
			strong: false,
			weak:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.a.StrongEqual(tt.b); got != tt.strong {
				t.Errorf("StrongEqual() = %v, want %v", got, tt.strong)
			}

			if got := tt.a.WeakEqual(tt.b); got != tt.weak {
				t.Errorf("WeakEqual() = %v, want %v", got, tt.weak)
			}
		})
	}
}

func TestMatchesIfNoneMatch(t *testing.T) {
	t.Parallel()

	tag := NewETag("abc", Strong)

	tests := []struct {
		name      string
		headerVal string
		want      bool
	}{
		{name: "wildcard", headerVal: "*", want: true},
		{name: "exact match", headerVal: `"abc"`, want: true},
		{name: "weak match", headerVal: `W/"abc"`, want: true},
		{name: "list contains match", headerVal: `"other", "abc", "third"`, want: true},
		{name: "list contains weak match", headerVal: `"other", W/"abc"`, want: true},
		{name: "no match", headerVal: `"different"`, want: false},
		{name: "empty header", headerVal: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchesIfNoneMatch(tag, tt.headerVal); got != tt.want {
				t.Errorf("MatchesIfNoneMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesIfMatch(t *testing.T) {
	t.Parallel()

	tag := NewETag("abc", Strong)

	tests := []struct {
		name      string
		headerVal string
		want      bool
	}{
		{name: "wildcard", headerVal: "*", want: true},
		{name: "strong match", headerVal: `"abc"`, want: true},
		{name: "list contains strong match", headerVal: `"other", "abc"`, want: true},
		{name: "weak tag does not strongly match", headerVal: `W/"abc"`, want: false},
		{name: "no match", headerVal: `"different"`, want: false},
		{name: "empty header", headerVal: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchesIfMatch(tag, tt.headerVal); got != tt.want {
				t.Errorf("MatchesIfMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesIfMatch_WeakServerTag(t *testing.T) {
	t.Parallel()

	// A weak server tag can never strongly match any If-Match value.
	weakTag := NewETag("abc", Weak)

	if MatchesIfMatch(weakTag, `"abc"`) {
		t.Error("weak server tag should not strongly match")
	}
}

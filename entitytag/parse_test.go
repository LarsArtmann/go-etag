package entitytag

import (
	"testing"
)

func TestParseETag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantOk     bool
		wantOpaque string
		wantWeak   bool
	}{
		{name: "strong tag", input: `"abc"`, wantOk: true, wantOpaque: "abc", wantWeak: false},
		{name: "weak tag", input: `W/"abc"`, wantOk: true, wantOpaque: "abc", wantWeak: true},
		{name: "with whitespace", input: `  "abc"  `, wantOk: true, wantOpaque: "abc", wantWeak: false},
		{name: "no quotes", input: `abc`, wantOk: false},
		{name: "empty quotes", input: `""`, wantOk: false},
		{name: "only opening quote", input: `"abc`, wantOk: false},
		{name: "only closing quote", input: `abc"`, wantOk: false},
		{name: "empty string", input: ``, wantOk: false},
		{name: "W/ only", input: `W/`, wantOk: false},
		// RFC 7232 §2.3: weak = %x57.2F ; "W/", case-sensitive.
		{name: "lowercase weak prefix", input: `w/"abc"`, wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag, ok := ParseETag(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("ParseETag(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}

			if !ok {
				return
			}

			if tag.OpaqueTag() != tt.wantOpaque {
				t.Errorf("opaque = %q, want %q", tag.OpaqueTag(), tt.wantOpaque)
			}

			if tag.IsWeak() != tt.wantWeak {
				t.Errorf("weak = %v, want %v", tag.IsWeak(), tt.wantWeak)
			}
		})
	}
}

func TestParseETagList(t *testing.T) {
	t.Parallel()

	tags := ParseETagList(`"a,b", W/"c", "d"`)
	if len(tags) != 3 {
		t.Fatalf("len(tags) = %d, want 3", len(tags))
	}

	if tags[0].OpaqueTag() != "a,b" {
		t.Errorf("tags[0] opaque = %q, want %q", tags[0].OpaqueTag(), "a,b")
	}

	if !tags[1].IsWeak() {
		t.Error("tags[1] should be weak")
	}

	if tags[1].OpaqueTag() != "c" {
		t.Errorf("tags[1] opaque = %q, want %q", tags[1].OpaqueTag(), "c")
	}

	if tags[2].OpaqueTag() != "d" {
		t.Errorf("tags[2] opaque = %q, want %q", tags[2].OpaqueTag(), "d")
	}
}

func TestParseETagList_SkipsMalformed(t *testing.T) {
	t.Parallel()

	tags := ParseETagList(`"valid", not-a-tag, "also-valid"`)
	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2 (malformed tag skipped)", len(tags))
	}
}

func TestParseETagList_Empty(t *testing.T) {
	t.Parallel()

	tags := ParseETagList("")
	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

func TestSplitRawETags_EscapedQuotes(t *testing.T) {
	t.Parallel()

	tags := splitRawETags(`"a\"b", "c"`)

	if len(tags) != 2 {
		t.Fatalf("len(tags) = %d, want 2", len(tags))
	}

	want0 := `"a\"b"`
	if tags[0] != want0 {
		t.Errorf("tags[0] = %q, want %q", tags[0], want0)
	}

	want1 := `"c"`
	if tags[1] != want1 {
		t.Errorf("tags[1] = %q, want %q", tags[1], want1)
	}
}

func TestSplitRawETags_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		count int
		want0 string
	}{
		{name: "trailing comma", input: `"a", "b",`, count: 2, want0: `"a"`},
		{name: "leading comma", input: `, "a", "b"`, count: 2, want0: `"a"`},
		{name: "empty entries", input: `"a",, "b"`, count: 2, want0: `"a"`},
		{name: "whitespace between commas", input: `"a",   , "b"`, count: 2, want0: `"a"`},
		{name: "single tag no comma", input: `"only"`, count: 1, want0: `"only"`},
		{name: "only commas", input: `,,,`, count: 0},
		{name: "only whitespace", input: `   `, count: 0},
		{name: "comma in opaque tag", input: `"a,b,c"`, count: 1, want0: `"a,b,c"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tags := splitRawETags(tt.input)
			if len(tags) != tt.count {
				t.Fatalf("len(tags) = %d, want %d (input=%q)", len(tags), tt.count, tt.input)
			}

			if tt.count > 0 && tags[0] != tt.want0 {
				t.Errorf("tags[0] = %q, want %q (input=%q)", tags[0], tt.want0, tt.input)
			}
		})
	}
}

func TestParseETagList_AllMalformed(t *testing.T) {
	t.Parallel()

	tags := ParseETagList("garbage, also-not-a-tag, 12345")
	if tags == nil {
		t.Fatal("ParseETagList returned nil for all-malformed input")
	}

	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0 for all-malformed input", len(tags))
	}
}

func TestParseETagList_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	tags := ParseETagList("   ")
	if tags == nil {
		t.Fatal("ParseETagList returned nil for whitespace-only input")
	}

	if len(tags) != 0 {
		t.Errorf("len(tags) = %d, want 0", len(tags))
	}
}

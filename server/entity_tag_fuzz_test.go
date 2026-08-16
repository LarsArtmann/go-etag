package etag

import "testing"

// FuzzParseETag verifies that ParseETag never panics on arbitrary input and
// that every successfully parsed tag round-trips: re-parsing its String()
// output produces an identical ETag.
func FuzzParseETag(f *testing.F) {
	f.Add(`"abc"`)
	f.Add(`W/"abc"`)
	f.Add("abc")
	f.Add("")
	f.Add(`""`)
	f.Add(`"a,b"`)
	f.Add(`"a\"b"`)
	f.Add("*")
	f.Add(`  "abc"  `)
	f.Add(`"W/abc"`)
	f.Add(`W/W/"abc"`)

	f.Fuzz(func(t *testing.T, raw string) {
		tag, ok := ParseETag(raw)

		if !ok {
			if tag.IsValid() {
				t.Fatalf("ParseETag(%q) returned ok=false but tag is valid: %v", raw, tag)
			}

			return
		}

		if !tag.IsValid() {
			t.Fatalf("ParseETag(%q) returned ok=true but tag is not valid", raw)
		}

		reparsed, reparseOK := ParseETag(tag.String())
		if !reparseOK {
			t.Fatalf("ParseETag(%q).String() = %q failed to re-parse", raw, tag.String())
		}

		if reparsed != tag {
			t.Fatalf("round-trip mismatch: %v != %v", reparsed, tag)
		}
	})
}

// FuzzParseETagList verifies that ParseETagList never panics on arbitrary
// input, always returns a non-nil slice, and every element is a valid ETag
// that round-trips through String().
func FuzzParseETagList(f *testing.F) {
	f.Add(`"abc"`)
	f.Add(`"abc", "def"`)
	f.Add(`"abc", W/"def"`)
	f.Add("*")
	f.Add("")
	f.Add(`"abc",, "def"`)
	f.Add(`"a,b"`)
	f.Add(`"a\"b", "c"`)
	f.Add(`W/"x", "y", W/"z"`)
	f.Add(`"unterminated`)

	f.Fuzz(func(t *testing.T, header string) {
		tags := ParseETagList(header)

		if tags == nil {
			t.Fatalf("ParseETagList(%q) returned nil; expected non-nil slice", header)
		}

		for i, tag := range tags {
			if !tag.IsValid() {
				t.Fatalf("ParseETagList(%q)[%d] is not a valid ETag: %v", header, i, tag)
			}

			reparsed, ok := ParseETag(tag.String())
			if !ok || reparsed != tag {
				t.Fatalf("ParseETagList(%q)[%d] round-trip failed: %v", header, i, tag)
			}
		}
	})
}

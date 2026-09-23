package entitytag

import "fmt"

// ExampleParseETag round-trips the RFC 7232 §2.3 wire format: a parsed
// strong tag prints back identically, a weak tag keeps its W/ prefix, and
// a value that is not a well-formed entity-tag refuses to parse instead
// of being guessed at.
func ExampleParseETag() {
	strong, ok := ParseETag(`"v1"`)
	fmt.Println("strong:", strong.String(), ok)

	weak, weakOK := ParseETag(`W/"v1"`)
	fmt.Println("weak:", weak.String(), weakOK)

	_, unquotedOK := ParseETag(`v1`)
	fmt.Println("unquoted:", unquotedOK)

	// Output:
	// strong: "v1" true
	// weak: W/"v1" true
	// unquoted: false
}

// ExampleETag_weakVsStrong contrasts the two RFC 7232 §2.3.2 comparison
// functions: weak comparison ignores strength, so W/"v1" and "v1" are
// equal; strong comparison additionally requires both tags to be strong,
// so the same pair is not equal while two strong tags with one opaque
// value are.
func ExampleETag_weakVsStrong() {
	strong := NewETag("v1", Strong)
	weak := NewETag("v1", Weak)

	fmt.Println("weak vs strong, WeakEqual:", weak.WeakEqual(strong))
	fmt.Println("weak vs strong, StrongEqual:", weak.StrongEqual(strong))

	strongAgain := NewETag("v1", Strong)
	fmt.Println("strong vs strong, StrongEqual:", strong.StrongEqual(strongAgain))

	// Output:
	// weak vs strong, WeakEqual: true
	// weak vs strong, StrongEqual: false
	// strong vs strong, StrongEqual: true
}

// ExampleParseETagList parses an If-None-Match list holding a strong tag,
// a weak tag, and a trailing entry, preserving order and strength. The
// wildcard is not an entity-tag and parses to nothing; callers match it
// against the raw header string themselves.
func ExampleParseETagList() {
	for _, tag := range ParseETagList(`"a", W/"b", "c"`) {
		fmt.Println(tag.String())
	}

	fmt.Println("wildcard tags:", len(ParseETagList(`*`)))

	// Output:
	// "a"
	// W/"b"
	// "c"
	// wildcard tags: 0
}

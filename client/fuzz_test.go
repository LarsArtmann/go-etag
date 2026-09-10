package etagclient

import (
	"net/http"
	"strings"
	"testing"
)

// FuzzHasNoStoreDirective verifies that Cache-Control parsing never panics
// on arbitrary input and stays sound: any directive the quote-aware splitter
// extracts whose name is literally no-store must be detected by
// hasNoStoreDirective, so a parser regression can never silently start
// storing no-store responses (RFC 9111 §3, §5.2.2.5).
func FuzzHasNoStoreDirective(f *testing.F) {
	f.Add("no-store")
	f.Add("max-age=60")
	f.Add("no-cache, max-age=0, must-revalidate")
	f.Add(`private="a\"b, no-store"`)
	f.Add(`"no-store"`)
	f.Add("NO-STORE")
	f.Add("no-store=")
	f.Add("no-store, no-store")
	f.Add(`"unterminated`)
	f.Add("")
	f.Add(`max-age="60, no-store"`)
	f.Add("  no-store  ,max-age=0")

	f.Fuzz(func(t *testing.T, value string) {
		header := http.Header{}
		header.Set("Cache-Control", value)

		for _, directive := range cacheControlDirectives(value) {
			name, _, _ := strings.Cut(directive, "=")
			if !strings.EqualFold(strings.TrimSpace(name), "no-store") {
				continue
			}

			if !hasNoStoreDirective(header) {
				t.Fatalf(
					"hasNoStoreDirective(%q) = false, but %q is a literal no-store directive",
					value,
					directive,
				)
			}
		}
	})
}

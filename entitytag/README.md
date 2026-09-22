# entitytag — the RFC 7232 entity-tag domain type (package `entitytag`)

The shared domain type at the heart of [go-etag](../README.md): an opaque
validator string plus an explicit `Strength` (`Strong`/`Weak`), with the ABNF
parser and both §2.3.2 comparison functions. Extracted from `server/` on
2026-09-18 so the [server middleware](../server/README.md) and the
[client cache](../client/README.md) share one typed validator — dependency
direction `server → entitytag ← client`, with no `client → server` edge.

```go
import "github.com/larsartmann/go-etag/entitytag"

tag, ok := entitytag.ParseETag(`W/"abc123"`)
tags := entitytag.ParseETagList(`"a", W/"b", "c"`)

strong := entitytag.NewETag("abc123", entitytag.Strong)
strong.WeakEqual(tag)  // true  — weak comparison ignores strength
strong.StrongEqual(tag) // false — strong requires both to be strong
```

Surface:

- `ETag` with `String()` (wire format), `OpaqueTag()`, `IsWeak()`,
  `IsValid()`.
- `ParseETag` / `ParseETagList` — quote-aware, escaped-quote-safe parsing.
- `MatchesIfNoneMatch` (weak comparison, §3.2) and `MatchesIfMatch` (strong
  comparison, §3.1) helpers for applications evaluating conditional requests
  by hand.
- `Strength.IsValid`.

Values that do not parse as an RFC 7232 §2.3 entity-tag (unclosed quote,
bare `*`, lowercase `w/`) never match anything — the typed parser makes that
bug class unrepresentable.

The `server` package re-exports this entire surface (`etag.ETag`,
`etag.ParseETag`, …), so existing `go-etag/server` consumers need no import
change. Full API:
[pkg.go.dev/github.com/larsartmann/go-etag/entitytag](https://pkg.go.dev/github.com/larsartmann/go-etag/entitytag).

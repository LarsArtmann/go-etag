# go-etag — Filed Items from the httputil Consumer Review (2026-08-29)

Filed by the httputil session of 2026-08-29 (plan tasks T25 + T29) after a scratch-module consumer verification.

## Consumer verification result

An external module consuming go-etag via `replace` (and, by construction, via the module proxy) built and served correctly:

- `etag.New(etag.DefaultETagConfig())` composes as `Middleware` into a stdlib chain.
- First GET: 200 with `ETag` header present.
- Ergonomics note: `New(cfg ETagConfig)` does not accept nil; `DefaultETagConfig()` is the zero-thought entry point. Consider documenting that `New(DefaultETagConfig())` is the blessed minimal usage.

## Suggested v0.1.1 items (small, non-breaking)

1. ~~README: document `New(DefaultETagConfig())` as the minimal usage example (found missing during consumer verification).~~ done — the server quick start (README since `b9eb32e`, carried through `bc5a551`) is exactly `etag.New(etag.DefaultETagConfig())(mux)`
2. README/cache-policy guidance: a 304 response preserves ETag; callers own `Cache-Control`/`Vary`. Cross-link the guidance now living in httputil's README (added 2026-08-29).
3. Ordering note for chains: ETag must see the FINAL representation — place it inside (after) compression; document the recommended Chain position (httputil README carries the equivalent guidance for the deprecated adapter path).
4. ~~Compliance-suite ownership: confirm whether the RFC 7232 compliance suite lives in this repo (httputil's 23-33 report moved items 10-17 here) and note that in `docs/`.~~ done — server suite: `server/etag_bdd_test.go`; client suite: `client/spec_test.go` (RFC 9111) at `9204885`; both noted in AGENTS.md Testing Conventions
5. ~~`deprecated.go`: confirm the `Deprecated:` doc comments carry `// Deprecated:` prefixes so gopals/staticcheck flag call sites.~~ done — verified: 20 `// Deprecated:` prefixes present

_**Status (2026-09-10 docs-health pass):** items 2-3 remain open and are tracked as TODO_LIST.md #9._

## Non-goals carried over

- If-Match / Last-Modified / If-Range helpers remain post-v1.0 scope for go-etag (httputil ROADMAP defers conditional-request scope decisions here).

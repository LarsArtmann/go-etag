# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Typed error-code surface in the server package (mirrors httputil): `Code` with six family constructors (`Rejection`, `Conflict`, `Transient`, `Corruption`, `Infrastructure`, `Orchestration`) plus matching `Wrap*` methods, a `Domain` type, and `DomainOf`/`InDomain` helpers so consumers can route classified errors by failing component without string parsing.
- `ExampleFreshenPolicy`: a GoDoc example showing `FreshenFields` restricting 304 freshening to `Retry-After` while the stored `Date` survives and the validator flows through.
- `ExampleETagConfig_Validate` (server): a GoDoc example pinning the sentinel contract — `Validate` returns a fresh context-bearing error that `errors.Is` matches against `ErrInvalidConfig` by code and family, never the sentinel instance itself.

### Changed

- CI: the `GOTOOLCHAIN` toolchain pin moved from workflow level to the test, lint, and fuzz jobs; the govulncheck job is deliberately unpinned so its `govulncheck@latest` install can fetch a newer Go toolchain instead of failing.
- Minimum Go is now 1.27.1 (go.mod directive) and `go-error-family` is bumped to v0.10.1; the test, lint, and fuzz CI pins were synced to `go1.27.1` to match. Consumers on older toolchains must upgrade Go to consume this version.
- `docs/rfc9111-conformance.md`: a new "Interpretation decisions" section records the RFC judgment calls (304-`no-store` update ≠ storage, `no-store` HEAD neutrality, unprovable HEAD identity means stale, `FreshenOn304` is 304-scoped).
- `ErrInvalidConfig` (server package and the deprecated root shim) is declared as the `error` interface instead of `*errorfamily.Error` (erraudit's sentinel guard rejects concrete-typed sentinels). `errors.Is(err, ErrInvalidConfig)` matching is unchanged — matching is by code and family, not identity — and `Validate` still returns context-bearing classified errors derived from the same code and message. Consumers assigning the sentinel to a `*errorfamily.Error` variable must switch to the interface type or `errors.AsType`.

### Fixed

- `FreshenFields()` with no arguments now returns the `FreshenNone` policy instead of a named-fields policy with an empty list (identical behavior, honest representation; documented on the constructor).

## [0.3.1] - 2026-09-11

### Changed

- CI: the pinned GitHub Actions group (checkout, setup-go, upload-artifact) was bumped via Dependabot PR #1, clearing the Node 20 deprecation warnings; the merged run is green. Docs and CI only — `go.mod`, `go.sum`, and every `.go` file are byte-identical to v0.3.0.

## [0.3.0] - 2026-09-11

### Added

- `client/spec_test.go`: an RFC 9111-grounded test suite for the client transport, motivated by a field report of a CDN serving a two-day-stale 200 (Age: 137882) whose ETag faithfully described the stale entity. Pins Age surfacing, §4.3.4 freshening, stored-validator persistence, the no-store storage ban (§3), §4.4 invalidation, HEAD bypass, and caller-owned `If-None-Match`.
- RFC 9111 §4.4 conformance: a non-error (2xx/3xx) response to an unsafe request method (anything but GET/HEAD/OPTIONS/TRACE) now invalidates the stored entry for that URI, so a mutation cannot leave a pre-mutation body waiting to be rebuilt.
- `client/integration_test.go`: a real `httptest.Server` + `http.Client` round trip verifying canonical header forms, the bodiless 304, and Age freshening against production net/http rather than stubs.
- Five unpinned RFC statements now have conformance tests: the 304's mandatory 200 metadata (RFC 7232 §4.1), the case-sensitive `W/` weak prefix (RFC 7232 §2.3), §4.4 invalidation scoped to the mutated URI only, the never-store rule for validator-less 200s, and weak-comparison adoption of a 304's `W/`-marked validator (RFC 9110 §8.8.3.2).
- `deprecated_test.go`: an export-parity suite for the deprecated root shim — compile-time type-identity assertions plus behavior smoke tests through every re-exported wrapper, so alias drift breaks the build instead of silently splitting the domain (root package coverage 0% → 100%).
- Client branch-coverage specs: `freshen` skip-guard on concurrent replacement, the store path when a body refuses to Close, `weaklyMatchesValidator` edge table, `mergeHeader` exact-key/canonical-fallback duality, and passthrough of a contract-violating `(nil, nil)` from `next` (client coverage 94.3% → 97.7%).
- Client spec pin-ups: only a 200 with a validator is stored (201/206/304/500 pass through), a `no-store`-carrying 304 still freshens and rebuilds, the exhaustive safe/unsafe method table, additive freshening of fields the stored response lacked, and Age monotonicity across repeated revalidations.
- `client/fuzz_test.go`: `FuzzHasNoStoreDirective` fuzzes the untrusted `Cache-Control` parser with a soundness property (a literal top-level `no-store` directive is always detected) and a CI fuzz job mirroring the server's.
- `server/integration_test.go`: real-wire server tests pinning Content-Length framing on 200s, HEAD body suppression with advertised length, and a bodiless 304 without Content-Length over TCP.
- `TestIntegrationUnsafeMethodInvalidatesThroughRealServer`: §4.4 invalidation verified through a real server — a 204 PUT forces the next GET to refetch unconditionally.
- `Example_ageAwareStalenessCheck`: documents rejecting edge-served stale entities via the surfaced Age header.
- RFC 9111 §4.3.5 HEAD-based freshening: a HEAD 200 whose validators match the stored entry (ETag weakly, Last-Modified exactly) and whose Content-Length matches the stored body updates the stored metadata via the §3.2 update rules; any mismatch — or an incomparable validator — marks the entry stale so the next GET refetches; a `no-store` HEAD does neither. HEAD remains never conditional and never rebuilt. Pinned by a table-driven spec suite and a real-server integration test.
- Request-side `Cache-Control: no-store` handling (RFC 9111 §5.2.2.5 request directive): a `no-store` GET bypasses the cache entirely (no validator injection, no rebuild, no storage), a `no-store` HEAD skips freshening, and unsafe methods still invalidate (deletion stores nothing).
- `docs/rfc9111-conformance.md`: a requirement-by-requirement RFC 9111 conformance table (done / documented deviation / not-applicable) citing the pinning test for every row, linked from the README compliance section.
- Benchmark baselines under `reports/bench/` (`-benchmem -count=6`): before/after the lazy-key change and a full post-§4.3.5 baseline, replacing smoke runs.

### Changed

- **BREAKING (pre-1.0):** `Options.PreserveOn304 []string` is replaced by `Options.FreshenOn304 FreshenPolicy` with the constructors `FreshenPerRFC()` (the zero value, selecting the §4.3.4 default), `FreshenFields("X", …)` (restrict freshening to named fields), and `FreshenNone()` (disable it). The nil-vs-empty-slice overload is gone and the name no longer lies about what the option governs.
- Cache keys are now derived lazily: `KeyFunc` runs only for GET lookups and for non-GET responses that actually trigger invalidation, so plain non-GET passthrough never pays key derivation (~19% faster, one fewer allocation per passthrough round trip).
- CI hardening: `GOTOOLCHAIN: go1.26.7` env pin (the setup-go manifest can lag go.dev; keep in sync with go.mod), a pinned `govulncheck` job (`golang/govulncheck-action` v1.1.0), and golangci-lint bumped v2.12.2 → v2.13.2 in CI to match the local toolchain.
- Linting migrated from the deprecated `exhaustruct` to `exhaustruct_v5` (deprecation-free runs; v5 settings use `ignore-patterns` regexes, and `//nolint:exhaustruct_v5` directives are required on intentional zero values).
- README: client options updated for `FreshenOn304`; a new "Cache policy: who owns which headers" section (the transport owns validators; callers own `Cache-Control`/`Vary`; place the ETag middleware beneath compression so tags describe the final representation); HEAD-freshening and request-`no-store` bullets.
- `CONTRIBUTING.md` rewritten for the current workflow: commands, lint constraints, testing conventions, benchmark discipline, and the CI job description.
- `MaxBodyBytes` semantics documented: the cap counts decoded bytes when net/http transparently decompresses the body (`client/options.go`).
- Age-omission semantics documented in `client/doc.go`: a 304 without `Age` freezes the last known age instead of resetting it.
- Ecosystem sweep (GitHub code search): every known consumer remains in-house (six LarsArtmann repositories); no external consumers found.
- A revalidation 304 now freshens the stored entry itself (RFC 9111 §4.3.4), so the validator the 304 returns replaces the stored one for subsequent requests, and freshened metadata persists across rebuilds.
- Rebuilt responses now carry `Uncompressed` when the stored body is the form net/http transparently decoded, so downstream consumers see honest body metadata.

### Fixed

- Stale `Age` no longer survives 304 rebuilds: a 304 reporting a grown Age updates the rebuilt response, so an edge cache's age cannot run backwards through revalidation (the field report's masked symptom).
- A 304 declaring a validator that does not weak-match the one it just validated (a broken server's claim of a different representation) no longer replaces the stored validator or the rebuilt response's ETag, per the RFC 9111 §4.3.4 validator filtering.
- Responses carrying `Cache-Control: no-store` are no longer stored (RFC 9111 §3); their bodies still stream through intact.
- A caller-supplied `If-None-Match` is no longer clobbered by the stored validator; a 304 answering the caller's own conditional passes through instead of being rebuilt from the transport's entry.
- Hop-by-hop fields (Connection and its listed fields, Keep-Alive, Proxy-*) are stripped from stored responses per RFC 9111 §3.1, so they cannot be resurrected by later rebuilds.

## [0.2.0] - 2026-08-16

### Added

- `client` package (import `github.com/larsartmann/go-etag/client`, package name `etagclient`): a client-side conditional GET cache as an `http.RoundTripper`. `NewTransport(next, opts)` replays stored validators as `If-None-Match` and rebuilds 304 responses into the cached 200 the caller expects.
- `etagclient.Options`: `KeyFunc` (credential scoping hook; defaults to the request URL), `MaxEntries` (FIFO bound, default 256), `MaxBodyBytes` (cached-body cap, default 1 MiB; oversized responses pass through uncached), `PreserveOn304` (headers merged fresh from the 304 onto the rebuilt 200, default `["Date"]` per RFC 7232 §4.1), and `FromCacheHeader` (optional from-cache marker).
- `etagclient.Transport.Stats()` returning `Stats{Hits, Stored, Entries}` for cache telemetry.
- GoDoc example `ExampleNewTransport` demonstrating a full store-then-revalidate round trip.
- `ETagConfig.OnETagGenerated func(ETag)`: observability hook firing each time the middleware computes and sets a new entity-tag from the buffered body. Does not fire for handler-provided tags adopted via `SkipIfPresent`. Pairs with `On304` to derive cache hit ratios.
- `ETagConfig.On304 func(ETag)`: observability hook firing after a 304 Not Modified is committed, with the matching entity-tag regardless of tag source. Fires in addition to `OnETagGenerated` (which fires first).
- `ETagConfig.OnBufferOverflow func(int)`: observability hook firing at most once per response when the body exceeds `MaxBufferSize` and the response degrades to streaming without an ETag. The argument is the exceeded limit. Handler-initiated `Flush()` does not fire it.
- GoDoc example `ExampleNew_observabilityHooks` deriving a cache hit ratio from the new hooks.

### Changed

- The server middleware moved to `github.com/larsartmann/go-etag/server` (package name stays `etag`, so migration is a pure import-path swap with zero call-site changes). The root package `github.com/larsartmann/go-etag` is now a deprecated alias shim for the server package: every v0.1.x identifier remains available unchanged, with `// Deprecated:` markers nudging consumers toward the new import path. The shim will be removed in v1.0.0.

## [0.1.1] - 2026-08-14

### Changed

- `Middleware` is now a type alias (`type Middleware = func(http.Handler) http.Handler`) instead of a distinct named type, making it fully interchangeable with the standard middleware signature without an explicit conversion step. Backwards compatible — existing code compiles unchanged.

### Fixed

- Replaced the proprietary LICENSE with the MIT License.

## [0.1.0] - 2026-08-07

### Added

- RFC 7232 HTTP ETag middleware that buffers GET/HEAD response bodies, computes entity-tags, and handles `If-None-Match` conditional requests with `304 Not Modified` responses.
- Dedicated `ETag` value type with `Strength` enum (`Strong`/`Weak`) encoding RFC 7232 §2.1 validator semantics directly in the type system.
- `StrongEqual` and `WeakEqual` methods implementing both RFC 7232 §2.3.2 comparison functions.
- `ParseETag` and `ParseETagList` for parsing entity-tags from wire format (quote-aware, handles escaped quotes and commas inside quotes).
- `MatchesIfNoneMatch` (weak comparison) and `MatchesIfMatch` (strong comparison) exported helpers for manual conditional-request evaluation.
- `SkipIfPresent` config field to respect handler-set ETags instead of overwriting them.
- `Skip` config predicate to exclude specific routes from ETag processing (SSE, large downloads, streaming).
- `HashFunc` accepts `func([]byte) string` for arbitrary opaque-tag values (not just uint64 hashes); defaults to FNV-64a.
- RFC 7230 §3.3 compliance: HEAD responses set `Content-Length` without sending a body.
- RFC 7232 §4.1 compliance: 304 responses strip `Content-Length`.
- Classified errors via `go-error-family` for retry-aware observability with `OnError` callback hook.
- BDD-style spec suite (`etag_bdd_test.go`) mapping RFC 7232 sections to behavioral expectations.
- Fuzz tests for `ParseETag` and `ParseETagList` verifying no panics and round-trip integrity on arbitrary input.

[Unreleased]: https://github.com/Larsartmann/go-etag/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/Larsartmann/go-etag/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/Larsartmann/go-etag/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Larsartmann/go-etag/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/larsartmann/go-etag/releases/tag/v0.1.1
[0.1.0]: https://github.com/larsartmann/go-etag/releases/tag/v0.1.0

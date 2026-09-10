# Features

Honest inventory of what exists, by status. Evidence cites code; every
FULLY_FUNCTIONAL row is exercised by the test suite (verified 2026-09-10:
`go test -race ./...` green, server 99.0% / client 94.3% coverage).

Statuses: FULLY_FUNCTIONAL, PARTIALLY_FUNCTIONAL, BROKEN, PLANNED.

## Server middleware (`server/`, package `etag`)

| Feature                                                          | Status             | Evidence / Notes                                                                                       |
| ---------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------ |
| Automatic ETag generation from response bodies (FNV-64a default) | FULLY_FUNCTIONAL   | `server/etag.go` (`New`, `computeETag`); spec + unit tests                                            |
| `If-None-Match` → 304 Not Modified (weak comparison, 2xx only)   | FULLY_FUNCTIONAL   | `server/etag.go` (`shouldReturnNotModified`); `etag_bdd_test.go` RFC 7232 §3.2 specs                   |
| HEAD compliance — Content-Length set, no body sent               | FULLY_FUNCTIONAL   | RFC 7230 §3.3; `TestNew_HeadRequest_NoBody`, `TestSpec_RFC7232_HeadRequest`                            |
| 304 strips Content-Length, keeps ETag                            | FULLY_FUNCTIONAL   | RFC 7232 §4.1; `TestNew_304_ExcludesContentLength`                                                     |
| Handler-set ETag adoption (`SkipIfPresent`)                      | FULLY_FUNCTIONAL   | `server/etag.go` (`resolveETag`); 304-passthrough + fallback tests                                     |
| Per-route opt-out (`Skip` predicate)                             | FULLY_FUNCTIONAL   | `server/etag.go`; true/false path tests                                                                |
| Custom hash functions (`HashFunc func([]byte) string`)           | FULLY_FUNCTIONAL   | `TestNew_CustomHashFunc` (+ body-bytes test)                                                           |
| Buffer overflow → stream without ETag (`MaxBufferSize`)          | FULLY_FUNCTIONAL   | `TestNew_MemoryLimit_DisablesETag`, overflow write-error tests                                         |
| Hijack/Flush → streaming mode                                    | FULLY_FUNCTIONAL   | `server/wrapper.go`; hijack/flush delegate tests, streaming write tests                                |
| Observability hooks (`OnETagGenerated` / `On304` / `OnBufferOverflow` / `OnError`) | FULLY_FUNCTIONAL | v0.2.0; exactly-once/ordering/nil-safety specs; no telemetry dependency |
| `ETag` domain type — parse, strength, strong/weak comparison     | FULLY_FUNCTIONAL   | `server/entity_tag.go`; BDD specs + fuzz round-trip (`entity_tag_fuzz_test.go`)                         |
| Conditional-request helpers (`MatchesIfNoneMatch`, `MatchesIfMatch`) | FULLY_FUNCTIONAL | RFC 7232 §3.1/§3.2; lost-update example in README                                                      |
| Classified errors (`go-error-family`, 5 codes + sentinel)        | FULLY_FUNCTIONAL   | `server/errors.go`; `errors_test.go` (idempotent registration, templates)                              |

## Client transport (`client/`, package `etagclient`)

| Feature                                                              | Status             | Evidence / Notes                                                                          |
| -------------------------------------------------------------------- | ------------------ | ----------------------------------------------------------------------------------------- |
| Conditional GET transport — replay stored validators as `If-None-Match` | FULLY_FUNCTIONAL | `client/transport.go` (`RoundTrip`); `transport_test.go`                                  |
| 304 → 200 rebuild from cache (callers see no difference)             | FULLY_FUNCTIONAL   | `rebuildFromCache`; `TestRoundTripRebuilds304FromCache` + GoDoc example                    |
| RFC 9111 §4.3.4 freshening (default: every 304 field replaces stored; restricted / disabled modes) | FULLY_FUNCTIONAL | `freshenedHeader`; `spec_test.go` freshening specs — unreleased, sits in CHANGELOG `[Unreleased]` |
| §4.4 unsafe-method invalidation (2xx/3xx to unsafe method drops entry) | FULLY_FUNCTIONAL  | `responseCache.invalidate`, `roundTripUnsafe`; `TestSpecUnsafeMethodInvalidatesEntry`      |
| `Cache-Control: no-store` storage ban (quote-aware, case-insensitive) | FULLY_FUNCTIONAL  | `hasNoStoreDirective`; `TestSpecNoStoreResponseIsNeverCached` incl. quoted-arg subtests    |
| Caller-supplied `If-None-Match` is never clobbered                   | FULLY_FUNCTIONAL   | `TestSpecCallerSuppliedIfNoneMatchIsHonored`                                               |
| Hop-by-hop field stripping at store time (§3.1)                     | FULLY_FUNCTIONAL   | `TestSpecStoredHeaderShedsHopByHopFields`                                                  |
| Age surfaced verbatim on 200s; freshened from 304s (never backwards) | FULLY_FUNCTIONAL   | `TestSpecStale200SurfacesAgeToCaller`; motivated by a real CDN field report                |
| Mismatched 304 validators not adopted (§4.3.4 filtering)            | FULLY_FUNCTIONAL   | `restoreMismatchedValidator`; `TestSpecMismatched304ValidatorIsNotAdopted`                 |
| Bounded cache — FIFO eviction (`MaxEntries`), body-size skip (`MaxBodyBytes`) | FULLY_FUNCTIONAL | `client/cache.go`; eviction + boundary tests                                       |
| Credential-scoped cache keys (`KeyFunc` hook + loud doc warning)    | FULLY_FUNCTIONAL   | `TestRoundTripSeparatesCredentialsViaKeyFunc`, `client/doc.go` warning                     |
| From-cache marker header (`FromCacheHeader`)                        | FULLY_FUNCTIONAL   | set/absent/stale-strip tests                                                               |
| Cache telemetry (`Stats{Hits, Stored, Entries}`)                    | FULLY_FUNCTIONAL   | `TestResponseCacheStats`                                                                   |
| Real-wire verification via `httptest.Server` + real `http.Client`   | FULLY_FUNCTIONAL   | `client/integration_test.go` (canonical forms, bodiless 304, Age flow)                     |

## Root package (deprecated shim)

| Feature                                                        | Status             | Evidence / Notes                                                          |
| -------------------------------------------------------------- | ------------------ | -------------------------------------------------------------------------- |
| v0.1.x compatibility shim — full alias surface, `// Deprecated:` | FULLY_FUNCTIONAL  | `deprecated.go` (4 types, 7 consts, 1 var, 8 func wrappers); compile-verified against a scratch v0.1.1-style consumer at v0.2.0. No in-repo parity test yet (TODO_LIST #14). Removed at v1.0.0. |

## Planned (documented, no code)

| Feature                                             | Status   | Evidence / Notes                                                       |
| --------------------------------------------------- | -------- | ---------------------------------------------------------------------- |
| RFC 9111 §4.3.5 HEAD-based freshening               | PLANNED  | Deliberate opt-out, documented in `client/doc.go`; TODO_LIST #5        |
| `Vary`-aware cache selection                        | PLANNED  | Sharp edge documented (`client/doc.go` §Vary); KeyFunc is the mitigation |
| True freshness-based serving (§4.2 max-age/Expires) | PLANNED  | ROADMAP Theme 1 — changes the library's nature; needs opt-in design    |
| Client observability hooks                          | PLANNED  | ROADMAP Theme 3 — demand-gated behind `Stats()`                         |
| `go-etag/otel` sub-module                           | PLANNED  | Parked, demand-gated; see ROADMAP + `docs/review-and-roadmap.md`       |

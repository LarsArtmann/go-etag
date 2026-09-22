# Features

Honest inventory of what exists, by status. Evidence cites code; every
FULLY_FUNCTIONAL row is exercised by the test suite (verified 2026-09-23 at
the post-metrics tree: `go test -race -count=1 -cover ./...` green — root
100.0% / server 98.6% / client 99.1% / entitytag 98.9% / metrics 96.8%
coverage).

Statuses: FULLY_FUNCTIONAL, PARTIALLY_FUNCTIONAL, BROKEN, PLANNED.

## Server middleware (`server/`, package `etag`)

| Feature                                                                                   | Status           | Evidence / Notes                                                                                                                                          |
| ----------------------------------------------------------------------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Automatic ETag generation from response bodies (FNV-64a default)                          | FULLY_FUNCTIONAL | `server/etag.go` (`New`, `computeETag`); spec + unit tests                                                                                                |
| `If-None-Match` → 304 Not Modified (weak comparison, 2xx only)                            | FULLY_FUNCTIONAL | `server/etag.go` (`shouldReturnNotModified`); `etag_bdd_test.go` RFC 7232 §3.2 specs                                                                      |
| HEAD compliance — Content-Length set, no body sent                                        | FULLY_FUNCTIONAL | RFC 7230 §3.3; `TestNew_HeadRequest_NoBody`, `TestSpec_RFC7232_HeadRequest`                                                                               |
| 304 strips Content-Length, keeps ETag                                                     | FULLY_FUNCTIONAL | RFC 7232 §4.1; `TestNew_304_ExcludesContentLength`                                                                                                        |
| Handler-set ETag adoption (`SkipIfPresent`)                                               | FULLY_FUNCTIONAL | `server/etag.go` (`resolveETag`); 304-passthrough + fallback tests                                                                                        |
| Per-route opt-out (`Skip` predicate)                                                      | FULLY_FUNCTIONAL | `server/etag.go`; true/false path tests                                                                                                                   |
| Custom hash functions (`HashFunc func([]byte) string`)                                    | FULLY_FUNCTIONAL | `TestNew_CustomHashFunc` (+ body-bytes test)                                                                                                              |
| Buffer overflow → stream without ETag (`MaxBufferSize`)                                   | FULLY_FUNCTIONAL | `TestNew_MemoryLimit_DisablesETag`, overflow write-error tests                                                                                            |
| Hijack/Flush → streaming mode                                                             | FULLY_FUNCTIONAL | `server/wrapper.go`; hijack/flush delegate tests, streaming write tests                                                                                   |
| Observability hooks (`OnETagGenerated` / `On304` / `OnBufferOverflow` / `OnError`)        | FULLY_FUNCTIONAL | v0.2.0; exactly-once/ordering/nil-safety specs; no telemetry dependency                                                                                   |
| `ETag` domain type — parse, strength, strong/weak comparison                              | FULLY_FUNCTIONAL | `entitytag/entity_tag.go` (re-exported by `server/entity_tag.go`); `entitytag/entity_tag_test.go` + fuzz round-trip (`entitytag/entity_tag_fuzz_test.go`) |
| Conditional-request helpers (`MatchesIfNoneMatch`, `MatchesIfMatch`)                      | FULLY_FUNCTIONAL | RFC 7232 §3.1/§3.2; lost-update example in README                                                                                                         |
| Classified errors (`go-error-family`, 5 codes + sentinel + typed `Code`/`Domain` routing) | FULLY_FUNCTIONAL | `server/errors.go` + `server/code.go`; `errors_test.go` (registration, bidirectional template completeness via `allETagErrorCodes`), `code_test.go`       |

## Client transport (`client/`, package `etagclient`)

| Feature                                                                                                                           | Status           | Evidence / Notes                                                                                                         |
| --------------------------------------------------------------------------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Conditional GET transport — replay stored validators as `If-None-Match`                                                           | FULLY_FUNCTIONAL | `client/transport.go` (`RoundTrip`); `transport_test.go`                                                                 |
| 304 → 200 rebuild from cache (callers see no difference)                                                                          | FULLY_FUNCTIONAL | `rebuildFromCache`; `TestRoundTripRebuilds304FromCache` + GoDoc example                                                  |
| RFC 9111 §4.3.4 freshening (typed `FreshenPolicy`: default every-304-field / `FreshenFields` restricted / `FreshenNone` disabled) | FULLY_FUNCTIONAL | `freshenedHeader`, `Options.FreshenOn304`; `spec_test.go` freshening specs — shipped in v0.3.0                           |
| RFC 9111 §4.3.5 HEAD-based freshening (update or invalidate stored entries from HEAD 200s)                                        | FULLY_FUNCTIONAL | `roundTripHead`, `headConfirmsStored`; `TestSpecHeadFreshening` suite + real-server integration test — shipped in v0.3.0 |
| Request `no-store` bypass (§5.2.2.5 request directive: no read, no write)                                                         | FULLY_FUNCTIONAL | `TestSpecRequestNoStoreBypassesTheCache` — shipped in v0.3.0                                                             |
| §4.4 unsafe-method invalidation (2xx/3xx to unsafe method drops entry)                                                            | FULLY_FUNCTIONAL | `responseCache.invalidate`, `roundTripUnsafe`; `TestSpecUnsafeMethodInvalidatesEntry`                                    |
| `Cache-Control: no-store` storage ban (quote-aware, case-insensitive)                                                             | FULLY_FUNCTIONAL | `hasNoStoreDirective`; `TestSpecNoStoreResponseIsNeverCached` incl. quoted-arg subtests                                  |
| Caller-supplied `If-None-Match` is never clobbered                                                                                | FULLY_FUNCTIONAL | `TestSpecCallerSuppliedIfNoneMatchIsHonored`                                                                             |
| Hop-by-hop field stripping at store time (§3.1)                                                                                   | FULLY_FUNCTIONAL | `TestSpecStoredHeaderShedsHopByHopFields`                                                                                |
| Age surfaced verbatim on 200s; freshened from 304s (never backwards)                                                              | FULLY_FUNCTIONAL | `TestSpecStale200SurfacesAgeToCaller`; motivated by a real CDN field report                                              |
| Mismatched 304 validators not adopted (§4.3.4 filtering)                                                                          | FULLY_FUNCTIONAL | `restoreMismatchedValidator`; `TestSpecMismatched304ValidatorIsNotAdopted`                                               |
| Bounded cache — FIFO eviction (`MaxEntries`), body-size skip (`MaxBodyBytes`)                                                     | FULLY_FUNCTIONAL | `client/cache.go`; eviction + boundary tests                                                                             |
| Credential-scoped cache keys (`KeyFunc` hook + loud doc warning)                                                                  | FULLY_FUNCTIONAL | `TestRoundTripSeparatesCredentialsViaKeyFunc`, `client/doc.go` warning                                                   |
| From-cache marker header (`FromCacheHeader`)                                                                                      | FULLY_FUNCTIONAL | set/absent/stale-strip tests                                                                                             |
| Cache telemetry (`Stats{Hits, Stored, Entries}`)                                                                                  | FULLY_FUNCTIONAL | `TestResponseCacheStats`                                                                                                 |
| Real-wire verification via `httptest.Server` + real `http.Client`                                                                 | FULLY_FUNCTIONAL | `client/integration_test.go` (canonical forms, bodiless 304, Age flow)                                                   |

## Metrics companion (`metrics/`, package `metrics`)

Moved in from `httputil/etagmetrics` (2026-09-22); details: `docs/planning/2026-09-22_23-25_move-etagmetrics-into-go-etag-metrics.md`.

| Feature                                                                               | Status           | Evidence / Notes                                                                                                              |
| ------------------------------------------------------------------------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `Attach(cfg)` — counting hooks on a config copy, pre-existing hooks preserved         | FULLY_FUNCTIONAL | `metrics/metrics.go`; `TestAttach_ChainsExistingHooks` (user hooks fire after counting)                                       |
| Atomic counters `Generated` / `NotModified` / `BufferOverflows` + `Snapshot()`        | FULLY_FUNCTIONAL | `TestAttach_CountsFreshGET`, `TestAttach_CountsConditional304`, `TestAttach_CountsBufferOverflow`                             |
| `HitRatio()` = `NotModified / Generated` (exact denominator; fixed vs httputil)       | FULLY_FUNCTIONAL | `TestAttach_CountsConditional304` (1/2), `TestHitRatio_ZeroBeforeEvents`, `TestHitRatio_AdoptedTag304DoesNotCountAsGenerated` |
| Overflow contract — streamed overflow fires `OnBufferOverflow`, not `OnETagGenerated` | FULLY_FUNCTIONAL | `TestAttach_CountsBufferOverflow`                                                                                             |
| Hook overhead benchmark (~9 ns/op over hook-less baseline)                            | FULLY_FUNCTIONAL | `BenchmarkETagMetricsHookOverhead` vs `BenchmarkETagPlainNoHooks`                                                             |

## Root package (deprecated shim)

| Feature                                                          | Status           | Evidence / Notes                                                                                                                                                                                           |
| ---------------------------------------------------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| v0.1.x compatibility shim — full alias surface, `// Deprecated:` | FULLY_FUNCTIONAL | `deprecated.go` (4 types, 7 consts, 1 var, 8 func wrappers); `deprecated_test.go` export-parity suite (shipped at `a5de386`) pins type identity + wrapper behavior, root coverage 100%. Removed at v1.0.0. |

## Planned (documented, no code)

| Feature                                             | Status  | Evidence / Notes                                                         |
| --------------------------------------------------- | ------- | ------------------------------------------------------------------------ |
| `Vary`-aware cache selection                        | PLANNED | Sharp edge documented (`client/doc.go` §Vary); KeyFunc is the mitigation |
| True freshness-based serving (§4.2 max-age/Expires) | PLANNED | ROADMAP Theme 1 — changes the library's nature; needs opt-in design      |
| Client observability hooks                          | PLANNED | ROADMAP Theme 3 — demand-gated behind `Stats()`                          |
| `go-etag/otel` sub-module                           | PLANNED | Parked, demand-gated; see ROADMAP + `docs/review-and-roadmap.md`         |

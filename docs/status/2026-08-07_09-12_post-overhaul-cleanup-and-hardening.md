# Status Report: Post-Overhaul Cleanup & Hardening

**Date:** 2026-08-07 09:12
**Session scope:** Resolve all remaining issues from the comprehensive review self-audit

---

## Verification Baseline (end of session)

| Check                                 | Result                        |
| ------------------------------------- | ----------------------------- |
| `go build ./...`                      | PASS                          |
| `go vet ./...`                        | clean                         |
| `go test -race -count=1 -cover ./...` | PASS, 92.1% coverage          |
| `golangci-lint run ./...`             | 0 issues                      |
| `golangci-lint fmt`                   | clean                         |
| Fuzz `FuzzETag` (15s)                 | 4.6M execs, 0 crashes         |
| Fuzz `FuzzParseETag` (15s)            | 5.5M execs, 0 crashes         |
| Fuzz `FuzzParseETagList` (15s)        | 5.3M execs, 0 crashes         |
| Dependencies                          | 1 (`go-error-family v0.10.0`) |
| LOC (Go)                              | ~2439 across 13 files         |
| Total LOC (incl. docs)                | ~2920                         |

---

## a) FULLY DONE

### This session (5 items from prior self-audit)

1. **Hex encoding rewrite** (`hex.go`, `etag.go`) — Replaced `strings.Builder`-based `hexEncode` with `hexEncodeUint64`, a stack-allocated `[16]byte` array approach that writes hex nibbles directly from a `uint64`. Removed the `encoding/binary` import from `etag.go`. The orphaned `hashUint64HexChars` constant is now consumed as the array size. Cleaner code path, no intermediate `[]byte` allocation.

2. **Stale roadmap updated** (`docs/review-and-roadmap.md`) — Rewrote from 254 lines of open-bug descriptions to 91 lines reflecting completed state. All B1/B2/B3 bugs and D1-D8 design items marked done with fix locations. Restructured as a historical audit record.

3. **Fuzz tests for entity-tag parsers** (`entity_tag_fuzz_test.go`, 80 lines) — Added `FuzzParseETag` (verifies no panics + round-trip: `ParseETag(tag.String()) == tag`) and `FuzzParseETagList` (verifies non-nil result + every element valid + round-trips). Combined: 10.8M executions, 0 crashes.

4. **Migration guide** (`docs/migration/v0.2.md`, 155 lines) — Covers all 6 breaking changes with before/after code: `ETag()` → `New()`, `Weak bool` → `Strength`, `HashFunc` signature, new `ETag` type helpers, new config fields, behavior-changing bug fixes. Linked from README and CHANGELOG.

5. **Documentation cross-references** — Added migration guide link to README (callout block), updated CHANGELOG with fuzz test entry, hex encoding entry, and migration guide reference.

### Carried over from prior session (all verified still green)

- Dedicated `ETag` type (`entity_tag.go`) with `Strength` enum, `StrongEqual`/`WeakEqual`, `ParseETag`/`ParseETagList`, `MatchesIfNoneMatch`/`MatchesIfMatch`
- B1 fix: zero-value config clamping (`etag.go:176-179`)
- B2 fix: HEAD body suppression (`etag.go:264-270`)
- B3 fix: 304 Content-Length strip (`etag.go:251`)
- D1: `SkipIfPresent` config
- D2: `Strength` enum replacing `Weak bool`
- D3: `HashFunc func([]byte) string` signature
- D4: Trimmed `errors.go` classifications
- D5: `headerBuffered`/`headerCommitted` rename
- D6: `Skip` predicate
- D7: `MatchesIfMatch` helper
- D8: Dead code cleanup
- BDD spec suite (`etag_bdd_test.go`)
- Error classification via `go-error-family`

---

## b) PARTIALLY DONE

Nothing is partially done. All items are either complete or not started.

---

## c) NOT STARTED

- **Allocation reduction beyond hex encoding** — The 13 allocs/op in `BenchmarkETag` are dominated by `httptest.ResponseRecorder` infrastructure (Header.Clone, NewRecorder, MIMEHeader.Set), not production code. The production-path allocations that remain are: `hexEncodeUint64` string conversion (1, unavoidable for stack array → string), `ETag.String()` concatenation (1), `splitRawETags` slice (1), `ParseETagList` slice (1), `newETagWriter` struct (1), `body` buffer append (1-2). Further reduction requires rethinking the `ETag.String()` and header-set paths.

- **`If-Modified-Since` / `If-Unmodified-Since`** — RFC 7232 §3.3/§3.4 date-based validators. Out of current scope.

- **§6 Precedence ordering** — Full conditional-request precedence chain. Out of current scope.

- **Versioned release tag (`v0.2.0`)** — The CHANGELOG has an `[Unreleased]` section but no git tag has been cut.

---

## d) TOTALLY FUCKED UP

**Nothing.** All changes are clean, lint-passing, test-passing, and committed.

**Near-miss worth noting:** The LSP (`golangci_lint_ls`) reported stale `nestif` and `gocritic` warnings on `entity_tag_fuzz_test.go` throughout the session — the diagnostics referenced line numbers and code patterns (`if ok`) from an earlier draft that no longer existed in the file (`if !ok` with early return). Restarting the LSP cleared these. `golangci-lint run` confirmed 0 issues throughout. This was purely a stale cache, not a real problem.

---

## e) WHAT WE SHOULD IMPROVE

### Architectural / Design

1. **`ETag.String()` allocates on every call** — `W/"` + opaque + `"` uses string concatenation that escapes to heap. For the middleware hot path (called once per response), this is acceptable, but for comparison-heavy code paths it adds up. Could pre-compute the wire format at construction time in `NewETag`, trading a larger struct for zero allocs on `String()`.

2. **`splitRawETags` returns `[]string` then `ParseETagList` parses each** — Two-pass: split then parse. A single-pass parser that builds `[]ETag` directly would eliminate the intermediate string slice and the per-element `TrimSpace` allocations.

3. **`resolveETag` calls `w.Header().Set(headerETag, tag.String())` then `shouldReturnNotModified` re-parses the header** — The tag is computed, stringified, set on the header, then on the 304 path the header is not read again so this is fine. But on the non-304 path, the `tag.String()` allocation is wasted if `SkipIfPresent` already parsed an existing header. Minor.

4. **No `HashFunc` streaming interface** — The current `func([]byte) string` requires the full body in memory. A `hash.Hash`-based interface would enable streaming ETag computation for large responses without buffering, but this fundamentally conflicts with the buffer-and-compare architecture.

### Testing / Verification

5. **Coverage stuck at 92.1%** — The uncovered ~8% is likely error branches in `Flush`/`Hijack` paths and `OnError` callbacks. A coverage HTML report would identify exact lines.

6. **No benchmark for `ParseETag` / `ParseETagList` in isolation** — These are exported functions that users call in hot paths. Should have dedicated benchmarks separate from the middleware path.

7. **No integration test with a real `net/http.Server`** — All tests use `httptest.NewRecorder`. A test using `httptest.NewServer` would verify behavior through a real HTTP stack (chunked encoding, actual Content-Length, connection reuse).

### Documentation

8. **No GoDoc examples on individual functions** — The `example_test.go` has package-level examples but no function-specific `ExampleNewETag`, `ExampleParseETag`, `ExampleMatchesIfMatch`.

9. **The 3 open questions from the prior session are still unanswered** — FNV default strength, scope growth, subpackage split. These are product decisions, not technical ones.

### Operational

10. **No CI pipeline visible** — No `.github/workflows/` directory. The verification commands are documented but not automated.

11. **No `go.mod` version pinning for Go 1.26** — The `go.mod` specifies `go 1.26` but there's no toolchain directive. Consumers on older Go versions will get unclear errors.

---

## f) Up to 50 Things We Should Get Done Next

### P0 — Correctness & Safety

| # | Task                                                                                                                                                       | Effort |
| - | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1 | Write integration test using `httptest.NewServer` to verify real HTTP behavior (Content-Length, chunked, HEAD body suppression through actual TCP)         | M      |
| 2 | Generate coverage HTML report (`go test -coverprofile=cover.out -covermode=atomic ./... && go tool cover -html=cover.out`) and identify the uncovered 7.9% | S      |
| 3 | Add tests for every uncovered branch identified by the coverage report                                                                                     | S-M    |

### P1 — Performance

| # | Task                                                                                                                      | Effort |
| - | ------------------------------------------------------------------------------------------------------------------------- | ------ |
| 4 | Pre-compute wire format in `NewETag` to eliminate `ETag.String()` allocation                                              | S      |
| 5 | Single-pass `ParseETagList` that builds `[]ETag` directly without intermediate `[]string`                                 | S      |
| 6 | Add isolated benchmarks for `ParseETag`, `ParseETagList`, `StrongEqual`, `WeakEqual`                                      | XS     |
| 7 | Benchmark with larger bodies (4KB, 64KB, 256KB) to characterize scaling behavior                                          | XS     |
| 8 | Investigate whether `fnv.New64a()` can be replaced with an inline FNV implementation to avoid the interface call overhead | S      |

### P2 — API & Types

| #  | Task                                                                                                                           | Effort |
| -- | ------------------------------------------------------------------------------------------------------------------------------ | ------ |
| 9  | Add `ETag.IsStrong()` method for symmetry with `IsWeak()`                                                                      | XS     |
| 10 | Consider `ETag.IsEmpty()` or documenting that `!IsValid()` is the zero-value check                                             | XS     |
| 11 | Add `Strength.String()` method (`"strong"`/`"weak"`) for logging and debugging                                                 | XS     |
| 12 | Consider whether `MatchesIfMatch` should return `(bool, error)` instead of `bool` to handle parse failures distinctly          | S      |
| 13 | Evaluate whether `SkipIfPresent` should default to `true` (respect handler ETags by default) vs `false` (overwrite by default) | S      |

### P3 — Documentation

| #  | Task                                                                                                                               | Effort |
| -- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 14 | Add function-level `ExampleNewETag`, `ExampleParseETag`, `ExampleMatchesIfMatch`, `ExampleMatchesIfNoneMatch` in `example_test.go` | S      |
| 15 | Add a "Performance" section to README with benchmark numbers and allocation expectations                                           | XS     |
| 16 | Add an "RFC 7232 Compliance Matrix" to README mapping each spec section to implementation status                                   | S      |
| 17 | Document the collision tradeoff of FNV-64a more prominently (birthday bound table: 1K bodies, 1M bodies, 1B bodies)                | XS     |
| 18 | Add a `docs/` index or `CHANGELOG` link in README                                                                                  | XS     |
| 19 | Consider an ADR for the `Strength` enum vs `bool` decision                                                                         | S      |

### P4 — Testing

| #  | Task                                                                                                    | Effort |
| -- | ------------------------------------------------------------------------------------------------------- | ------ |
| 20 | Add property-based tests (rapid/go-quickcheck style via `testing/quick`) for ETag round-trip properties | M      |
| 21 | Add tests for concurrent `ServeHTTP` calls on the same middleware instance (thread safety)              | S      |
| 22 | Add tests for `OnError` callback invocation paths (write failure after header commit)                   | S      |
| 23 | Add tests for `Flush` → `Write` → `Flush` sequence (streaming after flush)                              | S      |
| 24 | Add tests for edge-case status codes (204, 206, 301, 302, 403, 404, 500)                                | S      |
| 25 | Add a test for `MaxBufferSize` boundary (body exactly equals limit)                                     | XS     |
| 26 | Add tests for `Skip` predicate with various request properties (method, path, headers)                  | XS     |
| 27 | Add a fuzz test for the middleware with `Skip` and `SkipIfPresent` enabled                              | S      |

### P5 — Scope & Features

| #  | Task                                                                               | Effort |
| -- | ---------------------------------------------------------------------------------- | ------ |
| 28 | Evaluate `If-Modified-Since` / `If-Unmodified-Since` support (RFC 7232 §3.3/§3.4)  | M      |
| 29 | Evaluate full §6 precedence ordering if scope grows to conditional-request library | L      |
| 30 | Consider a `WeakETag(opaque string)` convenience constructor                       | XS     |
| 31 | Consider a `StrongETag(opaque string)` convenience constructor                     | XS     |
| 32 | Consider exporting `splitRawETags` for users who need raw splitting                | XS     |

### P6 — Operational / Release

| #  | Task                                                                                               | Effort |
| -- | -------------------------------------------------------------------------------------------------- | ------ |
| 33 | Cut `v0.2.0` git tag once open questions are answered                                              | XS     |
| 34 | Set up CI pipeline (`.github/workflows/ci.yml`) with build, vet, lint, test -race, coverage upload | M      |
| 35 | Add `toolchain` directive to `go.mod`                                                              | XS     |
| 36 | Add `golangci-lint` version pinning (`.golangci-lint-version` or CI config)                        | XS     |
| 37 | Add `codecov.yml` or equivalent coverage gating config                                             | S      |
| 38 | Set up `goreleaser` config for automated releases                                                  | M      |

### P7 — Code Quality

| #  | Task                                                                                                                                   | Effort |
| -- | -------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 39 | Consider splitting `etag.go` (365 lines) — `etag.go` (middleware) + `config.go` (`ETagConfig`, `DefaultETagConfig`, `Validate`)        | S      |
| 40 | Evaluate whether `responseWrapper` and `etagWriter` should use composition more explicitly (interface for the wrapper contract)        | S      |
| 41 | Review whether `writeDefaultOK` should be called in `Write` or whether `flush` should handle it                                        | S      |
| 42 | Consider whether `etagWriter.Flush` and `etagWriter.flush(req)` naming collision is confusing (exported `Flush` vs unexported `flush`) | XS     |
| 43 | Review error message consistency (some use "etag writer X", others use "etag X")                                                       | XS     |

### P8 — Alignment with `go-error-family`

| #  | Task                                                                                                               | Effort |
| -- | ------------------------------------------------------------------------------------------------------------------ | ------ |
| 44 | Verify `OnError` callback receives properly classified errors in all failure paths                                 | S      |
| 45 | Consider whether `ErrInvalidConfig` should carry the field name and value as structured context                    | XS     |
| 46 | Document which `errorfamily.ErrorFamily` each error code belongs to in the error classification table              | XS     |
| 47 | Consider whether `defaultHashFunc`'s `NewOrchestration` panic message should include the body length for debugging | XS     |

### P9 — Ecosystem

| #  | Task                                                                                       | Effort |
| -- | ------------------------------------------------------------------------------------------ | ------ |
| 48 | Create a comparison table vs other Go ETag libraries (gin-contrib/etag, go-chi/etag, etc.) | S      |
| 49 | Add the library to awesome-go or similar curated lists                                     | XS     |
| 50 | Write a blog post or announcement for the v0.2 release                                     | M      |

---

## g) Open Questions (Cannot Resolve Without User Input)

### 1. Should FNV-64a default to `Strong` or `Weak`?

**Current:** `Strong` (set in `DefaultETagConfig`).

**The tradeoff:** FNV-64a is a non-cryptographic hash with a 64-bit output. Its birthday bound is ~4.3 billion distinct bodies before a 50% collision chance. A collision produces a stale 304 — effectively silent data corruption. Technically, only byte-exact content addresses or cryptographic hashes are true strong validators per RFC 7232 §2.1.

**Why I can't decide:** `Strong` is pragmatic — for 99.99% of applications, 4.3 billion bodies is astronomically unlikely. But `Weak` is honest — it doesn't claim a guarantee the hash can't back up. This is a correctness-vs-pragmatism product decision, not a technical one.

### 2. Should this grow into a full conditional-request library or stay focused?

**Current:** Focused on ETag generation + `If-None-Match` handling, with `MatchesIfMatch`/`MatchesIfNoneMatch` as exported helpers for manual use.

**The implication:** Growing scope means adding `If-Modified-Since` (§3.3), `If-Unmodified-Since` (§3.4), §6 precedence ordering, and potentially 412 Precondition Failed middleware. That's a different library — a conditional-request library that happens to also generate ETags.

**Why I can't decide:** This determines whether items #28-#29 in the next-steps table are in scope. It's a positioning decision: "the best ETag library for Go" vs "the best conditional-request library for Go." The name `go-etag` suggests the former.

### 3. Should `entity_tag.go` be split into a separate subpackage?

**Current:** Flat `etag` package — `entity_tag.go`, `etag.go`, `wrapper.go`, etc. are all `package etag`.

**The consideration:** A separate subpackage (e.g., `etag/etagvalue` or `etag/entity`) would allow the `ETag` type to be used without importing the middleware. This matters if someone wants the parsing/comparison functions without the `http.Handler` wrapping.

**Why I can't decide:** The flat package is simpler and the types are tightly coupled (the middleware uses `ETag` internally). A subpackage adds import complexity for a benefit that may be theoretical — most users want the middleware, not just the type. But I don't know if Lars has a preference or a downstream consumer that needs this.

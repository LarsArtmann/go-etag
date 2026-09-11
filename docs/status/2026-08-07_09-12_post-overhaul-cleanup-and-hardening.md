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

- **Allocation reduction beyond hex encoding** — The 13 allocs/op in `BenchmarkETag` are dominated by `httptest.ResponseRecorder` infrastructure (Header.Clone, NewRecorder, MIMEHeader.Set), not production code. The production-path allocations that remain are: `hexEncodeUint64` string conversion (1, unavoidable for stack array → string), `ETag.String()` concatenation (1), `splitRawETags` slice (1), `ParseETagList` slice (1), `newETagWriter` struct (1), `body` buffer append (1-2). Further reduction requires rethinking the `ETag.String()` and header-set paths. (Still open — the two-pass parse and String() concatenation remain.)
- ~~**`If-Modified-Since` / `If-Unmodified-Since`** — RFC 7232 §3.3/§3.4 date-based validators. Out of current scope.~~ **Won't implement server-side — scope boundary; client-side second-validator idea lives in ROADMAP Theme 1**
- ~~**§6 Precedence ordering** — Full conditional-request precedence chain. Out of current scope.~~ **Won't implement — deliberate scope boundary (ROADMAP Non-goals)**
- ~~**Versioned release tag (`v0.2.0`)** — The CHANGELOG has an `[Unreleased]` section but no git tag has been cut.~~ done — v0.1.0 tagged 2026-08-07; v0.2.0 (the split) at `be19640`

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

5. ~~**Coverage stuck at 92.1%** — The uncovered ~8% is likely error branches in `Flush`/`Hijack` paths and `OnError` callbacks. A coverage HTML report would identify exact lines.~~ done — coverage rose to 99.0% (server) / 94.3% (client) via the 09-32 session + RFC 9111 suites

6. **No benchmark for `ParseETag` / `ParseETagList` in isolation** — These are exported functions that users call in hot paths. Should have dedicated benchmarks separate from the middleware path.

~~7. **No integration test with a real `net/http.Server`** — All tests use `httptest.NewRecorder`. A test using `httptest.NewServer` would verify behavior through a real HTTP stack (chunked encoding, actual Content-Length, connection reuse).~~ done at `9204885` (client `integration_test.go`; the server package remains recorder-based — TODO_LIST #15)

### Documentation

8. **No GoDoc examples on individual functions** — The `example_test.go` has package-level examples but no function-specific `ExampleNewETag`, `ExampleParseETag`, `ExampleMatchesIfMatch`.

9. **The 3 open questions from the prior session are still unanswered** — FNV default strength, scope growth, subpackage split. These are product decisions, not technical ones.

### Operational

10. **No CI pipeline visible** — No `.github/workflows/` directory. The verification commands are documented but not automated.

~~11. **No `go.mod` version pinning for Go 1.26** — The `go.mod` specifies `go 1.26` but there's no toolchain directive. Consumers on older Go versions will get unclear errors.~~ done at `0e8ac6d` (go.mod pins `go 1.26.7`; CI reads `go-version-file: go.mod`)

---

## f) Up to 50 Things We Should Get Done Next

### P0 — Correctness & Safety

| # | Task                                                                                                                                                                                                                                                                                  | Effort |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1 | ~~Write integration test using `httptest.NewServer` to verify real HTTP behavior (client landed at `9204885`; server still open — TODO_LIST #15)~~ (Content-Length, chunked, HEAD body suppression through actual TCP) — server side done at `a5de386` (`server/integration_test.go`) | M      |
| 2 | ~~Generate coverage HTML report (superseded — coverage now 99.0%/94.3%)~~ (`go test -coverprofile=cover.out -covermode=atomic ./... && go tool cover -html=cover.out`) and identify the uncovered 7.9%                                                                                | S      |
| 3 | ~~Add tests for every uncovered branch identified by the coverage report~~ done at `a5de386`, `5eb226b` (branch-coverage suites; client 97.7%, gaps documented-intentional)                                                                                                           | S-M    |

### P1 — Performance

| # | Task                                                                                                                                                                         | Effort |
| - | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 4 | Pre-compute wire format in `NewETag` — open                                                                                                                                  | S      |
| 5 | Single-pass `ParseETagList` — open (two-pass remains)                                                                                                                        | S      |
| 6 | Add isolated benchmarks — open                                                                                                                                               | XS     |
| 7 | Benchmark with larger bodies — open                                                                                                                                          | XS     |
| 8 | ~~Investigate inline FNV — **Won't implement — interface-call overhead is negligible vs network I/O; benchmarks are sub-microsecond**~~ to avoid the interface call overhead | S      |

### P2 — API & Types

| #  | Task                                                                                                                                                                                                              | Effort |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 9  | Add `ETag.IsStrong()` — open (low)                                                                                                                                                                                | XS     |
| 10 | ~~Consider `ETag.IsEmpty()` — done: `IsValid()` is the shipped zero-value check, documented~~                                                                                                                     | XS     |
| 11 | Add `Strength.String()` — open (low)                                                                                                                                                                              | XS     |
| 12 | ~~Consider whether `MatchesIfMatch` should return `(bool, error)` instead of `bool` to handle parse failures distinctly~~ **Won't implement — parse failures are defined as no-match (RFC behavior), not errors** | S      |
| 13 | ~~Evaluate `SkipIfPresent` default — done (decision: `false`, documented)~~                                                                                                                                       | S      |

### P3 — Documentation

| #  | Task                                                                                                                   | Effort |
| -- | ---------------------------------------------------------------------------------------------------------------------- | ------ |
| 14 | Add function-level examples — open                                                                                     | S      |
| 15 | ~~Add "Performance" section to README — done at `b9eb32e` (Benchmarks section)~~                                       | XS     |
| 16 | ~~Add RFC 7232 Compliance Matrix — done at `b9eb32e`~~                                                                 | S      |
| 17 | ~~Document FNV-64a collision tradeoff — done (entity_tag.go + README Strong-vs-Weak section)~~                         | XS     |
| 18 | Add a `docs/` index or CHANGELOG link in README — open (low)                                                           | XS     |
| 19 | ~~Consider an ADR for Strength enum — **Won't implement — decision is documented in AGENTS.md + review-and-roadmap**~~ | S      |

### P4 — Testing

| #  | Task                                                                                                                                              | Effort |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 20 | ~~Add property-based tests — done at `e0fe51f` (fuzz round-trip invariants)~~                                                                     | M      |
| 21 | ~~Concurrent `ServeHTTP` tests — done (per-request writer + full suite under -race; client has explicit `TestRoundTripConcurrent` at `bc5a551`)~~ | S      |
| 22 | ~~`OnError` invocation-path tests — done at `c759373`~~                                                                                           | S      |
| 23 | Flush → Write → Flush sequence test — open                                                                                                        | S      |
| 24 | ~~Edge-case status-code tests — done at `c759373` (7 non-cacheable statuses + 299/300 boundary)~~                                                 | S      |
| 25 | `MaxBufferSize` boundary test — open                                                                                                              | XS     |
| 26 | ~~`Skip` predicate tests — done at `15114b3` (true/false paths)~~                                                                                 | XS     |
| 27 | Fuzz with `Skip`/`SkipIfPresent` enabled — open (low)                                                                                             | S      |

### P5 — Scope & Features

| #  | Task                                                                                                                               | Effort |
| -- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 28 | ~~`If-Modified-Since` / `If-Unmodified-Since` — **Won't implement server-side; client second-validator idea in ROADMAP Theme 1**~~ | M      |
| 29 | ~~Full §6 precedence — **Won't implement — scope boundary**~~                                                                      | L      |
| 30 | ~~`WeakETag` ctor — **Won't implement — `NewETag(opaque, Weak)` is explicit**~~                                                    | XS     |
| 31 | ~~`StrongETag` ctor — **Won't implement — same**~~                                                                                 | XS     |
| 32 | ~~Export `splitRawETags` — **Won't implement — internal detail**~~                                                                 | XS     |

### P6 — Operational / Release

| #  | Task                                                                                                | Effort |
| -- | --------------------------------------------------------------------------------------------------- | ------ |
| 33 | ~~Cut `v0.2.0` — done at `be19640`~~                                                                | XS     |
| 34 | ~~Set up CI pipeline — done at `37b68b3`~~                                                          | M      |
| 35 | ~~Add `toolchain` directive — done at `0e8ac6d` (go 1.26.7)~~                                       | XS     |
| 36 | ~~golangci-lint version pinning — done at `cf98ab5` (CI installs v2.12.2 from the v2 module path)~~ | XS     |
| 37 | Add `codecov.yml` or coverage gating — open (low)                                                   | S      |
| 38 | ~~goreleaser — **Won't implement — library, not binary; go-release flow suffices**~~                | M      |

### P7 — Code Quality

| #  | Task                                                                                               | Effort |
| -- | -------------------------------------------------------------------------------------------------- | ------ |
| 39 | ~~Split `etag.go` — **Won't implement — file size is manageable; current split is deliberate**~~   | S      |
| 40 | ~~Wrapper composition refactor — **Won't implement — current structure is idiomatic**~~            | S      |
| 41 | ~~`writeDefaultOK` placement review — done (placement settled; behavior pinned by wrapper tests)~~ | S      |
| 42 | ~~Flush/flush naming review — **Won't implement — conventional Go naming**~~                       | XS     |
| 43 | Error message consistency — open (low)                                                             | XS     |

### P8 — Alignment with `go-error-family`

| #  | Task                                                                                                            | Effort |
| -- | --------------------------------------------------------------------------------------------------------------- | ------ |
| 44 | ~~Verify `OnError` classification — done at `c759373`~~                                                         | S      |
| 45 | ~~`ErrInvalidConfig` structured context — done (`WithContextf("max_buffer_size"…/"strength"…) since`15114b3`)~~ | XS     |
| 46 | ~~Document error families in the table — done (AGENTS.md Family column)~~                                       | XS     |
| 47 | ~~Panic-message body length — **Won't implement — contract-violation path; minimal message is right**~~         | XS     |

### P9 — Ecosystem

| #  | Task                                                                                            | Effort |
| -- | ----------------------------------------------------------------------------------------------- | ------ |
| 48 | Comparison table — open (ROADMAP Theme 4)                                                       | S      |
| 49 | awesome-go listing — open (ROADMAP Theme 4)                                                     | XS     |
| 50 | ~~Blog post for v0.2 — **Won't implement — no blog channel; GitHub releases carry the notes**~~ | M      |

---

## g) Open Questions (Cannot Resolve Without User Input)

### 1. Should FNV-64a default to `Strong` or `Weak`?

**Current:** `Strong` (set in `DefaultETagConfig`).

**The tradeoff:** FNV-64a is a non-cryptographic hash with a 64-bit output. Its birthday bound is ~4.3 billion distinct bodies before a 50% collision chance. A collision produces a stale 304 — effectively silent data corruption. Technically, only byte-exact content addresses or cryptographic hashes are true strong validators per RFC 7232 §2.1.

**Why I can't decide:** `Strong` is pragmatic — for 99.99% of applications, 4.3 billion bodies is astronomically unlikely. But `Weak` is honest — it doesn't claim a guarantee the hash can't back up. This is a correctness-vs-pragmatism product decision, not a technical one.

_**Resolved:** `Strong` kept — tradeoff documented; standing revisit note in ROADMAP.md Open Questions #4._

### 2. Should this grow into a full conditional-request library or stay focused?

**Current:** Focused on ETag generation + `If-None-Match` handling, with `MatchesIfMatch`/`MatchesIfNoneMatch` as exported helpers for manual use.

**The implication:** Growing scope means adding `If-Modified-Since` (§3.3), `If-Unmodified-Since` (§3.4), §6 precedence ordering, and potentially 412 Precondition Failed middleware. That's a different library — a conditional-request library that happens to also generate ETags.

**Why I can't decide:** This determines whether items #28-#29 in the next-steps table are in scope. It's a positioning decision: "the best ETag library for Go" vs "the best conditional-request library for Go." The name `go-etag` suggests the former.

_**Resolved:** stayed focused on generation + 304; growth happened on the client side instead (conditional-GET transport, v0.2.0)._

### 3. Should `entity_tag.go` be split into a separate subpackage?

**Current:** Flat `etag` package — `entity_tag.go`, `etag.go`, `wrapper.go`, etc. are all `package etag`.

**The consideration:** A separate subpackage (e.g., `etag/etagvalue` or `etag/entity`) would allow the `ETag` type to be used without importing the middleware. This matters if someone wants the parsing/comparison functions without the `http.Handler` wrapping.

**Why I can't decide:** The flat package is simpler and the types are tightly coupled (the middleware uses `ETag` internally). A subpackage adds import complexity for a benefit that may be theoretical — most users want the middleware, not just the type. But I don't know if Lars has a preference or a downstream consumer that needs this.

_**Resolved:** split happened differently at `bc5a551` (server/ + client/); the domain type lives in `server/`. Client adoption of the type is an open ROADMAP question (Theme 2)._

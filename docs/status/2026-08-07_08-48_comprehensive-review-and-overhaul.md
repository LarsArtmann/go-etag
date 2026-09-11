# Status Report — go-etag

**Date:** 2026-08-07 08:48\
**Session scope:** Full codebase review against RFC 7232 + comprehensive improvement pass\
**Baseline:** commit `b3313ec` (initial release) → Current: `15114b3` + uncommitted docs

---

## a) FULLY DONE (verified green: build + vet + test -race + lint + fmt + fuzz + bench)

### Bugs Fixed

- **B1 — Zero-value config unbounded buffering [P0]**: `ETagConfig{}` with `MaxBufferSize: 0` previously disabled the overflow guard in `Write`, causing unbounded RAM buffering (memory DoS vector). Fixed: `newETagWriter` clamps `<= 0` to `defaultMaxBufferSize` (1 MB). Test: `TestNew_ZeroValueConfig_ClampsBufferSize`.
- **B2 — HEAD forwards message body [P0]**: HEAD requests ran the same body-write tail as GET, sending body bytes to the client. RFC 7230 §3.3 violation. Fixed: HEAD path sets `Content-Length` from buffered size, then skips the `Write`. Tests: `TestNew_HeadRequest_NoBody`, `TestSpec_RFC7232_HeadRequest`.
- **B3 — 304 leaks Content-Length [P1]**: The 304 path returned without deleting body-length metadata. RFC 7232 §4.1 violation. Fixed: `Header().Del(headerContentLength)` before `WriteHeader(304)`. Tests: `TestNew_304_ExcludesContentLength`, `TestSpec_RFC7232_NotModifiedResponse`.
- **Duplicate doc comment**: `etag.go` had `defaultETagHash` doc comment duplicated twice (merge artifact from concurrent process). Eliminated by rewrite.

### Design Improvements Shipped

- **ETag type extraction**: New `entity_tag.go` with `ETag` struct, `Strength` enum (`Strong`/`Weak`), `NewETag`, `ParseETag`, `ParseETagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`. Both RFC 7232 §2.3.2 comparison functions (`StrongEqual`/`WeakEqual`) are now first-class methods.
- **D1 — `SkipIfPresent`**: Middleware respects handler-set ETags when `true`. Tests: 3 tests covering respect, 304 passthrough, fallback.
- **D3 — Flexible `HashFunc`**: Changed from `func([]byte) uint64` to `func([]byte) string`, allowing arbitrary opaque-tag values (SHA-256, revision numbers, etc.). Test: `TestNew_CustomHashFunc`.
- **D4 — Trimmed `errors.go`**: Removed `ErrNoCookie`, `ErrNoLocation`, `ErrSkipAltProtocol` classifications (unrelated to an ETag library). Kept only `ErrNotSupported` and `ErrAbortHandler`.
- **D5 — Renamed confusing fields**: `wroteHeader` → `headerBuffered`, `headerWritten` → `headerCommitted` in `wrapper.go`.
- **D6 — `Skip` predicate**: Per-route opt-out from ETag processing. Tests: 2 tests covering true/false paths.
- **D8 — Dead code + lint cleanup**: Removed unused `assertBodyContains`, fixed `unparam` on `newTestRequest` (removed `path` and `origin` params that were always `/` and `""`).

### Testing

- **63 tests** (up from ~30), **92.1% coverage**.
- **BDD spec suite** (`etag_bdd_test.go`, 333 lines): Maps RFC 7232 sections (§2.3.2 strong/weak comparison, §3.1 If-Match, §3.2 If-None-Match, §4.1 304 response, §2.3 entity-tag format) to behavioral tests.
- **Fuzz**: `FuzzETag` — 4.9M executions, 0 crashes in 20s.
- **Benchmarks**: 5 benchmarks including `MatchesIfNoneMatch` sub-benchmarks.

### Documentation

- **README.md**: Fully rewritten with new API, ETag type section, configuration reference, conditional-request helper example.
- **AGENTS.md**: Updated architecture table, API design section, non-obvious behaviors, error classification table.
- **CHANGELOG.md**: Added `[Unreleased]` section documenting all breaking changes, additions, fixes.
- **doc.go**: Updated package doc to advertise `ETag` type first.
- **example_test.go**: Two examples (`ExampleNew`, `ExampleETag`) with `// Output:` directives.

---

## b) PARTIALLY DONE

### D2 — Honest strong/weak hash defaults

- **What was done**: `Strength` enum replaces `Weak bool`, `Validate()` rejects invalid `Strength` values, docs mention collision tradeoff.
- **What was NOT done**: Did not actually change the default from `Strong` to `Weak` or add a documented recommendation. FNV-64a with `Strong` is the default, which is _pragmatically correct_ (4.3B birthday bound) but arguably _dishonest_ per RFC 7232 §2.1 strict reading (a hash is not a "revision control" strong validator). Decision deferred to user.
  _**Resolved (standing decision):** `Strong` default kept — collision tradeoff documented; revisit note lives in ROADMAP.md Open Questions #4._

### D7 — If-Match / 412 support

- **What was done**: Exported `MatchesIfMatch` helper using `StrongEqual` — applications can call it in their handlers.
- **What was NOT done**: No middleware-level If-Match interception (no automatic 412 response). No RFC 7232 §6 precedence chain. This was a deliberate scope decision (ETag generation ≠ conditional request evaluation for unsafe methods), but it should be documented as a boundary.
  _**Resolved (scope decision):** the boundary is now documented — README shows the manual If-Match pattern; ROADMAP.md Non-goals records "no server-side If-Match interception / §6 precedence"._

### D3 — Hash flexibility

- **What was done**: `HashFunc func([]byte) string` allows arbitrary opaque values.
- **What was NOT done**: No streaming hash support (hash as you write, not after full buffer). No `hash.Hash` interface option. The `[]byte` input means the entire body must be in memory before hashing.
  _**Resolved (won't fix):** streaming hashing conflicts with the buffer-and-compare architecture by design; the overflow path streams instead. Recorded as a ROADMAP.md Non-goal in spirit (buffering is the design)._

---

## c) NOT STARTED

1. ~~**RFC 7232 §3.3 If-Modified-Since** — no date-based validator support.~~ **Won't implement (server) — out of scope**; the client-side second validator remains a ROADMAP Theme 1 idea
2. ~~**RFC 7232 §3.4 If-Unmodified-Since** — no date-based validator support.~~ **Won't implement — out of scope**
3. ~~**RFC 7232 §6 Precedence ordering** — no conditional-request precedence chain.~~ **Won't implement — deliberate scope boundary (ROADMAP Non-goals)**
4. ~~**RFC 7233 If-Range** — no range request support.~~ **Won't implement — different problem domain (ROADMAP Non-goals)**
5. ~~**Migration guide** — breaking API change with no migration doc.~~ done — written at `e0fe51f`, then removed at `8890f8d` (nothing was ever released; the guide was fiction)
6. ~~**Fuzz tests for `entity_tag.go`** — `ParseETag`, `ParseETagList`, `MatchesIfMatch` have no fuzz coverage.~~ done at `e0fe51f` (`FuzzParseETag`, `FuzzParseETagList`)
7. ~~**Streaming hash** — hash body as it arrives instead of buffering then hashing.~~ **Won't implement — conflicts with the buffer-and-compare architecture (see D3 above)**
8. ~~**Integration tests with real `net/http` server** — all tests use `httptest.Recorder`.~~ done at `9204885` (client `integration_test.go`; server package remains recorder-based — TODO_LIST #15)
9. ~~**CI/CD pipeline** — no GitHub Actions workflow.~~ done at `37b68b3` (first green run `cf98ab5`; fuzz scoping fixed at `f1c77db`)
10. ~~**Version tagging** — no `git tag v0.2.0` for the breaking change.~~ done — version fiction reconciled: v0.1.0 tagged 2026-08-07, v0.2.0 (the split) at `be19640`
11. ~~**GoDoc badge / pkg.go.dev integration**.~~ done — pkg.go.dev live since v0.1.1
12. ~~**Performance optimization** — 13 allocs/op on the hot path (see section e).~~ done at `e0fe51f` (zero-alloc `hexEncodeUint64`); the remaining allocs are documented in README's benchmark table

---

## d) TOTALLY FUCKED UP

### Nothing is catastrophically broken, but:

1. ~~**Unused constant `hashUint64HexChars`**: Defined in `etag.go:23` as `hashUint64Bytes * 2` but never referenced anywhere. The old `encodeHex` function that used the related `etagWeakLen`/`etagStrongLen` constants was removed, but this constant survived. `golangci-lint` didn't catch it (the `unused` linter may not flag package-level constants in all configs, or the `_test` exclusion is too broad). **Must delete.**~~ done at `e0fe51f` (consumed as the array size by `hexEncodeUint64`)

2. ~~**`docs/review-and-roadmap.md` is now stale**: It describes B1/B2/B3 as open work items with "should be applied after reconciling." All three are done. The roadmap priorities, open questions, and ETag type proposal are all resolved. This file misleads any future reader.~~ done at `e0fe51f` (rewritten as a historical audit record)

3. ~~**No commit for the documentation updates**: `AGENTS.md` and `CHANGELOG.md` are modified but uncommitted. The auto-git daemon may or may not commit them.~~ done (committed by the daemon; history verified)

4. ~~**CONTRIBUTING.md is stale**: Still says `golangci-lint run ./...` — the actual commands in AGENTS.md are `golangci-lint run` (no `./...`). Minor but inconsistent.~~ done at `721cfbd` (CONTRIBUTING.md fully rewritten, v0.3.0-era)

5. ~~**Allocation regression unaddressed**: The old README claimed "Zero allocations on the hot path." We dropped that claim from the new README (correct — it's 13 allocs/op now), but we didn't _fix_ the allocation regression. The `hexEncode` function allocates a `strings.Builder` + `[]byte` on every response, where the old `encodeHex` wrote into a stack-allocated fixed-size array. **This is a real performance regression we introduced.**~~ done at `e0fe51f` (`hexEncodeUint64` — stack-allocated `[16]byte`)

---

## e) WHAT WE SHOULD IMPROVE

### Performance (regression we introduced)

1. ~~**Fix `hexEncode` allocation regression**: The old code used stack-allocated `[16]byte` arrays with zero heap allocation. New `hexEncode` uses `strings.Builder` + `make([]byte, ...)` = 1-2 extra heap allocations per response. Should restore the zero-alloc hex encoding path.~~ done at `e0fe51f`
2. **Reduce `ParseETagList` allocations**: Currently allocates a `make([]ETag, 0)` + `splitRawETags` allocates `[]string`. On the 304 path, this runs on every If-None-Match request. Could use a pre-allocated buffer or `sync.Pool`. — open (still two-pass today)
3. **Benchmark at scale**: No benchmark for large bodies (1 KB, 100 KB, 1 MB). Allocation profile likely changes. — open

### Correctness hardening

4. ~~**Add fuzz tests for `ParseETag` and `ParseETagList`**: These parse untrusted client input (If-None-Match header). Currently only fuzzed indirectly through `FuzzETag`.~~ done at `e0fe51f`
5. **Test edge case: body that is exactly `MaxBufferSize`**: Boundary condition — does the `>` vs `>=` comparison work correctly? — open
6. **Test: handler sets ETag + writes body exceeding buffer with `SkipIfPresent`**: What happens when the handler-set ETag exists but body exceeds buffer? — open
7. **Test: `Skip` returns true for HEAD**: Does Skip bypass the HEAD no-body logic correctly? — open
8. **Test: multiple `Write` calls with `SkipIfPresent`**: Interaction between incremental buffering and handler-set ETag. — open
9. **Test: `Hijack` after `SkipIfPresent` resolve**: Does the flushed state interact correctly? — open

### Architecture

10. ~~**Streaming hash interface**: Allow `hash.Hash` streaming instead of `func([]byte) string`, so large bodies can be hashed incrementally without full buffering.~~ **Won't implement — conflicts with the buffer-and-compare architecture (resolved D3)**
11. ~~**Separate `etag` (middleware) from `entitytag` (domain type)**: The domain type could be its own subpackage for consumers who only want RFC 7232 parsing/comparison without the middleware.~~ done — the split happened differently at `bc5a551` (server/ + client/); the domain type lives in `server/`
12. **Consider `ETag` as `fmt.Stringer` + `fmt.GoStringer`**: Add `GoString()` for debug-friendly `%#v` output. — open (low)

### Spec compliance

13. **RFC 7232 §2.3 `obs-text` support in parser**: The ABNF allows `%x80-FF` (obs-text) in opaque-tags. Current parser accepts any character inside quotes but doesn't validate against the ABNF. — open
14. **RFC 7232 §2.3 backslash warning**: The spec says "servers ought to avoid backslash characters in entity tags." Our parser handles them but we should document that generated tags never contain backslashes. — open (low)
15. **RFC 7232 §2.4 `Vary` header**: A 304 response SHOULD include `Vary` if it would have been sent in a 200. We don't touch `Vary`. — done for the middleware surface (handler-set metadata incl. Vary survives 304 — `TestSpec_RFC7232_NotModifiedMetadata`, 2026-09-10); the client-side Vary sharp edge is documented in `client/doc.go`

### Testing

16. **Race condition test**: Concurrent requests through the same middleware instance. — done for the client (`TestRoundTripConcurrent`, `bc5a551`); the server writer is per-request, whole suite runs under `-race`
17. ~~**Stress test**: 10K requests/second through the middleware.~~ **Won't implement — benchmarks cover the hot path; a fixed-RPS harness adds CI cost without new signal**
18. ~~**Property-based test**: For any body `b`, `ParseETag(computeETag(b).String())` round-trips.~~ done at `e0fe51f` (fuzz round-trip invariants in `entity_tag_fuzz_test.go`)
19. ~~**Table-driven test for all cacheable status codes** (200-299): Currently only tests 200 and 201.~~ done at `c759373` (299/300 boundary + 7 non-cacheable statuses)
20. ~~**Test: handler calls `WriteHeader` multiple times**: Go's `http` package ignores duplicate calls; verify our wrapper does too.~~ done at `c759373` (`wrapper_test.go`)

### Documentation

21. ~~**Migration guide**: Breaking changes need a `docs/migration/v0.2.md` guide.~~ done — written at `e0fe51f`, removed at `8890f8d` (nothing was released; fiction)
22. ~~**Update `docs/review-and-roadmap.md`**: Mark B1/B2/B3/D1/D3/D4/D5/D6/D8 as done. Update priorities.~~ done at `e0fe51f`
23. ~~**CONTRIBUTING.md**: Update commands to match AGENTS.md.~~ done at `721cfbd`
24. **GoDoc examples for `MatchesIfMatch`**: Show the lost-update prevention pattern. — open (README shows it; a GoDoc example does not exist)
25. ~~**Document the `EntityTag` vs `ETag` naming**: The commit message calls it `EntityTag` but the code uses `ETag`. Align.~~ done — code and docs consistently use `ETag`

### Hygiene

26. ~~**Delete `hashUint64HexChars`**: Unused constant.~~ done at `e0fe51f` (consumed by `hexEncodeUint64`)
27. ~~**Remove or update `docs/review-and-roadmap.md`**: Stale.~~ done at `e0fe51f`
28. **`go mod tidy`**: Already verified clean, but should be part of CI. — open (CI downloads and builds; no explicit tidy-drift gate)
29. ~~**Add `//go:generate` directives** if any code generation is planned.~~ **Won't implement — no codegen in this library**
30. ~~**Consider a `CHANGELOG.md` link to migration guide**.~~ **NOT-DO — the migration guide was removed (`8890f8d`); nothing to link**

---

## f) Next 50 things to get done

| #   | Priority | Task                                                                                                                                                  | Effort |
| --- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1   | P0       | ~~Delete unused `hashUint64HexChars` constant~~ done at `e0fe51f`                                                                                     | XS     |
| 2   | P0       | ~~Fix `hexEncode` allocation regression (restore stack-allocated path)~~ done at `e0fe51f`                                                            | S      |
| 3   | P0       | ~~Commit uncommitted `AGENTS.md` + `CHANGELOG.md`~~ done (daemon committed; history verified)                                                         | XS     |
| 4   | P0       | ~~Update stale `docs/review-and-roadmap.md`~~ done at `e0fe51f`                                                                                       | S      |
| 5   | P1       | ~~Add fuzz tests for `ParseETag` / `ParseETagList`~~ done at `e0fe51f`                                                                                | S      |
| 6   | P1       | Add boundary test: body exactly `MaxBufferSize` — open                                                                                                | XS     |
| 7   | P1       | Test: `Skip` + `HEAD` interaction — open                                                                                                              | XS     |
| 8   | P1       | Test: `SkipIfPresent` + buffer overflow interaction — open                                                                                            | XS     |
| 9   | P1       | ~~Test: all 2xx cacheable status codes~~ done at `c759373`                                                                                            | XS     |
| 10  | P1       | ~~Test: handler calls `WriteHeader` multiple times~~ done at `c759373`                                                                                | XS     |
| 11  | P1       | ~~Property test: ETag round-trip (compute → String → Parse)~~ done at `e0fe51f` (fuzz round-trip)                                                     | S      |
| 12  | P1       | ~~Write migration guide (`docs/migration/v0.2.md`)~~ done — written `e0fe51f`, removed `8890f8d` (fiction)                                            | M      |
| 13  | P1       | ~~Update CONTRIBUTING.md commands~~ done at `721cfbd`                                                                                                  | XS     |
| 14  | P2       | Reduce `ParseETagList` allocations via `sync.Pool` or pre-allocation — open                                                                           | M      |
| 15  | P2       | Add benchmarks for large bodies (1 KB, 100 KB, 1 MB) — open                                                                                           | S      |
| 16  | P2       | ~~Streaming hash support (`hash.Hash` interface option)~~ Won't implement — conflicts with buffer-and-compare architecture                            | M      |
| 17  | P2       | GoDoc example for `MatchesIfMatch` (lost-update pattern) — open                                                                                       | XS     |
| 18  | P2       | ~~Align `EntityTag` vs `ETag` naming in commit history / docs~~ done — `ETag` everywhere                                                              | XS     |
| 19  | P2       | Add `GoString()` method to `ETag` for debug output — open (low)                                                                                       | XS     |
| 20  | P2       | Document backslash avoidance in generated tags — open (low)                                                                                           | XS     |
| 21  | P2       | ~~RFC 7232 §2.4: Preserve `Vary` header on 304~~ done — handler metadata incl. Vary survives 304 (`TestSpec_RFC7232_NotModifiedMetadata`, 2026-09-10) | S      |
| 221 | P3       | ~~`If-Modified-Since` support (RFC 7232 §3.3)~~ (typo-row duplicate of #22 — Won't implement server-side)                                             | M      |
| 23  | P3       | ~~`If-Unmodified-Since` support (RFC 7232 §3.4)~~ Won't implement — out of scope                                                                      | M      |
| 24  | P3       | ~~Full precedence chain (RFC 7232 §6)~~ Won't implement — scope boundary (ROADMAP Non-goals)                                                          | M      |
| 25  | P3       | ~~`If-Range` support (RFC 7233)~~ Won't implement — ROADMAP Non-goal                                                                                  | M      |
| 26  | P3       | ~~Integration tests with real `net/http` server~~ done at `9204885` (client; server open — TODO_LIST #15)                                             | S      |
| 27  | P3       | ~~Stress test: 10K req/s~~ Won't implement — benchmarks suffice                                                                                       | S      |
| 28  | P3       | ~~Race condition test: concurrent requests~~ done at `bc5a551` (client `TestRoundTripConcurrent`; suite runs under -race)                             | S      |
| 29  | P3       | ~~GitHub Actions CI workflow~~ done at `37b68b3` (green since `cf98ab5`)                                                                              | S      |
| 30  | P3       | ~~Version tag `v0.2.0`~~ done — v0.1.0 first (`3a7d532` era), v0.2.0 at `be19640`                                                                     | XS     |
| 31  | P3       | ~~GoDoc badge / pkg.go.dev submission~~ done — live since v0.1.1                                                                                      | XS     |
| 32  | P3       | ~~Consider subpackage split: `entitytag` vs `middleware`~~ done at `bc5a551` (server/ + client/ split instead)                                        | M      |
| 33  | P3       | ABNF validation for `obs-text` in `ParseETag` — open                                                                                                  | S      |
| 34  | P3       | ~~`//go:generate` directives if codegen planned~~ Won't implement — no codegen                                                                        | S      |
| 35  | P3       | ~~Allocation profiling and optimization pass~~ done at `e0fe51f` (zero-alloc hex; remaining allocs documented)                                        | M      |
| 36  | P3       | ~~Consider `Skip` as a `func(*http.Request) bool` interface for composability~~ done — `Skip` IS `func(*http.Request) bool` since `15114b3`           | S      |
| 37  | P3       | ~~Add `ETagConfig` defaults documentation to GoDoc~~ done at `b9eb32e` (README config table + field docs)                                             | XS     |
| 38  | P3       | ~~Consider `WithStrength(Strength)` builder pattern~~ Won't implement — struct config is the house style                                              | S      |
| 39  | P3       | ~~Add `Last-Modified` header support (companion to ETag)~~ Won't implement server-side; client second-validator idea lives in ROADMAP Theme 1         | M      |
| 40  | P3       | ~~Consider `Cache-Control` integration~~ Won't implement — callers own Cache-Control (guidance TODO_LIST #9)                                          | M      |
| 41  | P3       | ~~Add `ETag.IsEmpty()` vs `ETag.IsValid()` distinction~~ done — `IsValid()` shipped; zero-value is the empty case                                     | XS     |
| 42  | P3       | ~~Document FNV-64a collision probability in GoDoc~~ done — documented in `entity_tag.go` + README                                                     | XS     |
| 43  | P3       | Add `BenchmarkParseETagList` with realistic header sizes — open                                                                                       | XS     |
| 44  | P3       | ~~Consider `ETag` as `json.Marshaler`/`json.Unmarshaler`~~ Won't implement — no consumer demand                                                       | S      |
| 45  | P3       | ~~Add `Equal(ETag) bool` convenience method (defaults to weak)~~ Won't implement — `WeakEqual`/`StrongEqual` are explicit                             | XS     |
| 46  | P3       | Consider `Strength.String()` method for logging — open (low)                                                                                          | XS     |
| 47  | P3       | ~~Add `docs/architecture.md` with design decisions~~ Won't implement — AGENTS.md carries the architecture; `docs/decisions/` carries ADRs             | S      |
| 48  | P3       | ~~Consider `ETagHeader` custom type for `http.Header` integration~~ Won't implement — plain string constants suffice                                  | M      |
| 49  | P3       | ~~Add negative tests: nil `HashFunc`, nil `Skip`~~ done at `15114b3` (default-fn tests; Skip true/false paths)                                        | XS     |
| 50  | P3       | ~~Consider WebSocket upgrade detection (skip ETag)~~ Won't implement — the `Skip` predicate covers opt-out                                            | S      |

---

## g) Questions I CANNOT figure out myself

### 1. Should FNV-64a default to `Strong` or `Weak`?

RFC 7232 §2.1 says a strong validator must be "unique across all versions of all representations." FNV-64a has a birthday-bound collision probability of ~4.3 billion distinct bodies. For a small API, this is effectively impossible. For a large CDN-scale system, it's a real risk. **Should we default to `Weak` (honest, safe, spec-purist) or `Strong` (pragmatic, what most users expect)?** This changes the library's default behavior and user perception.

_**Resolved:** `Strong` kept (pragmatic; tradeoff documented in README's Strong-vs-Weak section). Standing revisit note: ROADMAP.md Open Questions #4._

### 2. Should this library grow into a full conditional-request library, or stay an ETag-generation middleware?

Adding `If-Match` middleware interception, `If-Modified-Since`, precedence chains (RFC 7232 §6) would make this a comprehensive conditional-request library — but it would also increase scope, complexity, and the dependency surface. The alternative is staying focused on ETag generation + 304 handling and documenting that application code handles unsafe-method preconditions via the exported helpers. **Which direction do you want?**

_**Resolved:** stayed focused on generation + 304 — and grew on the CLIENT side instead (conditional-GET transport, v0.2.0). Server-side interception remains a ROADMAP Non-goal._

### 3. Should we split `entity_tag.go` into a separate subpackage?

The `ETag` type, `Strength` enum, and comparison functions are useful independently of the middleware. Consumers who only want RFC 7232 entity-tag parsing/comparison (e.g., for a cache layer or proxy) currently must import the full middleware package. Splitting into `github.com/larsartmann/go-etag/entitytag` would give them a minimal dependency, but it adds module complexity and import verbosity. **Do you want a subpackage split, or keep everything flat?**

_**Resolved:** split happened differently at `bc5a551` — server/ + client/ packages; the domain type lives in `server/` (package `etag`). Whether the client should adopt it is an open ROADMAP question (Theme 2)._

---

## Verification Snapshot

| Check                          | Result                      |
| ------------------------------ | --------------------------- |
| `go build ./...`               | PASS                        |
| `go vet ./...`                 | PASS                        |
| `go test -race -count=1 ./...` | PASS (63 tests)             |
| `golangci-lint run`            | 0 issues                    |
| `golangci-lint fmt`            | PASS                        |
| Fuzz (20s)                     | 4.9M execs, 0 crashes       |
| Coverage                       | 92.1%                       |
| Uncommitted                    | `AGENTS.md`, `CHANGELOG.md` |

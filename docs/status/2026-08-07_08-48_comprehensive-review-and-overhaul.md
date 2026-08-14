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

### D7 — If-Match / 412 support

- **What was done**: Exported `MatchesIfMatch` helper using `StrongEqual` — applications can call it in their handlers.
- **What was NOT done**: No middleware-level If-Match interception (no automatic 412 response). No RFC 7232 §6 precedence chain. This was a deliberate scope decision (ETag generation ≠ conditional request evaluation for unsafe methods), but it should be documented as a boundary.

### D3 — Hash flexibility

- **What was done**: `HashFunc func([]byte) string` allows arbitrary opaque values.
- **What was NOT done**: No streaming hash support (hash as you write, not after full buffer). No `hash.Hash` interface option. The `[]byte` input means the entire body must be in memory before hashing.

---

## c) NOT STARTED

1. **RFC 7232 §3.3 If-Modified-Since** — no date-based validator support.
2. **RFC 7232 §3.4 If-Unmodified-Since** — no date-based validator support.
3. **RFC 7232 §6 Precedence ordering** — no conditional-request precedence chain.
4. **RFC 7233 If-Range** — no range request support.
5. **Migration guide** — breaking API change with no migration doc.
6. **Fuzz tests for `entity_tag.go`** — `ParseETag`, `ParseETagList`, `MatchesIfMatch` have no fuzz coverage.
7. **Streaming hash** — hash body as it arrives instead of buffering then hashing.
8. **Integration tests with real `net/http` server** — all tests use `httptest.Recorder`.
9. **CI/CD pipeline** — no GitHub Actions workflow.
10. **Version tagging** — no `git tag v0.2.0` for the breaking change.
11. **GoDoc badge / pkg.go.dev integration**.
12. **Performance optimization** — 13 allocs/op on the hot path (see section e).

---

## d) TOTALLY FUCKED UP

### Nothing is catastrophically broken, but:

1. **Unused constant `hashUint64HexChars`**: Defined in `etag.go:23` as `hashUint64Bytes * 2` but never referenced anywhere. The old `encodeHex` function that used the related `etagWeakLen`/`etagStrongLen` constants was removed, but this constant survived. `golangci-lint` didn't catch it (the `unused` linter may not flag package-level constants in all configs, or the `_test` exclusion is too broad). **Must delete.**

2. **`docs/review-and-roadmap.md` is now stale**: It describes B1/B2/B3 as open work items with "should be applied after reconciling." All three are done. The roadmap priorities, open questions, and ETag type proposal are all resolved. This file misleads any future reader.

3. **No commit for the documentation updates**: `AGENTS.md` and `CHANGELOG.md` are modified but uncommitted. The auto-git daemon may or may not commit them.

4. **CONTRIBUTING.md is stale**: Still says `golangci-lint run ./...` — the actual commands in AGENTS.md are `golangci-lint run` (no `./...`). Minor but inconsistent.

5. **Allocation regression unaddressed**: The old README claimed "Zero allocations on the hot path." We dropped that claim from the new README (correct — it's 13 allocs/op now), but we didn't _fix_ the allocation regression. The `hexEncode` function allocates a `strings.Builder` + `[]byte` on every response, where the old `encodeHex` wrote into a stack-allocated fixed-size array. **This is a real performance regression we introduced.**

---

## e) WHAT WE SHOULD IMPROVE

### Performance (regression we introduced)

1. **Fix `hexEncode` allocation regression**: The old code used stack-allocated `[16]byte` arrays with zero heap allocation. New `hexEncode` uses `strings.Builder` + `make([]byte, ...)` = 1-2 extra heap allocations per response. Should restore the zero-alloc hex encoding path.
2. **Reduce `ParseETagList` allocations**: Currently allocates a `make([]ETag, 0)` + `splitRawETags` allocates `[]string`. On the 304 path, this runs on every If-None-Match request. Could use a pre-allocated buffer or `sync.Pool`.
3. **Benchmark at scale**: No benchmark for large bodies (1 KB, 100 KB, 1 MB). Allocation profile likely changes.

### Correctness hardening

4. **Add fuzz tests for `ParseETag` and `ParseETagList`**: These parse untrusted client input (If-None-Match header). Currently only fuzzed indirectly through `FuzzETag`.
5. **Test edge case: body that is exactly `MaxBufferSize`**: Boundary condition — does the `>` vs `>=` comparison work correctly?
6. **Test: handler sets ETag + writes body exceeding buffer with `SkipIfPresent`**: What happens when the handler-set ETag exists but body exceeds buffer?
7. **Test: `Skip` returns true for HEAD**: Does Skip bypass the HEAD no-body logic correctly?
8. **Test: multiple `Write` calls with `SkipIfPresent`**: Interaction between incremental buffering and handler-set ETag.
9. **Test: `Hijack` after `SkipIfPresent` resolve**: Does the flushed state interact correctly?

### Architecture

10. **Streaming hash interface**: Allow `hash.Hash` streaming instead of `func([]byte) string`, so large bodies can be hashed incrementally without full buffering.
11. **Separate `etag` (middleware) from `entitytag` (domain type)**: The domain type could be its own subpackage for consumers who only want RFC 7232 parsing/comparison without the middleware.
12. **Consider `ETag` as `fmt.Stringer` + `fmt.GoStringer`**: Add `GoString()` for debug-friendly `%#v` output.

### Spec compliance

13. **RFC 7232 §2.3 `obs-text` support in parser**: The ABNF allows `%x80-FF` (obs-text) in opaque-tags. Current parser accepts any character inside quotes but doesn't validate against the ABNF.
14. **RFC 7232 §2.3 backslash warning**: The spec says "servers ought to avoid backslash characters in entity tags." Our parser handles them but we should document that generated tags never contain backslashes.
15. **RFC 7232 §2.4 `Vary` header**: A 304 response SHOULD include `Vary` if it would have been sent in a 200. We don't touch `Vary`.

### Testing

16. **Race condition test**: Concurrent requests through the same middleware instance.
17. **Stress test**: 10K requests/second through the middleware.
18. **Property-based test**: For any body `b`, `ParseETag(computeETag(b).String())` round-trips.
19. **Table-driven test for all cacheable status codes** (200-299): Currently only tests 200 and 201.
20. **Test: handler calls `WriteHeader` multiple times**: Go's `http` package ignores duplicate calls; verify our wrapper does too.

### Documentation

21. **Migration guide**: Breaking changes need a `docs/migration/v0.2.md` guide.
22. **Update `docs/review-and-roadmap.md`**: Mark B1/B2/B3/D1/D3/D4/D5/D6/D8 as done. Update priorities.
23. **CONTRIBUTING.md**: Update commands to match AGENTS.md.
24. **GoDoc examples for `MatchesIfMatch`**: Show the lost-update prevention pattern.
25. **Document the `EntityTag` vs `ETag` naming**: The commit message calls it `EntityTag` but the code uses `ETag`. Align.

### Hygiene

26. **Delete `hashUint64HexChars`**: Unused constant.
27. **Remove or update `docs/review-and-roadmap.md`**: Stale.
28. **`go mod tidy`**: Already verified clean, but should be part of CI.
29. **Add `//go:generate` directives** if any code generation is planned.
30. **Consider a `CHANGELOG.md` link to migration guide**.

---

## f) Next 50 things to get done

| #   | Priority | Task                                                                        | Effort |
| --- | -------- | --------------------------------------------------------------------------- | ------ |
| 1   | P0       | Delete unused `hashUint64HexChars` constant                                 | XS     |
| 2   | P0       | Fix `hexEncode` allocation regression (restore stack-allocated path)        | S      |
| 3   | P0       | Commit uncommitted `AGENTS.md` + `CHANGELOG.md`                             | XS     |
| 4   | P0       | Update stale `docs/review-and-roadmap.md`                                   | S      |
| 5   | P1       | Add fuzz tests for `ParseETag` / `ParseETagList`                            | S      |
| 6   | P1       | Add boundary test: body exactly `MaxBufferSize`                             | XS     |
| 7   | P1       | Test: `Skip` + `HEAD` interaction                                           | XS     |
| 8   | P1       | Test: `SkipIfPresent` + buffer overflow interaction                         | XS     |
| 9   | P1       | Test: all 2xx cacheable status codes                                        | XS     |
| 10  | P1       | Test: handler calls `WriteHeader` multiple times                            | XS     |
| 11  | P1       | Property test: ETag round-trip (compute → String → Parse)                   | S      |
| 12  | P1       | Write migration guide (`docs/migration/v0.2.md`)                            | M      |
| 13  | P1       | Update CONTRIBUTING.md commands                                             | XS     |
| 14  | P2       | Reduce `ParseETagList` allocations via `sync.Pool` or pre-allocation        | M      |
| 15  | P2       | Add benchmarks for large bodies (1 KB, 100 KB, 1 MB)                        | S      |
| 16  | P2       | Streaming hash support (`hash.Hash` interface option)                       | M      |
| 17  | P2       | GoDoc example for `MatchesIfMatch` (lost-update pattern)                    | XS     |
| 18  | P2       | Align `EntityTag` vs `ETag` naming in commit history / docs                 | XS     |
| 19  | P2       | Add `GoString()` method to `ETag` for debug output                          | XS     |
| 20  | P2       | Document backslash avoidance in generated tags                              | XS     |
| 21  | P2       | RFC 7232 §2.4: Preserve `Vary` header on 304                                | S      |
| 221 | P3       | `If-Modified-Since` support (RFC 7232 §3.3)                                 | M      |
| 23  | P3       | `If-Unmodified-Since` support (RFC 7232 §3.4)                               | M      |
| 24  | P3       | Full precedence chain (RFC 7232 §6)                                         | M      |
| 25  | P3       | `If-Range` support (RFC 7233)                                               | M      |
| 26  | P3       | Integration tests with real `net/http` server                               | S      |
| 27  | P3       | Stress test: 10K req/s                                                      | S      |
| 28  | P3       | Race condition test: concurrent requests                                    | S      |
| 29  | P3       | GitHub Actions CI workflow                                                  | S      |
| 30  | P3       | Version tag `v0.2.0`                                                        | XS     |
| 31  | P3       | GoDoc badge / pkg.go.dev submission                                         | XS     |
| 32  | P3       | Consider subpackage split: `entitytag` vs `middleware`                      | M      |
| 33  | P3       | ABNF validation for `obs-text` in `ParseETag`                               | S      |
| 34  | P3       | `//go:generate` directives if codegen planned                               | S      |
| 35  | P3       | Allocation profiling and optimization pass                                  | M      |
| 36  | P3       | Consider `Skip` as a `func(*http.Request) bool` interface for composability | S      |
| 37  | P3       | Add `ETagConfig` defaults documentation to GoDoc                            | XS     |
| 38  | P3       | Consider `WithStrength(Strength)` builder pattern                           | S      |
| 39  | P3       | Add `Last-Modified` header support (companion to ETag)                      | M      |
| 40  | P3       | Consider `Cache-Control` integration                                        | M      |
| 41  | P3       | Add `ETag.IsEmpty()` vs `ETag.IsValid()` distinction                        | XS     |
| 42  | P3       | Document FNV-64a collision probability in GoDoc                             | XS     |
| 43  | P3       | Add `BenchmarkParseETagList` with realistic header sizes                    | XS     |
| 44  | P3       | Consider `ETag` as `json.Marshaler`/`json.Unmarshaler`                      | S      |
| 45  | P3       | Add `Equal(ETag) bool` convenience method (defaults to weak)                | XS     |
| 46  | P3       | Consider `Strength.String()` method for logging                             | XS     |
| 47  | P3       | Add `docs/architecture.md` with design decisions                            | S      |
| 48  | P3       | Consider `ETagHeader` custom type for `http.Header` integration             | M      |
| 49  | P3       | Add negative tests: nil `HashFunc`, nil `Skip`                              | XS     |
| 50  | P3       | Consider WebSocket upgrade detection (skip ETag)                            | S      |

---

## g) Questions I CANNOT figure out myself

### 1. Should FNV-64a default to `Strong` or `Weak`?

RFC 7232 §2.1 says a strong validator must be "unique across all versions of all representations." FNV-64a has a birthday-bound collision probability of ~4.3 billion distinct bodies. For a small API, this is effectively impossible. For a large CDN-scale system, it's a real risk. **Should we default to `Weak` (honest, safe, spec-purist) or `Strong` (pragmatic, what most users expect)?** This changes the library's default behavior and user perception.

### 2. Should this library grow into a full conditional-request library, or stay an ETag-generation middleware?

Adding `If-Match` middleware interception, `If-Modified-Since`, precedence chains (RFC 7232 §6) would make this a comprehensive conditional-request library — but it would also increase scope, complexity, and the dependency surface. The alternative is staying focused on ETag generation + 304 handling and documenting that application code handles unsafe-method preconditions via the exported helpers. **Which direction do you want?**

### 3. Should we split `entity_tag.go` into a separate subpackage?

The `ETag` type, `Strength` enum, and comparison functions are useful independently of the middleware. Consumers who only want RFC 7232 entity-tag parsing/comparison (e.g., for a cache layer or proxy) currently must import the full middleware package. Splitting into `github.com/larsartmann/go-etag/entitytag` would give them a minimal dependency, but it adds module complexity and import verbosity. **Do you want a subpackage split, or keep everything flat?**

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

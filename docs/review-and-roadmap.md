# go-etag — Review & Roadmap

_Point-in-time audit of go-etag against RFC 7232, conducted 2026-08-07. All P0–P3 items below have been implemented. This document serves as the historical record of findings and decisions._

---

## Baseline (post-overhaul, verified 2026-08-07)

| Check                 | Result                          |
| --------------------- | ------------------------------- |
| `go test -race ./...` | PASS (63 tests, 92.1% coverage) |
| `go vet ./...`        | clean                           |
| `golangci-lint run`   | 0 issues                        |
| `golangci-lint fmt`   | clean                           |
| Fuzz `FuzzETag` (30s) | 14.8M executions, **0 crashes** |
| Dependencies          | 1 (`go-error-family`)           |
| LOC (Go)              | ~1400 across 10 files           |

The foundation is well-built: clean flat-package structure, strong lint culture, dedicated `ETag` type with RFC 7232 §2.3.2 strong and weak comparison functions, quote-aware list parsing, graceful overflow/hijack/flush handling, honest strong/weak validator defaults.

---

## Spec compliance (RFC 7232)

### Implemented correctly

- **§2.3.2 Strong comparison** — `ETag.StrongEqual`: both must be strong validators with matching opaque-tags. Used by `MatchesIfMatch`.
- **§2.3.2 Weak comparison** — `ETag.WeakEqual`: opaque-tags match regardless of strength. Used by `MatchesIfNoneMatch` and the middleware's `If-None-Match` path.
- **§2.3 ABNF** — `ParseETag` validates the `entity-tag = [ weak ] opaque-tag` grammar. `ParseETagList` respects commas inside quotes and handles escaped quotes (`\"`).
- **§2.1 Strength** — `Strength` enum (`Strong`/`Weak`) replaces the old `bool` field, making validator semantics explicit and type-safe.
- **§3.1 `If-Match`** — `MatchesIfMatch` helper implements strong comparison for lost-update prevention. Applications call this in their handlers for unsafe methods.
- **§3.2 `If-None-Match`** — middleware supports `*`, list values, and multiple header lines (RFC 9110 §5.2 combination). `MatchesIfNoneMatch` helper available for manual use.
- **§4.1 304 Not Modified** — returns empty body with ETag header, strips `Content-Length`.

### Not implemented (scope gaps, documented)

- **§3.3 `If-Modified-Since`** / **§3.4 `If-Unmodified-Since`** — no date-based validators. Out of scope for an ETag-focused library.
- **§6 Precedence ordering** — no full conditional-request precedence chain. The library handles `If-None-Match` only; applications own the rest.

---

## Completed work

### Bugs (RFC violations) — all fixed

| ID | Description                           | Severity | Fix location                                                |
| -- | ------------------------------------- | -------- | ----------------------------------------------------------- |
| B1 | Zero-value config unbounded buffering | P0       | `newETagWriter` clamps `MaxBufferSize <= 0` to default      |
| B2 | HEAD forwards message body            | P0       | `flush` sets `Content-Length` and skips body write for HEAD |
| B3 | 304 leaks `Content-Length`            | P1       | `flush` deletes `Content-Length` before writing 304         |

### Design improvements — all shipped

| ID | Description                                | Severity | What changed                                              |
| -- | ------------------------------------------ | -------- | --------------------------------------------------------- |
| D1 | Overwrites handler-set ETags               | P1       | `SkipIfPresent bool` config field respects handler ETags  |
| D2 | FNV-64a claims "strong" dishonestly        | P2       | `Strength` enum with documented collision tradeoff        |
| D3 | `HashFunc` locked to 64-bit                | P2       | Signature changed to `func([]byte) string` (opaque value) |
| D4 | `errors.go` classifies unrelated sentinels | P2       | Trimmed to `ErrNotSupported`, `ErrAbortHandler` only      |
| D5 | `wroteHeader`/`headerWritten` confusing    | P3       | Renamed to `headerBuffered`/`headerCommitted`             |
| D6 | No per-route skip predicate                | P3       | `Skip func(*http.Request) bool` config field              |
| D7 | No `If-Match` / 412 support                | P3       | `MatchesIfMatch` helper with strong comparison            |
| D8 | Dead code and lint nits                    | P3       | Removed dead helper, simplified `newTestRequest`          |

### Foundational refactor

- **Dedicated `ETag` type** (`entity_tag.go`) — `Strength` enum, `NewETag`, `ParseETag`, `ParseETagList`, `StrongEqual`, `WeakEqual`, `MatchesIfNoneMatch`, `MatchesIfMatch`. This type makes both RFC 7232 §2.3.2 comparison functions first-class named methods, enabling correct strong comparison for `If-Match`.

---

## File layout

| File            | Purpose                                                   |
| --------------- | --------------------------------------------------------- |
| `entity_tag.go` | `ETag` type, `Strength` enum, comparison methods, parsers |
| `etag.go`       | Middleware, `ETagConfig`, `etagWriter`                    |
| `wrapper.go`    | `responseWrapper` (shared ResponseWriter wrapping)        |
| `errors.go`     | Error codes + `go-error-family` classification            |
| `hex.go`        | Zero-alloc hex encoding for FNV-64a                       |
| `middleware.go` | `Middleware` type alias                                   |
| `doc.go`        | Package-level documentation                               |

---

## Open questions

1. **Scope:** Is this purely an ETag-generation middleware, or should it grow into a full conditional-request library (`If-Match`, `If-Modified-Since`, precedence)? The answer determines whether §6 precedence ordering is in scope. Currently: focused on ETag generation + `If-None-Match` handling, with `MatchesIfMatch`/`MatchesIfNoneMatch` helpers for applications that need more.

2. **Hash defaults:** Should FNV-64a default to `Weak` strength (honest) or stay `Strong` (fast, pragmatic)? Currently `Strong` with documented collision tradeoff. A collision would produce a stale 304 (effectively data corruption), but the birthday bound (~4.3 billion bodies) is astronomically unlikely for practical use.

3. **Subpackage split:** Should `entity_tag.go` move to a separate subpackage? Currently: flat package is simpler and the types are tightly coupled to the middleware.

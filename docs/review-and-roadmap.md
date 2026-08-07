# go-etag — Review, Roadmap & TODOs

_Point-in-time audit of go-etag against RFC 7232. Findings, design proposals, and prioritized work items to make this the best ETag library for Go._

---

## Baseline (verified 2026-08-07)

| Check                 | Result                          |
| --------------------- | ------------------------------- |
| `go test -race ./...` | PASS                            |
| `go vet ./...`        | clean                           |
| `golangci-lint run`   | 1 trivial nit (`unparam`)       |
| Fuzz `FuzzETag` (30s) | 14.8M executions, **0 crashes** |
| Dependencies          | 1 (`go-error-family`)           |
| LOC (Go)              | ~1400 across 8 files            |

The foundation is well-built: clean flat-package structure, strong lint culture, correct RFC 7232 §2.3.2 weak comparison, quote-aware list parsing, graceful overflow/hijack/flush handling, zero-allocation hex encoding.

---

## Spec compliance (RFC 7232)

### Implemented correctly

- **§2.3.2 Weak comparison** for `If-None-Match` — ignores `W/` prefix via `stripWeakPrefix`.
- **§2.3 ABNF** — `parseETagList` respects commas inside quotes and handles escaped quotes (`\"`).
- **§4.1 304 Not Modified** — returns empty body with ETag header.
- **§3.2 `If-None-Match`** — supports `*`, list values, and multiple header lines (RFC 9110 §5.2 combination).

### Not implemented (scope gaps, not bugs)

- **§3.1 `If-Match`** — no strong-comparison path, no 412 Precondition Failed for lost-update protection.
- **§3.3 `If-Modified-Since`** / **§3.4 `If-Unmodified-Since`** — no date-based validators.
- **§6 Precedence ordering** — no conditional-request precedence chain.
- **§2.1 Strong validator guarantees** — FNV-64a is used and claimed strong, but see D2 below.

---

## Bugs (RFC violations)

These are confirmed against the current source. Each is small, surgical, and unambiguous.

### B1 — Zero-value config causes unbounded buffering [P0]

**Where:** `etag.go` `newETagWriter` + `Write`

**Problem:** `ETagConfig{}` (zero value) sets `MaxBufferSize: 0`. In `Write`, the overflow guard `w.maxBufferSize > 0 && …` is **false**, so the middleware buffers the **entire** response body in RAM — a memory exhaustion vector for any caller who forgets `Validate()`.

`Validate()` catches a non-positive size, but `ETag()` never calls `Validate()`.

**Fix:** Clamp `MaxBufferSize <= 0` to `defaultETagMaxBufferSize` inside `newETagWriter`. A zero-value config must be safe to use.

```go
maxBufferSize := cfg.MaxBufferSize
if maxBufferSize <= 0 {
    maxBufferSize = defaultETagMaxBufferSize
}
```

### B2 — HEAD forwards a message body [P0]

**Where:** `etag.go` `flush`

**Problem:** A non-304 `HEAD` request runs the same body-write tail as `GET`, sending the body bytes to the client. RFC 7230 §3.3: a HEAD response **MUST NOT** have a message body.

**Fix:** For `HEAD`, set `Content-Length` from the buffered length _before_ committing the header, then skip the `Write`:

```go
if req.Method == http.MethodHead {
    w.Header().Set(headerContentLength, strconv.Itoa(len(w.body)))
}
w.writeHeaderToUnderlying()
if req.Method == http.MethodHead {
    return
}
```

### B3 — 304 leaks Content-Length [P1]

**Where:** `etag.go` `flush`, the 304 branch

**Problem:** The 304 path returns without deleting body-length metadata a handler may have attached. RFC 7232 §4.1: a 304 carries no message body. The existing `TestETag_304_ExcludesContentLength` passes only because `httptest.Recorder` never auto-sets it.

**Fix:**

```go
w.Header().Del(headerContentLength)
w.ResponseWriter.WriteHeader(http.StatusNotModified)
```

---

## Design gaps & improvements

### D1 — Overwrites handler-set ETags [P1]

**Problem:** A handler providing a true strong validator (e.g. a database revision number) gets it clobbered by a weaker body hash. RFC 7232 §2.1: "only the service author knows the best validator for a resource."

**Recommendation:** Add `SkipIfPresent bool` (default: respect an existing `ETag` header set by the handler).

### D2 — FNV-64a claims "strong" but isn't collision-proof [P2]

**Problem:** A collision produces a silent stale 304 (effectively data corruption). FNV-64a birthday bound is ~4.3 billion bodies. The library defaults to "strong" ETags, but a 64-bit non-cryptographic hash is a weak validator in spirit.

**Recommendation:** Offer a stronger hash option (e.g. SHA-256 truncated). Document the collision tradeoff explicitly. Consider defaulting to `Weak` when using FNV.

### D3 — `HashFunc func([]byte) uint64` is locked to 64-bit [P2]

**Problem:** Can't plug in SHA-256 (256-bit) or any algorithm with a different output width. The function signature constrains all custom hashers to 64 bits.

**Recommendation:** Consider `HashFunc func([]byte) string` (returns the opaque-tag content directly) for flexibility, or a streaming `hash.Hash64`/`hash.Hash` interface.

### D4 — `errors.go` classifies unrelated stdlib sentinels [P2]

**Problem:** `RegisterErrorClassifications` registers `http.ErrNoCookie`, `http.ErrNoLocation`, `http.ErrSkipAltProtocol` — none of which this ETag library ever returns. This is scope creep and a split brain: why does an ETag library classify cookie/location errors?

**Recommendation:** Trim to only the sentinels this library actually returns (`http.ErrNotSupported`, `http.ErrAbortHandler`).

### D5 — `wroteHeader` vs `headerWritten` naming is confusing [P3]

**Problem:** Two near-synonym field names (`wroteHeader`, `headerWritten`) meaning different things ("header was buffered/captured" vs "header was committed to the underlying writer"). Readability landmine.

**Recommendation:** Rename to `headerBuffered` / `headerCommitted`.

### D6 — No per-route `Skip` predicate [P3]

**Problem:** Can't opt out of ETag generation per request (huge streaming endpoints, already-compressed assets, endpoints with known validators).

**Recommendation:** Add `Skip func(*http.Request) bool` to `ETagConfig`.

### D7 — No `If-Match` / 412 support [P3]

**Problem:** Only `If-None-Match` → 304 is handled. No `If-Match` → 412 for unsafe methods (lost-update protection), no precedence ordering per §6.

**Recommendation:** Defensible scope for an ETag-generation library. Document the boundary clearly. If added, requires the dedicated `ETag` type (see proposal below) for strong comparison.

### D8 — Dead code and lint nits [P3]

**Problem:** `assertBodyContains` in `testutil_test.go:130` is unused. `newTestRequest` triggers `unparam` (path always `/`).

**Recommendation:** Remove dead helper; simplify `newTestRequest` signature.

---

## Proposal: Dedicated `ETag` type [foundational]

The current `string`-everywhere design is **structurally blocking a spec feature**: RFC 7232 §2.3.2 defines **two** comparison functions, but `stripWeakPrefix(a) == stripWeakPrefix(b)` can only express weak comparison. **Strong comparison is impossible** — yet it's _required_ for `If-Match` → 412.

A dedicated type makes both comparisons first-class, named methods that map 1:1 to the spec.

### What it buys

| Today (`string`)                            | With `ETag` type                                        |
| ------------------------------------------- | ------------------------------------------------------- |
| `stripWeakPrefix(etag)` scattered           | `e.IsWeak()` — one place                                |
| `etagInList(list, etag)` (weak only)        | `e.WeakEqual(o)` _and_ `e.StrongEqual(o)`               |
| `computeETag() string` — any string allowed | `NewETag(opaque)` — ABNF-validated at construction      |
| `Weak bool` in config (smell)               | `Strength` enum (`Strong`/`Weak`) — spec vocabulary     |
| `parseETagList → []string`                  | `→ []ETag` — can't confuse a header fragment with a tag |

### Proposed API

```go
// entity_tag.go

type Strength int

const (
    Strong Strength = iota
    Weak
)

// ETag is an RFC 7232 entity-tag: an opaque validator with a strength.
type ETag struct {
    opaque   string   // unquoted content, no W/ prefix
    strength Strength
}

func NewETag(opaque string, strength Strength) ETag   // constructs a valid tag
func (e ETag) String() string                          // wire form: W/"abc" or "abc"
func (e ETag) IsWeak() bool
func (e ETag) Opaque() string                          // unquoted content for comparison
func (e ETag) StrongEqual(o ETag) bool                 // §2.3.2: both strong + opaque match
func (e ETag) WeakEqual(o ETag) bool                   // §2.3.2: opaque match, ignore strength
func ParseETag(s string) (ETag, error)                 // one tag from wire form
func ParseETagList(header string) ([]ETag, error)      // comma-separated, quote-aware
```

### Properties

- **Allocation-neutral:** struct of `string` + `int` — no heavier than the `string` it replaces.
- **Parse-once:** `parseETagList` already allocates a slice today; returning `[]ETag` costs nothing extra.
- **Strong typing:** can't accidentally pass a raw header fragment where a tag is expected.
- **Foundation for If-Match:** `StrongEqual` is the prerequisite for D7.

### Migration impact

- `ETagConfig.Weak bool` → `ETagConfig.Strength Strength`.
- `computeETag() string` → `computeETag() ETag`.
- `matchesIfNoneMatch` uses `WeakEqual`.
- `parseETagList` → `ParseETagList` (exported, returns `[]ETag`).
- New file: `entity_tag.go` (domain type). Keep middleware in `etag.go`.

### File split

| File            | Purpose                                                   |
| --------------- | --------------------------------------------------------- |
| `entity_tag.go` | `ETag` type, `Strength` enum, comparison methods, parsers |
| `etag.go`       | Middleware, `ETagConfig`, `etagWriter`                    |
| `wrapper.go`    | `responseWrapper` (unchanged)                             |
| `errors.go`     | Error codes + classification                              |
| `hex.go`        | Hex encoding (unchanged)                                  |
| `middleware.go` | `Middleware` type alias (unchanged)                       |

---

## Prioritized roadmap (Pareto order)

| Priority | Item                                       | Effort | Impact                       |
| -------- | ------------------------------------------ | ------ | ---------------------------- |
| **P0**   | B1 — Zero-value config unbounded buffering | XS     | 🔴 Memory DoS fix            |
| **P0**   | B2 — HEAD forwards message body            | XS     | 🔴 RFC compliance            |
| **P1**   | B3 — 304 leaks Content-Length              | XS     | 🟠 RFC compliance            |
| **P1**   | D8 — Dead code + lint cleanup              | XS     | 🟢 Hygiene                   |
| **P1**   | D5 — Rename `wroteHeader`/`headerWritten`  | S      | 🟢 Readability               |
| **P1**   | D1 — `SkipIfPresent` config                | S      | 🟠 Don't harm users          |
| **P2**   | ETag type extraction (proposal above)      | M      | 🟠 Foundation for D2/D3/D7   |
| **P2**   | D4 — Trim `errors.go` scope                | XS     | 🟢 Clarity                   |
| **P2**   | D3 — Flexible `HashFunc` signature         | S      | 🟠 Extensibility             |
| **P2**   | D2 — Honest strong/weak hash defaults      | S      | 🟠 Correctness               |
| **P3**   | D6 — `Skip` predicate                      | XS     | 🟢 Flexibility               |
| **P3**   | D7 — `If-Match` / 412 support              | M      | 🟢 Feature (needs ETag type) |
| **P3**   | BDD specs (Ginkgo) mapping RFC sections    | M      | 🟢 Spec confidence           |

### Execution order rationale

1. **B1/B2/B3** first — correctness, ~15 lines total, highest value, lowest risk.
2. **D8/D5** alongside — instant cleanup, zero API risk.
3. **D1** `SkipIfPresent` — the single biggest "respect the spec / don't harm users" win.
4. **ETag type** — foundational refactor that D2, D3, D7, and BDD all build on.
5. **D4** trim errors — clarity.
6. **D3/D2** hash flexibility + honest defaults.
7. **D6/D7** features + scope.
8. **BDD** — optional differentiator; this library is all about spec correctness.

---

## Open questions

1. **Scope:** Is this purely an ETag-generation middleware, or should it grow into a full conditional-request library (`If-Match`, `If-Modified-Since`, precedence)? The answer determines whether D7 is in or out.
2. **Hash defaults:** Should FNV-64a default to `Weak` strength (honest) or stay `Strong` (fast, pragmatic)? This is a correctness-vs-performance tradeoff.
3. **Concurrency note:** At time of audit, another process was actively editing `etag.go` / `etag_test.go` (adding `OnError`, `ErrInvalidConfig`, `ErrCodeHashWriteFailed`). B1/B2/B3 fixes should be applied after reconciling that work. Note: `etag.go:57-65` has a duplicated `defaultETagHash` doc comment (merge artifact) to fix during reconciliation.

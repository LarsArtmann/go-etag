# go-etag — AGENTS.md

## Hard Constraints (Will Break Your Code)

### Allowed Dependencies

`depguard` allows `$gostd`, `$module` root and subpackages, and `github.com/larsartmann/go-error-family` (same author, zero transitive deps). No other third-party libraries.

### `exhaustruct` — Every Struct Field Must Be Set

When creating any struct literal, you must populate **every field**. Use `//nolint:exhaustruct` only for intentional zero-value returns (e.g. `ETag{}` for "no tag"). In test files this is relaxed.

### `err113` — No Inline `errors.New()`

Package-level sentinel errors only. Do not call `errors.New()` or `fmt.Errorf()` inside functions to create error values that could be package-level sentinels.

### `wsl_v5` — Strict Whitespace Rules

Enforces blank lines before `return`, after declarations, and around control flow. Run `golangci-lint fmt` after editing.

### `nonamedreturns` — No Named Return Values

Do not use named returns in function signatures.

### `noctx` — Always Use Context

`http.NewRequest` is banned. Use `http.NewRequestWithContext`.

### `godot` — Comments End With Periods

All comments must end with a period.

### `mnd` — No Magic Numbers

Extract numeric literals into named constants.

### `gosec` — G705 Excluded Globally

This library's purpose is writing HTTP response bodies, so every `ResponseWriter.Write` is intentional output.

### `paralleltest` — Every Test Must Call `t.Parallel()`

If you write a test function, it must call `t.Parallel()` as its first line.

### `noinlineerr` — No Inline Error Checks

Forbidden: `if err := foo(); err != nil`. Use a separate assignment followed by the check.

### `canonicalheader` — Canonical Header Keys

Header keys must match Go's canonical MIME header form.

### `testableexamples` — Examples Need Output

Every `Example*` function must include a `// Output:` comment directive.

### `thelper` — Test Helpers Must Call `t.Helper()`

Any function taking `*testing.T` that calls `t.Fatal`/`t.Error` must start with `t.Helper()`.

## Commands

```bash
go test ./...              # Run tests
go test -race ./...        # Race detection (REQUIRED for tests with t.Parallel() or shared state)
go vet ./...               # Vet
go test -bench=. ./...     # Benchmarks
golangci-lint run          # Lint
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
```

## Architecture

One module (`github.com/larsartmann/go-etag`), three packages: the real code lives in `server/` (package name `etag`) and `client/` (package `etagclient`); the root is a deprecated alias shim over the server package (deleted at v1.0.0). One external dependency: `github.com/larsartmann/go-error-family`. Go 1.26+.

| File                   | Exports                                                                                                                                                                             | Purpose                                                                                            |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `server/entity_tag.go` | `ETag`, `Strength`, `Strong`, `Weak`, `NewETag`, `ParseETag`, `ParseETagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`                                                               | RFC 7232 §2.3 entity-tag domain type: opaque value + strength, strong/weak comparison, ABNF parser |
| `server/etag.go`       | `ETagConfig`, `DefaultETagConfig()`, `Validate()`, `New()`                                                                                                                          | ETag middleware: FNV-64a generation, If-None-Match 304, buffer overflow, Hijack/Flush streaming    |
| `server/wrapper.go`    | (unexported `responseWrapper`)                                                                                                                                                      | Shared ResponseWriter wrapper: buffers WriteHeader, delegates Hijack/Flush                         |
| `server/middleware.go` | `Middleware`                                                                                                                                                                        | Type alias for `func(http.Handler) http.Handler`                                                   |
| `server/errors.go`     | `ErrCodeETagWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeInvalidConfig`, `ErrCodeHashWriteFailed`, `ErrInvalidConfig`, `RegisterErrorClassifications()` | Error codes + stdlib sentinel registration + message templates                                     |
| `server/hex.go`        | (unexported `hexEncodeUint64`, `hexDigitsLower`)                                                                                                                                    | Zero-alloc stack-allocated hex encoding for uint64 FNV hashes                                      |
| `server/doc.go`        | (package doc only)                                                                                                                                                                  | Package-level GoDoc documentation                                                                  |
| `client/options.go`    | `Options` (KeyFunc / MaxEntries / MaxBodyBytes / PreserveOn304 / FromCacheHeader)                                                                                                   | Client transport options with defaults (256 entries / 1 MiB / `["Date"]`)                          |
| `client/cache.go`      | `Stats` (Hits/Stored/Entries); unexported `responseCache` (map + order-slice FIFO under a mutex)                                                                                    | Bounded concurrency-safe cache with counters                                                       |
| `client/transport.go`  | `NewTransport`, `Transport` (`RoundTrip`, `Stats`)                                                                                                                                  | Conditional GET transport: If-None-Match replay, 304→200 rebuild, body store with size skip        |
| `client/doc.go`        | (package doc only)                                                                                                                                                                  | Package docs incl. the KeyFunc credential warning                                                  |
| `deprecated.go`        | Full alias surface of `server/` (4 types, 7 consts, 1 var, 8 func wrappers), all `// Deprecated:`                                                                                   | Compatibility shim keeping v0.1.x imports compiling; removed at v1.0.0                             |
| `doc.go`               | (package doc only)                                                                                                                                                                  | Tombstone package doc with migration snippet                                                       |

**Middleware pattern:** `New()` returns `func(http.Handler) http.Handler` (aliased as `Middleware`).
**Client pattern:** `NewTransport(next, opts) *Transport`; nil `next` falls back to `http.DefaultTransport`; the cache is embedded (no nil-cache state).

## API Design

### ETag Type (RFC 7232 §2.3)

The `ETag` struct holds an opaque string and a `Strength` (Strong/Weak). It provides:

- `String()` — wire format (`"opaque"` or `W/"opaque"`)
- `StrongEqual(o ETag) bool` — §2.3.2 strong comparison (both must be strong)
- `WeakEqual(o ETag) bool` — §2.3.2 weak comparison (ignores strength)
- `ParseETag(s string) (ETag, bool)` — parse one entity-tag from wire format
- `ParseETagList(header string) []ETag` — parse comma-separated list
- `MatchesIfNoneMatch(tag, header) bool` — weak comparison helper (used internally)
- `MatchesIfMatch(tag, header) bool` — strong comparison helper (for application code)

### Config

`ETagConfig` has fields: `Strength`, `MaxBufferSize`, `HashFunc` (returns string), `SkipIfPresent`, `Skip`, `OnError`, `OnETagGenerated`, `On304`, `OnBufferOverflow`.

- Zero-value `ETagConfig{}` is safe: MaxBufferSize clamped to default (1 MB).
- `HashFunc func([]byte) string` returns opaque-tag content (not a uint64).
- Observability hooks (all nil by default, zero cost when nil): `OnETagGenerated func(ETag)`, `On304 func(ETag)`, `OnBufferOverflow func(int)`. They make the library OTEL/Prometheus-ready without a telemetry dependency.

### Client (package `etagclient`)

`Options` has fields: `KeyFunc` (nil = request URL string; MUST scope by credential when responses vary by caller), `MaxEntries` (default 256, FIFO eviction), `MaxBodyBytes` (default 1 MiB; oversized bodies pass through uncached with bodies intact), `PreserveOn304` (default `["Date"]` per RFC 7232 §4.1; empty non-nil slice disables merging), `FromCacheHeader` (default empty = disabled). `Transport.Stats()` returns `Stats{Hits, Stored, Entries}`.

**Client gotchas:**

- **RoundTrip never mutates the caller's request** — If-None-Match rides on a clone (net/http RoundTripper contract).
- **Oversized bodies keep streaming** — the buffered prefix is chained to the unread remainder; Close still reaches the original body.
- **Go 1.26 canonical header form of `ETag` is `Etag`** — map literals with `"ETag"` keys are invisible to `Header.Get/Set` (Get canonicalizes on read). Build stub headers via `Header.Set` in tests; real transports canonicalize on both sides.

## Error Classification

Errors from `etagWriter` are classified using `go-error-family`:

| Source       | Error Code                    | Family         | Retryable | When                                              |
| ------------ | ----------------------------- | -------------- | --------- | ------------------------------------------------- |
| `Write`      | `http.etag_write_failed`      | Transient      | Yes       | Streaming, overflow, or flush write failure       |
| `Hijack`     | `http.hijack_unsupported`     | Infrastructure | No        | Underlying writer doesn't implement Hijacker      |
| `Hijack`     | `http.hijack_failed`          | Transient      | Yes       | Underlying Hijack call fails                      |
| `Validate`   | `http.etag_config_invalid`    | Rejection      | No        | ETagConfig field value is invalid (e.g. <= 0)     |
| `hash.Write` | `http.etag_hash_write_failed` | Orchestration  | No        | Hash.Write returned an error (contract violation) |

All errors are `*errorfamily.Error` — classified, contextual, retryable-aware.
`ErrInvalidConfig` is the package-level sentinel for `Validate`; `errors.Is` matches by code+family.
Hijack errors include `writer_type` context via `WithContextf`.
Flush-path write errors are forwarded to `ETagConfig.OnError` (a `func(*errorfamily.Error)`) for observability, since they cannot be surfaced to the client or returned from `Write`.

## Non-Obvious Behaviors

- **The middleware constructor is `New(cfg ETagConfig)`** — not `ETag()`. `ETag` is the domain type.
- **`HashFunc` returns a string** — the opaque-tag content, not a uint64. The middleware wraps it with quotes and optional `W/`.
- **Zero-value config is safe** — MaxBufferSize <= 0 is clamped to 1 MB in `newETagWriter`.
- **HEAD responses have no body** — RFC 7230 §3.3. Content-Length is set from the buffered body size.
- **304 strips Content-Length** — RFC 7232 §4.1.
- **`SkipIfPresent` defaults to false** — set to true to respect handler-set ETags.
- **`Skip` predicate** — exclude specific routes from ETag processing.
- **`MatchesIfMatch` is an exported helper** — applications call it in their handlers for unsafe-method lost-update prevention.
- **`ETag` `If-None-Match` uses RFC 7232 §2.3.2 weak comparison** — `W/"abc"` and `"abc"` are treated as equivalent.
- **Buffer overflow disables ETag** — responses exceeding `MaxBufferSize` are streamed without ETag.
- **`OnETagGenerated` only fires for computed tags** — handler-provided tags adopted via `SkipIfPresent` do NOT fire it (name honesty: "generated" ≠ "adopted").
- **`On304` fires regardless of tag source** — computed or handler-provided — and fires in addition to `OnETagGenerated` (ordering: generated first, then 304).
- **`OnBufferOverflow` fires exactly once per response** — the flushed-flag transition guarantees single fire; handler-initiated `Flush()` does NOT fire it. Payload is the exceeded `MaxBufferSize`.
- **Hooks are synchronous and unprotected** — they run in the request goroutine with no recover; net/http isolates panics like handler panics.
- **Hijack/Flush switches to streaming mode** — after either call, the middleware writes through without buffering.
- **Hash.Write errors panic with a classified Orchestration error** — the `hash.Hash` contract guarantees Write never fails; if it does, the hash implementation is broken and we panic with `http.etag_hash_write_failed`.

## Testing Conventions

- **Same package** (`package etag` in `server/`, `package etagclient` in `client/`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **BDD-style specs** in `server/etag_bdd_test.go` — Describe/Context/It pattern using `t.Run`
- **`t.Errorf`** for non-fatal, **`t.Fatalf`** for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for server doubles; `roundTripperFunc` stubs for client doubles
- **Shared test helpers** in `server/testutil_test.go`; client helpers live in `client/transport_test.go`

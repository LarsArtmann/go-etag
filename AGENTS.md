# go-etag — AGENTS.md

## Hard Constraints (Will Break Your Code)

### Allowed Dependencies

`depguard` allows `$gostd`, `$module` root and subpackages, and `github.com/larsartmann/go-error-family` (same author, zero transitive deps). No other third-party libraries.

### `exhaustruct` — Every Struct Field Must Be Set

When creating any struct literal, you must populate **every field**. In test files this is relaxed.

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

Single flat package: `etag`. One external dependency: `github.com/larsartmann/go-error-family`. Go 1.26+.

| File            | Exports                                                                                                       | Purpose                                                                             |
| --------------- | ------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `etag.go`       | `ETagConfig`, `DefaultETagConfig()`, `ETag()`, `Validate()`                                                   | ETag generation (FNV-64a) + RFC 7232 weak-comparison `If-None-Match` 304 middleware |
| `wrapper.go`    | (unexported `responseWrapper`)                                                                                | Shared ResponseWriter wrapper: buffers WriteHeader, delegates Hijack/Flush          |
| `middleware.go` | `Middleware`                                                                                                  | Type alias for `func(http.Handler) http.Handler`                                    |
| `errors.go`     | `ErrCodeETagWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `RegisterErrorClassifications()` | Error codes + stdlib sentinel registration + message templates                      |
| `hex.go`        | (unexported `hexDigitsLower`)                                                                                 | Shared lowercase hex lookup table for ETag encoding                                 |
| `doc.go`        | (package doc only)                                                                                            | Package-level GoDoc documentation                                                   |

**Middleware pattern:** `ETag()` returns `func(http.Handler) http.Handler` (aliased as `Middleware`).

## Error Classification

Errors from `etagWriter` are classified using `go-error-family`:

| Source   | Error Code                | Family         | Retryable | When                                         |
| -------- | ------------------------- | -------------- | --------- | -------------------------------------------- |
| `Write`  | `http.etag_write_failed`  | Transient      | Yes       | Streaming or overflow write failure          |
| `Hijack` | `http.hijack_unsupported` | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `Hijack` | `http.hijack_failed`      | Transient      | Yes       | Underlying Hijack call fails                 |

## Non-Obvious Behaviors

- **`ETag` uses FNV-64a by default** — the `HashFunc func([]byte) uint64` field on `ETagConfig` allows replacing the hash algorithm.
- **`ETag` `If-None-Match` uses RFC 7232 §2.3.2 weak comparison** — `W/"abc"` and `"abc"` are treated as equivalent.
- **`ETag` always overwrites handler-set ETags** — there is no `SkipIfPresent` config.
- **Buffer overflow disables ETag** — responses exceeding `MaxBufferSize` are streamed without ETag.
- **Hijack/Flush switches to streaming mode** — after either call, the middleware writes through without buffering.

## Testing Conventions

- **Same package** (`package etag`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **`t.Errorf`** for non-fatal, `t.Fatalf` for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for HTTP doubles
- **Shared test helpers** in `testutil_test.go`

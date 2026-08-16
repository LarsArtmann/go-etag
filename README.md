# go-etag

> RFC 7232 HTTP ETag middleware for Go — automatic entity-tag generation, conditional request handling, and `304 Not Modified` responses.

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-98.9%25-brightgreen)](#benchmarks)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-etag.svg)](https://pkg.go.dev/github.com/larsartmann/go-etag)

---

## Why?

Every HTTP API that serves reusable representations should send `ETag` headers and honor `If-None-Match`. Doing this by hand in every handler is repetitive, error-prone, and easy to get wrong (HEAD bodies, Content-Length on 304, weak vs strong comparison, malformed header parsing).

**go-etag** is a single-middleware solution that:

- **Buffers the response body**, computes an ETag, and sets the header automatically.
- **Handles `If-None-Match`** with RFC 7232 weak comparison, returning `304 Not Modified` when the client's cache is fresh.
- **Respects handler-set ETags** when `SkipIfPresent` is enabled, so database revision numbers and other strong validators pass through.
- **Streams gracefully** when the body exceeds `MaxBufferSize` or the handler calls `Hijack`/`Flush`.
- **Classifies errors** via [`go-error-family`](https://github.com/larsartmann/go-error-family) for retry-aware observability.

Zero configuration required — wrap your handler and go.

## Quick Start

```go
package main

import (
    "net/http"

    "github.com/larsartmann/go-etag"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte("hello world"))
    })

    // Wrap the entire mux — every GET/HEAD response gets an ETag.
    server := &http.Server{
        Addr:     ":8080",
        Handler:  etag.New(etag.DefaultETagConfig())(mux),
    }

    _ = server.ListenAndServe()
}
```

Verify with curl:

```bash
# First request — server computes and returns the ETag
curl -v http://localhost:8080/
# < ETag: "779a65e7023cd2e7"

# Second request — client sends If-None-Match, server returns 304
curl -v -H 'If-None-Match: "779a65e7023cd2e7"' http://localhost:8080/
# < HTTP/1.1 304 Not Modified
# (no body transferred)
```

## Features

- **Dedicated `ETag` type** with RFC 7232 §2.3.2 strong and weak comparison methods
- **FNV-64a hashing** by default (fast, 64-bit, collision-resistant for practical body counts)
- **RFC 7232 weak comparison** for `If-None-Match` (treats `W/"abc"` and `"abc"` as equivalent)
- **RFC 7230 compliant HEAD** — sets `Content-Length` without sending a body
- **RFC 7232 §4.1 compliant 304** — strips `Content-Length`, includes `ETag`
- **`SkipIfPresent`** — respect handler-set ETags instead of overwriting them
- **`Skip` predicate** — exclude specific routes (SSE, large downloads, streaming)
- **`MatchesIfMatch`** — exported helper for RFC 7232 §3.1 strong comparison (lost-update prevention)
- **Buffer overflow handling** — responses exceeding `MaxBufferSize` stream without an ETag
- **Hijack/Flush aware** — switches to streaming mode when the handler hijacks or flushes
- **Classified errors** via `go-error-family` for retry decisions
- **Zero-value safe** — `ETagConfig{}` clamps to defaults (no unbounded buffering)
- **98.9% test coverage** with table-driven unit tests, fuzz tests, and BDD-style RFC specs

## The ETag Type

The `etag.ETag` type represents an RFC 7232 entity-tag with explicit strength:

```go
tag := etag.NewETag("abc123", etag.Strong)

tag.String()        // "abc123"         — wire format for HTTP headers
tag.OpaqueTag()     // abc123           — unquoted opaque content
tag.IsWeak()        // false
tag.IsValid()       // true             — zero-value ETag{} is not valid

weak := etag.NewETag("abc123", etag.Weak)
weak.String()       // W/"abc123"

// RFC 7232 §2.3.2 comparison functions:
tag.StrongEqual(weak)  // false — strong requires both to be strong
tag.WeakEqual(weak)    // true  — weak ignores strength

// Parsing from header values:
parsed, ok := etag.ParseETag(`W/"abc123"`)
tags := etag.ParseETagList(`"a", W/"b", "c"`)
```

## Configuration

```go
cfg := etag.ETagConfig{
    Strength:         etag.Strong,             // etag.Strong (default) or etag.Weak
    MaxBufferSize:    1024 * 1024,             // 1 MB buffer before streaming (default)
    HashFunc:         nil,                     // nil = FNV-64a; or func([]byte) string
    SkipIfPresent:    false,                   // true = respect handler-set ETags
    Skip:             nil,                     // func(*http.Request) bool — skip routes
    OnError:          nil,                     // func(*errorfamily.Error) — write-failure hook
    OnETagGenerated:  nil,                     // func(etag.ETag) — fires per computed tag
    On304:            nil,                     // func(etag.ETag) — fires per 304 Not Modified
    OnBufferOverflow: nil,                     // func(int) — fires when MaxBufferSize is exceeded
}

// Validate checks for invalid field values. Call at startup.
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

handler := etag.New(cfg)(myHandler)
```

| Field              | Type                       | Default          | Description                                                     |
| ------------------ | -------------------------- | ---------------- | --------------------------------------------------------------- |
| `Strength`         | `Strength`                 | `Strong`         | Strong or weak validator per RFC 7232 §2.1                      |
| `MaxBufferSize`    | `int`                      | `1048576` (1 MB) | Max bytes buffered before abandoning ETag and streaming         |
| `HashFunc`         | `func([]byte) string`      | FNV-64a          | Computes the opaque-tag value from the body. Nil = FNV-64a      |
| `SkipIfPresent`    | `bool`                     | `false`          | When true, handler-set ETags are respected and not overwritten  |
| `Skip`             | `func(*http.Request) bool` | `nil`            | When non-nil, requests returning true bypass ETag processing    |
| `OnError`          | `func(*errorfamily.Error)` | `nil`            | Called when a post-commit write fails (client disconnect, etc.) |
| `OnETagGenerated`  | `func(ETag)`               | `nil`            | Fires each time the middleware computes and sets a new tag      |
| `On304`            | `func(ETag)`               | `nil`            | Fires after a 304 is committed; the cache-hit signal            |
| `OnBufferOverflow` | `func(int)`                | `nil`            | Fires once when a response exceeds MaxBufferSize (arg = limit)  |

## Observability Hooks

The library takes no telemetry dependency. Instead, three lifecycle hooks
give any backend (OpenTelemetry, Prometheus, slog) full visibility in a few
lines:

- `OnETagGenerated` — the denominator of your cache hit ratio. Fires only for
  tags **computed** by the middleware, not handler-provided ones under
  `SkipIfPresent`.
- `On304` — the cache hit. Fires after the 304 is committed, in addition to
  `OnETagGenerated` (which fires first).
- `OnBufferOverflow` — a response silently degraded to streaming without an
  ETag. Fires at most once per response; handler-initiated `Flush()` does not
  fire it.

All hooks fire synchronously in the request goroutine; keep them fast and
panic-free (net/http isolates panics the same way as handler panics). The
pattern is the same for every backend — point the hooks at your handles:

```go
var cacheHits, etagWrites atomic.Int64 // or otel metric / prom counters

cfg := etag.DefaultETagConfig()
cfg.OnETagGenerated = func(etag.ETag) { etagWrites.Add(1) }
cfg.On304 = func(etag.ETag) { cacheHits.Add(1) }
```

See `ExampleNew_observabilityHooks` in the [GoDoc](https://pkg.go.dev/github.com/larsartmann/go-etag)
for a runnable version deriving a hit ratio from plain counters.

## Strong vs Weak Validators

RFC 7232 §2.1 defines two validator strengths:

| Strength | Wire format  | Guarantees                                       | Use for If-Match? |
| -------- | ------------ | ------------------------------------------------ | ----------------- |
| Strong   | `"abc123"`   | Changes whenever the representation data changes | Yes               |
| Weak     | `W/"abc123"` | May not change for every representation change   | No                |

The default FNV-64a hash produces **strong** validators. Use weak validators when your hash function may produce the same value for semantically equivalent but byte-different representations (e.g., JSON key reordering, whitespace normalization).

## Conditional Request Helpers

### If-None-Match (automatic)

The middleware handles `If-None-Match` automatically for GET and HEAD using **weak comparison** (RFC 7232 §3.2). You don't need to write any code.

### If-Match (manual, for unsafe methods)

For `PUT`, `POST`, and `DELETE`, evaluate `If-Match` in your handler using **strong comparison** (RFC 7232 §3.1) to prevent lost updates:

```go
func updateHandler(w http.ResponseWriter, r *http.Request) {
    currentTag := etag.NewETag(currentRevision, etag.Strong)

    if !etag.MatchesIfMatch(currentTag, r.Header.Get("If-Match")) {
        w.WriteHeader(http.StatusPreconditionFailed)
        return
    }

    // ... perform the update ...
}
```

## How It Works

```
Request (GET/HEAD)
    │
    ▼
┌──────────────────┐     Non-GET/HEAD or Skip → pass through (no ETag)
│  Buffer response │
│  body in memory  │
└────────┬─────────┘
         │
         ▼  handler returns
┌──────────────────┐
│  Compute ETag    │     FNV-64a of buffered body (or custom HashFunc)
│  from body       │
└────────┬─────────┘
         │
    ┌────┴────┐
    │ ETag    │
    │ valid?  │
    ├─────────┤
    │ No      │──→ stream body without ETag (overflow / hijack / flush)
    │ Yes     │
    └────┬────┘
         │
         ▼
┌──────────────────┐
│ If-None-Match    │     RFC 7232 weak comparison
│ matches?         │
├──────────────────┤
│ Yes (2xx status) │──→ 304 Not Modified (strip Content-Length, keep ETag)
│ No               │──→ 200 + body + ETag
└──────────────────┘
```

Only `GET` and `HEAD` requests with cacheable status codes (200-299) are eligible for `304`. The middleware never buffers `POST`, `PUT`, or other methods.

## RFC Compliance

| RFC      | Section | Requirement                                    | Status    |
| -------- | ------- | ---------------------------------------------- | --------- |
| RFC 7232 | §2.1    | Strong and weak validator types                | Compliant |
| RFC 7232 | §2.3    | Entity-tag ABNF parsing                        | Compliant |
| RFC 7232 | §2.3.2  | Strong and weak comparison functions           | Compliant |
| RFC 7232 | §3.1    | `If-Match` strong comparison helper            | Compliant |
| RFC 7232 | §3.2    | `If-None-Match` weak comparison (304 handling) | Compliant |
| RFC 7232 | §4.1    | 304 strips Content-Length, includes ETag       | Compliant |
| RFC 7230 | §3.3    | HEAD sets Content-Length without a body        | Compliant |

## Error Classification

Errors from the ETag writer are classified using [`go-error-family`](https://github.com/larsartmann/go-error-family):

| Code                          | Family         | Retryable | When                                             |
| ----------------------------- | -------------- | --------- | ------------------------------------------------ |
| `http.etag_write_failed`      | Transient      | Yes       | Write failure during streaming or overflow       |
| `http.hijack_unsupported`     | Infrastructure | No        | ResponseWriter doesn't implement `http.Hijacker` |
| `http.hijack_failed`          | Transient      | Yes       | Underlying `Hijack()` call fails                 |
| `http.etag_config_invalid`    | Rejection      | No        | `ETagConfig` validation failure                  |
| `http.etag_hash_write_failed` | Orchestration  | No        | Hash function contract violation (bug)           |

Call `etag.RegisterErrorClassifications()` once at startup to enable classification:

```go
func main() {
    etag.RegisterErrorClassifications()
    // ... start server ...
}
```

## Benchmarks

Measured on AMD Ryzen AI MAX+ 395:

| Benchmark                         | ns/op | B/op | allocs/op |
| --------------------------------- | ----- | ---- | --------- |
| Full middleware (GET + body)      | 566   | 1192 | 13        |
| Full middleware (GET + 304 match) | 668   | 1232 | 15        |
| `MatchesIfNoneMatch` (single tag) | 69    | 40   | 2         |
| `MatchesIfNoneMatch` (3-tag list) | 182   | 232  | 4         |

The middleware adds sub-microsecond overhead per request. For a typical API returning small JSON payloads, the ETag computation and If-None-Match matching are negligible compared to network I/O.

## Installation

```bash
go get github.com/larsartmann/go-etag
```

## License

MIT

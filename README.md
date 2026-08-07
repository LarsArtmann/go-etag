# go-etag

HTTP ETag middleware for Go — generates entity-tag headers from response body content and handles `If-None-Match` conditional requests with `304 Not Modified`.

## Features

- **Dedicated `ETag` type** with RFC 7232 §2.3.2 strong and weak comparison methods
- **FNV-64a hashing** by default (fast, 64-bit, collision-resistant)
- **RFC 7232 weak comparison** for `If-None-Match` matching (ignores `W/` prefix)
- **RFC 7230 compliant HEAD** — sends Content-Length without a body
- **RFC 7232 §4.1 compliant 304** — strips Content-Length, includes ETag
- **Configurable** hash function (returns arbitrary opaque-tag strings), buffer size, and strength
- **`SkipIfPresent`** — respects handler-set ETags instead of overwriting them
- **`Skip` predicate** — exclude specific routes from ETag processing
- **`MatchesIfMatch`** — exported helper for RFC 7232 §3.1 strong comparison (lost-update prevention)
- **Buffer overflow handling** — responses exceeding `MaxBufferSize` are streamed without ETag
- **Hijack/Flush aware** — switches to streaming mode when the handler hijacks or flushes
- **Classified errors** via [`go-error-family`](https://github.com/larsartmann/go-error-family) for retry decisions
- **Zero-value safe** — `ETagConfig{}` is clamped to defaults (no unbounded buffering)

## Installation

```bash
go get github.com/larsartmann/go-etag
```

## Quick Start

```go
package main

import (
    "net/http"

    "github.com/larsartmann/go-etag"
)

func main() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte("hello world"))
    })

    wrapped := etag.New(etag.DefaultETagConfig())(handler)

    http.Handle("/", wrapped)
    http.ListenAndServe(":8080", nil)
}
```

## The ETag Type

The `etag.ETag` type represents an RFC 7232 entity-tag with explicit strength and both comparison functions:

```go
tag := etag.NewETag("abc123", etag.Strong)

tag.String()        // "abc123" (wire format for headers)
tag.OpaqueTag()     // abc123 (unquoted content)
tag.IsWeak()        // false

weak := etag.NewETag("abc123", etag.Weak)
weak.String()       // W/"abc123"

// RFC 7232 §2.3.2 comparison functions:
tag.StrongEqual(weak)  // false (strong requires both strong)
tag.WeakEqual(weak)    // true  (weak ignores strength)

// Parsing from header values:
parsed, ok := etag.ParseETag(`W/"abc123"`)
```

## Configuration

```go
cfg := etag.ETagConfig{
    Strength:      etag.Weak,             // Strong (default) or Weak
    MaxBufferSize: 2 * 1024 * 1024,      // 2 MB buffer before streaming
    HashFunc:      myCustomHash,          // func([]byte) string (nil = FNV-64a)
    SkipIfPresent: true,                  // Respect handler-set ETags
    Skip:          func(r *http.Request) bool {
        return r.URL.Path == "/stream"    // Skip ETag for specific routes
    },
    OnError:       func(e *errorfamily.Error) {
        log.Printf("etag write error: %v", e)
    },
}

if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

handler := etag.New(cfg)(myHandler)
```

## Conditional Request Helpers

For unsafe methods (PUT, POST, DELETE) where `If-Match` must be evaluated by application code:

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

1. For `GET` and `HEAD` requests, the middleware buffers the response body.
2. After the handler returns, it computes the ETag from the buffered body.
3. If `If-None-Match` matches (using RFC 7232 weak comparison), it returns `304 Not Modified`.
4. Otherwise, it writes the buffered body with the `ETag` header.
5. For `HEAD` requests, it sets `Content-Length` but does not send a body (RFC 7230 §3.3).
6. If the body exceeds `MaxBufferSize`, the middleware switches to streaming mode (no ETag).
7. If the handler calls `Hijack` or `Flush`, the middleware switches to streaming mode immediately.
8. If `SkipIfPresent` is true and the handler already set an ETag, it is respected.

## Error Classification

Errors from the ETag writer are classified using `go-error-family`:

| Code                      | Family         | Retryable | When                                             |
| ------------------------- | -------------- | --------- | ------------------------------------------------ |
| `http.etag_write_failed`  | Transient      | Yes       | Write failure during streaming or overflow       |
| `http.hijack_unsupported` | Infrastructure | No        | ResponseWriter doesn't implement `http.Hijacker` |
| `http.hijack_failed`      | Transient      | Yes       | Underlying `Hijack()` call fails                 |
| `http.etag_config_invalid`| Rejection      | No        | `ETagConfig` validation failure                  |
| `http.etag_hash_write_failed` | Orchestration | No     | Hash function contract violation (bug)           |

Call `etag.RegisterErrorClassifications()` once at startup to enable classification.

## License

MIT

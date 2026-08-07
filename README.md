# go-etag

HTTP ETag middleware for Go — generates ETag headers from response body content and handles `If-None-Match` conditional requests with `304 Not Modified`.

## Features

- **FNV-64a hashing** by default (fast, 64-bit, collision-resistant)
- **RFC 7232 weak comparison** for `If-None-Match` matching (ignores `W/` prefix)
- **Configurable** hash function, buffer size, and weak/strong ETag mode
- **Buffer overflow handling** — responses exceeding `MaxBufferSize` are streamed without ETag
- **Hijack/Flush aware** — switches to streaming mode when the handler hijacks or flushes
- **Classified errors** via [`go-error-family`](https://github.com/larsartmann/go-error-family) for retry decisions
- **Zero allocations** on the hot path for common responses

## Installation

```bash
go get github.com/larsartmann/go-etag
```

## Quick Start

```go
package main

import (
    "net/http"

    etag "github.com/larsartmann/go-etag"
)

func main() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("hello world"))
    })

    // Wrap with ETag middleware.
    wrapped := etag.ETag(etag.DefaultETagConfig())(handler)

    http.Handle("/", wrapped)
    http.ListenAndServe(":8080", nil)
}
```

## Configuration

```go
cfg := etag.ETagConfig{
    Weak:          true,                // Generate weak ETags (W/"...")
    MaxBufferSize: 2 * 1024 * 1024,    // 2 MB buffer before streaming
    HashFunc:      myCustomHash,        // Custom hash function (nil = FNV-64a)
}

if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

handler := etag.ETag(cfg)(myHandler)
```

## How It Works

1. For `GET` and `HEAD` requests, the middleware buffers the response body.
2. After the handler returns, it computes the ETag from the buffered body.
3. If `If-None-Match` matches (using RFC 7232 weak comparison), it returns `304 Not Modified`.
4. Otherwise, it writes the buffered body with the `ETag` header.
5. If the body exceeds `MaxBufferSize`, the middleware switches to streaming mode (no ETag).
6. If the handler calls `Hijack` or `Flush`, the middleware switches to streaming mode immediately.

## Error Classification

Errors from the ETag writer are classified using `go-error-family`:

| Code | Family | Retryable | When |
| --- | --- | --- | --- |
| `http.etag_write_failed` | Transient | Yes | Write failure during streaming or overflow |
| `http.hijack_unsupported` | Infrastructure | No | ResponseWriter doesn't implement `http.Hijacker` |
| `http.hijack_failed` | Transient | Yes | Underlying `Hijack()` call fails |

Call `etag.RegisterErrorClassifications()` once at startup to enable classification.

## License

MIT

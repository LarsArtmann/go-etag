# server — HTTP ETag middleware (package `etag`)

Server-side half of [go-etag](../README.md): buffers `GET`/`HEAD` response
bodies, computes an `ETag` header (FNV-64a by default), and answers matching
`If-None-Match` requests with `304 Not Modified` — RFC 7232 compliant,
zero configuration required.

```go
import (
    "net/http"

    etag "github.com/larsartmann/go-etag/server"
)

handler := etag.New(etag.DefaultETagConfig())(mux)
```

Highlights:

- **Weak comparison** for `If-None-Match` (`W/"abc"` ≡ `"abc"`), 304 only on
  cacheable 2xx statuses; `Content-Length` stripped per RFC 7232 §4.1.
- **HEAD compliant** (RFC 7230 §3.3): `Content-Length` set, no body sent.
- **Graceful degradation**: bodies larger than `MaxBufferSize` stream through
  without an ETag; `Hijack`/`Flush` switch to streaming mode.
- **Handler-set ETags respected** via `SkipIfPresent`; per-route opt-out via
  `Skip`.
- **Observability hooks** (`OnETagGenerated`, `On304`, `OnBufferOverflow`,
  `OnError`) — no telemetry dependency; see the
  [`metrics`](../metrics/README.md) companion for ready-made counters.
- **Classified errors** via
  [go-error-family](https://github.com/larsartmann/go-error-family), plus a
  typed `Code`/`Domain` routing surface for retry-aware consumers.
- The `ETag` domain type itself lives in
  [`entitytag`](../entitytag/README.md); this package re-exports the full
  surface so `etag.ETag` keeps compiling unchanged.

Full API, config reference, and examples:
[pkg.go.dev/github.com/larsartmann/go-etag/server](https://pkg.go.dev/github.com/larsartmann/go-etag/server).
For conditional `If-Match` handling (lost-update prevention) the exported
`etag.MatchesIfMatch` helper keeps unsafe-method preconditions in application
hands.

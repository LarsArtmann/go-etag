# client — conditional GET cache transport (package `etagclient`)

Client-side half of [go-etag](../README.md): an `http.RoundTripper` that
stores 200 responses by their ETag, replays the validator as
`If-None-Match` on the next GET, and rebuilds the cheap `304 Not Modified`
into the `200` the caller expects — RFC 9111 conformant.

```go
import "github.com/larsartmann/go-etag/client"

client := &http.Client{
    Transport: etagclient.NewTransport(http.DefaultTransport, etagclient.Options{}),
}
```

The transport always revalidates before serving a stored body (it is a
conditional-GET accelerator, not a freshness cache), so it can never serve
stale origin content — while unchanged responses cost a bodiless 304 instead
of a full re-download.

Highlights:

- **§4.3.4 freshening** (the zero-value `FreshenPolicy`): every header field
  the 304 provides replaces the stored value, except hop-by-hop fields and
  `Content-Length`/`Content-Range`; `FreshenFields(...)` restricts,
  `FreshenNone()` disables.
- **§4.3.5 HEAD freshening**: a confirming `HEAD` 200 updates stored metadata
  (or invalidates it) without spending a GET.
- **§4.4 invalidation**: a non-error response to an unsafe method drops the
  stored entry for that URI.
- **`Cache-Control: no-store`** responses are never stored (§3); a `no-store`
  **request** bypasses the cache entirely (§5.2.2.5); caller-supplied
  `If-None-Match` is never clobbered.
- **`Age` surfaced verbatim** on 200s and freshened from 304s — the §5.1
  signal that an edge cache served a stale copy.
- **Bounded memory**: FIFO eviction (`MaxEntries`, default 256) and a body
  cap (`MaxBodyBytes`, default 1 MiB); oversized bodies stream through intact.
- **Telemetry**: `Stats()` reports `{Hits, Stored, Entries}`.

> **Credential warning:** the default cache key is the request URL. If the
> same URL can return different responses for different callers, supply a
> `KeyFunc` scoped by credential — or one principal's cached response may be
> served to another.

Full API and examples:
[pkg.go.dev/github.com/larsartmann/go-etag/client](https://pkg.go.dev/github.com/larsartmann/go-etag/client).
The requirement-by-requirement RFC 9111 conformance table lives in
[`docs/rfc9111-conformance.md`](../docs/rfc9111-conformance.md).

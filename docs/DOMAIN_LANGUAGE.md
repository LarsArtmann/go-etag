# Domain Language

Ubiquitous vocabulary for go-etag. Terms are RFC-grounded; "Where" cites the
code that embodies each term.

## Glossary

| Term                  | Definition                                                                                                                                                     | Where                                      |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| Entity tag (ETag)     | An opaque identifier for a specific version of a representation (RFC 9110 §8.8.3): `W/"abc123"` or `"abc123"`.                                                 | `server/entity_tag.go` (`ETag`)            |
| Validator            | A cache validator in general — an entity tag or a last-modified date — used by conditional requests. The client stores one per cache entry.                       | `client/cache.go` (`cacheEntry.etag`)      |
| Strength             | Whether an entity tag is `Strong` or `Weak`. Strong tags must change when the representation changes; weak tags may not. Encoded in the type, not a bool.         | `server/entity_tag.go` (`Strength`)        |
| Opaque tag           | The quoted content of an entity tag without quotes or the `W/` prefix. Servers treat it as opaque.                                                               | `ETag.OpaqueTag()`                         |
| Strong comparison    | RFC 7232 §2.3.2: both tags must be strong and byte-equal. Required for `If-Match`.                                                                                | `ETag.StrongEqual`, `MatchesIfMatch`       |
| Weak comparison      | RFC 7232 §2.3.2: opaque-tags equal, strength ignored. Correct for `If-None-Match`.                                                                               | `ETag.WeakEqual`, `MatchesIfNoneMatch`     |
| Conditional GET      | A GET carrying `If-None-Match`; the server answers `304 Not Modified` when the validator still matches, transferring no body.                                      | server middleware + client transport       |
| Revalidation         | Asking the origin (or edge) whether a stored response is still fresh, by sending its validator. The client always revalidates before serving from cache.            | `client/transport.go` (`RoundTrip`)        |
| Rebuild              | Synthesizing the 200 the caller expects from a stored body + a 304's fresh headers. SDKs see no difference from a real 200.                                        | `rebuildFromCache`                         |
| Freshening           | RFC 9111 §4.3.4/§3.2: every header field a 304 provides replaces the stored value (except hop-by-hop, Content-Length/Content-Range, and Content-Encoding of a decoded body). The default (`PreserveOn304: nil`). | `freshenedHeader`, `persistFreshened` |
| Freshen exceptions   | The fields §3.1/§3.2 exclude from freshening: hop-by-hop, `Content-Length`, `Content-Range`, `Content-Encoding` when net/http transparently decoded the body.       | `skippedByFreshening`                      |
| Invalidation         | RFC 9111 §4.4: a non-error (2xx/3xx) response to an unsafe method drops the stored entry for that URI, so mutations cannot resurrect pre-mutation bodies.           | `responseCache.invalidate`                 |
| Unsafe method        | Any method except GET/HEAD/OPTIONS/TRACE (including unrecognized ones) — the methods that can mutate origin state and therefore trigger invalidation.                 | `isUnsafeMethod`                           |
| No-store ban         | RFC 9111 §3: responses carrying `Cache-Control: no-store` MUST NOT be stored; bodies still stream through intact. Quote-aware and case-insensitive.                 | `hasNoStoreDirective`                      |
| Hop-by-hop fields    | Connection-scoped fields (`Connection`-listed, `Keep-Alive`, `Proxy-*`) stripped at store time (§3.1) so rebuilds cannot resurrect connection state.                 | store path, `TestSpecStoredHeaderShedsHopByHopFields` |
| Age                  | RFC 9111 §5.1: seconds since the response was generated. Presence implies it did not come fresh from the origin — the stale-edge signal. Surfaced verbatim on 200s; freshened from 304s. | `client/doc.go`, `spec_test.go` |
| Cache key / KeyFunc  | The string a stored response is filed under (default: request URL). MUST be credential-scoped when responses vary by caller — the sharp edge the docs warn about loudly. | `client/options.go` (`KeyFunc`)            |
| Buffer overflow      | A server response body exceeding `MaxBufferSize`: the middleware abandons the ETag and streams the body through. Fires `OnBufferOverflow` exactly once.              | `server/etag.go` (`Write` overflow path)  |
| Streaming mode       | The middleware's post-Hijack/post-Flush/post-overflow state: writes pass through unbuffered, no ETag is computed.                                                   | `responseWrapper`, `markFlushed`           |
| From-cache marker    | Optional header (`FromCacheHeader`) set to `1` on rebuilt responses for diagnostics — policy, off by default.                                                       | `client/options.go`                        |
| Classified error     | An error as `*errorfamily.Error` with code, family, retryability, and context (`writer_type`, `max_buffer_size`) — retry-aware observability without a telemetry dep. | `server/errors.go`                         |

## Bounded contexts

- **Server** speaks RFC 7232 (precondition headers on the wire). **Client**
  speaks RFC 9111 (stored-response lifecycle). Both share the entity-tag
  grammar; "validator" is client-side vocabulary, "entity tag" server-side.
- **`OnError` (server)** covers only post-commit write failures — errors a
  handler can no longer act on. Streaming `Write` errors still return to the
  handler. Not a catch-all.

## Deprecated terms

- **Root package usage** (`import "github.com/larsartmann/go-etag"`) — the
  deprecated shim; say "server package" for new code. Removed at v1.0.0.

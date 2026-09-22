# metrics — ready-made counters for the ETag hooks (package `metrics`)

The observability companion to the [server middleware](../server/README.md):
[`Attach`](https://pkg.go.dev/github.com/larsartmann/go-etag/metrics) turns
the `ETagConfig` observability hooks (`OnETagGenerated`, `On304`,
`OnBufferOverflow`) into plain atomic counters in one call — preserving any
hooks already installed, so your Prometheus exporter or slog logger and these
counters coexist on one config.

```go
import "github.com/larsartmann/go-etag/metrics"

cfg, counters := metrics.Attach(etag.DefaultETagConfig())
handler := etag.New(cfg)(mux)

// elsewhere:
slog.Info("etag cache", "hit_ratio", counters.HitRatio())
```

Surface:

- `Counters` — plain `atomic.Int64` fields `Generated`, `NotModified`,
  `BufferOverflows`.
- `Snapshot()` — a consistent copy of the values.
- `HitRatio()` — `NotModified / Generated`. The denominator is exact: for a
  computed tag, `On304` fires in addition to `OnETagGenerated`, so `Generated`
  alone already counts every tag-computing response. (Caveat: 304s on
  handler-adopted tags via `SkipIfPresent` fire `On304` without
  `OnETagGenerated` and can push the ratio above 1.)

The package is deliberately metric-agnostic: exposition format (Prometheus,
OpenTelemetry, slog) stays with the consumer, keeping go-etag free of a
telemetry dependency. Moved in from `httputil/etagmetrics` (2026-09-22) so
the companion ships next to the hooks it counts, on go-etag's release
cadence; the `HitRatio` formula here corrects the double-counting variant
shipped in httputil v1.3.0.

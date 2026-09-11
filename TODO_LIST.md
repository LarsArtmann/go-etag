# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

| # | Priority | Task                                                                                                                                                            | Effort | Evidence / Origin                                                                          |
| - | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------ |
| 1 | High     | Push cqrs-htmx's local alignment commits (7 ahead of origin) and watch its CI green: httputil v1.0.1, go-etag v0.3.1, server_timing v1.0.1 across 22 modules.  | S      | httputil v1.0.1 + `server_timing/v1.0.1` tagged, proxy + clean-room `go get` verified; cqrs-htmx train checker 43 → 1 (residual is the go-cqrs-lite `badgerengine` v4.2.1 family, different train) |
| 2 | Low      | Fix cqrs-htmx's pre-existing standalone (`GOWORK=off`) test failures in `systemadapter`, `examples/system-demo`, `integration_test` (fail on the pre-bump commit too). | M   | found during the v1.0.1 alignment sweep 2026-09-11; `system-demo`: go-cqrs-lite `metaengine.Store.Reset` undefined (version mismatch) |

## Done this cycle (2026-09-11)

- httputil release train: `v1.0.1` + `server_timing/v1.0.1` cut carrying go-etag v0.3.1 (CHANGELOG cut, 9-gate prerelease script, CI green on the tagged commit, tags pushed, proxy + sum.golang.org verified, clean-room `go get` + run verified, GitHub Releases v1.0.1 (Latest) + backfilled v1.0.0 page).
- cqrs-htmx alignment (local): httputil v0.12.0 → v1.0.1 (22 modules), go-etag v0.2.0 → v0.3.1 (22), server_timing v0.12.0 → v1.0.1 (20), vendor refreshed; both workspace and per-module standalone modes verified.
- v0.3.0 GitHub Release notes: migration diff (before/after + mapping table) appended under the Breaking section.
- `ExampleFreshenPolicy` added (`client/example_test.go`, Output-pinned).
- `FreshenFields()` zero-arg normalized to `FreshenNone` (behavior identical, representation honest; documented + tested).
- CI `GOTOOLCHAIN` pin scoped to test/lint/fuzz jobs; govulncheck job unpinned so `govulncheck@latest` can fetch newer toolchains.
- `docs/rfc9111-conformance.md`: "Interpretation decisions" section added (section instead of column — rationale table reads better than a mostly-empty column).
- `go list -m -u`: go-error-family v0.10.0 is current; no updates.

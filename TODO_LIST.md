# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

| # | Priority | Task                                                                                                                                                          | Effort | Evidence / Origin                                             |
| - | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------- |
| 1 | High     | Cut an httputil release carrying go-etag v0.3.0, then align cqrs-htmx's 43 train-lag modules (they stay on go-etag v0.2.0 via published httputil tags until then). | M      | report `2026-09-11_02-51` follow-up 1; report `2026-09-11_02-56` §f.11–12 |
| 2 | High     | Append the `PreserveOn304`→`FreshenOn304` migration diff (README "Upgrading from v0.2.x") to the v0.3.0 GitHub Release notes via `gh release edit`.               | S      | README Upgrading section; report `2026-09-11_02-38` §f.5       |
| 3 | Medium   | Add a GoDoc example for `FreshenPolicy` (`client/example_test.go`, Output-pinned per `testableexamples`) — the only new v0.3.0 API surface without one.           | S      | `client/example_test.go` (no FreshenPolicy example); report `2026-09-11_02-08` §f.6 |
| 4 | Medium   | Scope the `GOTOOLCHAIN` pin to test/fuzz jobs, or pin the govulncheck version, so `@latest` installs cannot break CI.                                             | S      | `.github/workflows/ci.yml:16` (workflow-level pin); report `2026-09-11_02-08` §b.1 |
| 5 | Medium   | Document or normalize `FreshenFields()` with zero arguments — it constructs named-fields mode with an empty list, silently behaving like `FreshenNone()`.         | XS     | `client/options.go:79`; report `2026-09-11_02-08` §b.4         |
| 6 | Medium   | Add an "Interpretation decisions" column to `docs/rfc9111-conformance.md` for the judgment calls (no-store HEAD neutrality, unprovable identity = stale, FreshenOn304 ≠ HEAD). | S      | `docs/rfc9111-conformance.md`; report `2026-09-11_02-08` §e.3  |
| 7 | Low      | `go list -m -u` sweep for go-error-family updates as post-release hygiene (last verified v0.10.0).                                                                | S      | `go.mod`; report `2026-09-11_02-38` §f.10                      |

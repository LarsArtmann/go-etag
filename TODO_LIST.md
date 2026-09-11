# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

| # | Priority | Task                                                                                                                                                                               | Effort | Evidence / Origin                                                    |
| - | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------- |
| 1 | Critical | **Migrate the six in-house consumers (`httputil`, `go-github-kit`, `cqrs-htmx`, `DiscordSync`, `nsfw-classifier`, `library-policy`) off the removed `PreserveOn304` onto `FreshenOn304`** — they are compile-broken against v0.3.0. Run the go-ecosystem-upgrade flow per repo. | M      | `CHANGELOG.md` `[0.3.0]` BREAKING; report `2026-09-11_02-08` item 2 |
| 2 | High     | Diagnose the Dependabot "actions group" PR red CI (run 34545953600), then land the actions bumps (Node 20 deprecation fix) once green. | S      | `gh run list` at release; report `2026-09-11_02-38` §f.2–3 |
| 3 | High     | README "Upgrading": add the v0.2→v0.3 `PreserveOn304`→`FreshenOn304` migration diff (before/after constructor mapping). | S      | CHANGELOG `[0.3.0]` BREAKING; report `2026-09-11_02-38` §f.4 |
| 4 | Medium   | Append the same migration diff to the v0.3.0 GitHub Release notes via `gh release edit`. | S      | Report `2026-09-11_02-38` §f.5 |
| 5 | Medium   | Re-run `go test -bench=.` at the v0.3.0 tree; archive the baseline under `reports/bench/` (reproduce the ~19% lazy-key claim). | S      | CHANGELOG `[0.3.0]` Changed; report `2026-09-11_02-38` §f.6 |
| 6 | Medium   | Scope the `GOTOOLCHAIN` pin to test/fuzz jobs, or pin the govulncheck version, so `@latest` installs cannot break CI. | S      | Report `2026-09-11_02-08` item 4 |
| 7 | Medium   | Fix the stale FEATURES.md shim row (export-parity suite exists since `a5de386`). | S      | Report `2026-09-11_02-08` item 5 |

# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

The former items #1–#7 (v0.5.0 release train, go-floor decision, consumer sweep,
LSP fix, HEAD-freshening hardening ×2, stored-validator before-state) were all
executed and closed on 2026-09-23 — see `CHANGELOG.md` [0.5.0] and
`docs/status/2026-09-23_04-52_v050-release-full-backlog-execution.md`. The next
open work lives in `docs/planning/2026-09-23_01-40_pareto-v05-cycle-full-backlog.md`
(M1 owner decision batch OQ2–OQ8, then M10–M27; M8/M9 already landed pre-tag).

| # | Priority | Task                                                                                                                                                                                                                                                                                                                                                                                                                                                 | Effort | Evidence / Origin                                                             |
| - | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ----------------------------------------------------------------------------- |
| 1 | Medium   | Owner decision batch (OQ2–OQ8 from ROADMAP): Alex fixtures/email, release workflow, FNV affirm, open-low promote-or-die, shim scope, constructor trim, art-dupl enforcement — one structured prompt, answers recorded in ROADMAP                                                                                                                                                                                                                     | M      | `ROADMAP.md` Open Questions; pareto plan M1 (gates the M24–M27 roadmap epics) |
| 2 | Low      | Consumer-repo leftovers surfaced by the v0.5.0 sweep (each repo's own backlog, listed for handoff): library-policy's 307-line `structured_formatters.go` (quality gate), cqrs-htmx's flake toolchain pin (1.26.7 < floor, `scripts/check-go-toolchain.sh` red, predates the sweep), DiscordSync's two pre-existing test failures (mime mapping + disk-space health threshold), ~35 fleet repos carrying dead (unimported) go-etag requires at v0.3.1 | M      | status report 2026-09-23_04-52 §c; per-repo commits of the sweep              |
| 3 | Low      | Pareto plan tier "rest" + roadmap epics (M10–M27): benchstat tables, erraudit-in-CI, coverage floor decision, fuzz expansion, spec pin-ups, README sections, Theme 1 design doc, client hooks spike, v1.0.0 criteria                                                                                                                                                                                                                                 | L      | `docs/planning/2026-09-23_01-40_pareto-v05-cycle-full-backlog.md`             |

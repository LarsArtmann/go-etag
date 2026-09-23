# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

Item #1 (the v0.6.0 release — the 2026-09-23 module split shipped via the bottom-up
tag staircase: five annotated tags, CI green per stair, proxy/sum verified, clean-room
`go get` ×5, GitHub Release Latest, all six in-house consumers swept) was executed
2026-09-23 — see `CHANGELOG.md` [0.6.0] and
`docs/status/2026-09-23_16-40_v060-staircase-execution-and-polish.md`. The
verification-gap tier (M7: PACKAGES single-sourcing + gate agreement check + actionlint)
and the art-dupl re-baseline (M9) closed in the same session. The current plan of
record is still `docs/planning/2026-09-23_14-07_v060-staircase-and-full-backlog.md`
(M10–M27 numbers reference it).

| # | Priority | Task                                                                                                                                                                                                                                                                                                | Effort | Evidence / Origin                                                                                              |
| - | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------- |
| 1 | High     | Owner decision batch OQ2–OQ10 + fleet dead-requires: Alex fixtures/email, release-workflow automation, FNV affirm, open-low promote-or-die, shim scope, constructor trim, art-dupl enforcement posture, Validate-vs-New semantics (OQ9), Alex-response lane — one structured prompt, answers recorded in ROADMAP; separately: drop-or-bump the 38 fleet repos carrying dead (unimported) go-etag requires at v0.3.1/v0.5.0 (exact list in the 16:40 status report) | M      | `ROADMAP.md` Open Questions; `docs/status/2026-09-23_15-35_brutal-self-review-session-and-parallel-staircase.md` g#3 |
| 2 | Medium   | Benchstat tables + five-module post-split no-drift proof (M11); fuzz expansion — `FuzzStoredValidatorWeaklyMatches`, `FuzzMergeHeader`, Cache-Control directive corpus (M16); spec pin-ups — request `no-cache` §5.2.2.2, HEAD × `Uncompressed`, `restoreMismatchedValidator` restricted mode (M17) | M      | pareto plan 14:07 M11/M16/M17                                                                                   |
| 3 | Medium   | erraudit-in-CI posture (M12, OWNER), coverage floor decision with reconciled numbers (M13, OWNER), dependency sanity — gosec note + go-error-family v0.10.1 diff read (M14), httputil mirror-test upstream: verified unpinned 2026-09-23 — httputil declares its own `http.hijack_unsupported` string with no test pinning go-etag's templates (M15/F56) | S      | pareto plan 14:07 M12–M15; verified this session                                                               |
| 4 | Low      | README sections: Middleware Chaining, Troubleshooting, CDN ETag-stripping note; rfc9111-conformance client-module note (M19); CI polish: `workflow_dispatch`, LICENSE/README drift check, dprint decision, OWNER branch protection (M20); ROADMAP annotations + otel note (M21) | M      | pareto plan 14:07 M19–M21                                                                                       |
| 5 | Low      | Epics (owner-gated): Theme 1 freshness design doc (M22), client hooks spike (M23), v1.0.0 criteria — deletion checklist seeded in AGENTS Release Conventions (M24), public presence spike (M25), daemon go-directive root-cause upstream (M26), consumer-repo leftovers: library-policy 307-line formatter gate, cqrs-htmx toolchain pin, DiscordSync mime + disk-space test failures (M27) | L      | pareto plan 14:07 M22–M27; AGENTS v1.0.0 seed                                                                   |

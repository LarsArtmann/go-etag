# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

Items executed and closed on 2026-09-23: the v0.6.0 release (five-tag bottom-up
staircase, post-release proof, six-consumer sweep — see `CHANGELOG.md` [0.6.0]
and `docs/status/2026-09-23_16-40_v060-staircase-execution-and-polish.md`) and
the entire owner decision batch (OQ2–OQ10 answered and recorded in
`ROADMAP.md` Open Questions; the release workflow, art-dupl gate enforcement,
and Validate doc reconciliation shipped same-day; the 54-manifest dead-requires
fleet was batch-bumped to v0.6.0 per owner decision). The plan of record is
still `docs/planning/2026-09-23_14-07_v060-staircase-and-full-backlog.md`
(M-numbers reference it).

| # | Priority | Task                                                                                                                                                                                                                                                                                                | Effort | Evidence / Origin                                                                                              |
| - | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------- |
| 1 | Medium   | Benchstat tables + five-module post-split no-drift proof (M11); fuzz expansion — `FuzzStoredValidatorWeaklyMatches`, `FuzzMergeHeader`, Cache-Control directive corpus (M16); spec pin-ups — request `no-cache` §5.2.2.2, HEAD × `Uncompressed`, `restoreMismatchedValidator` restricted mode (M17) | M      | pareto plan 14:07 M11/M16/M17                                                                                   |
| 2 | Medium   | erraudit-in-CI posture (M12), dependency sanity — gosec note + go-error-family v0.10.1 diff read (M14), httputil mirror-test upstream: verified unpinned 2026-09-23 — httputil declares its own `http.hijack_unsupported` string with no test pinning go-etag's templates (M15/F56) | S      | pareto plan 14:07 M12–M15; verified this session                                                               |
| 3 | Low      | README sections: Middleware Chaining, Troubleshooting, CDN ETag-stripping note; rfc9111-conformance client-module note (M19); CI polish: `workflow_dispatch` for ci.yml, LICENSE/README drift check, dprint decision (M20); ROADMAP annotations + otel note (M21) | M      | pareto plan 14:07 M19–M21                                                                                       |
| 4 | Low      | Epics: Theme 1 freshness design doc (M22), client hooks spike (M23), v1.0.0 criteria — deletion checklist seeded in AGENTS Release Conventions (M24), public presence spike (M25), daemon go-directive root-cause upstream (M26 — including the go-get directive-rewrite variant found in nsfw-classifier), consumer-repo leftovers: library-policy 307-line formatter gate, cqrs-htmx toolchain pin, DiscordSync mime + disk-space test failures (M27) | L      | pareto plan 14:07 M22–M27; AGENTS v1.0.0 seed                                                                   |

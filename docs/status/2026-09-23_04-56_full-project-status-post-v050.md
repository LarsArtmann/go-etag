# Status Report — Full Project Status After the 2026-09-23 Backlog Execution

**Date:** 2026-09-23 04:56 CEST · **Trigger:** explicit a–g status request
**Scope:** whole go-etag project + the consumer fleet it feeds. Companion to (not a rewrite of) `docs/status/2026-09-23_04-52_v050-release-full-backlog-execution.md`, which documents this session's execution in detail.
**Repo state:** master `768d1c3` (pushed, CI green), tag `v0.5.0` = `af2b6d1` live on proxy.golang.org + sum.golang.org + pkg.go.dev. Working tree clean.

---

## a. FULLY DONE

1. **v0.5.0 released end-to-end, zero skipped steps** — CHANGELOG cut before tag (`af2b6d1`), CI green on the exact commit (run 35803874554) AND on the frozen tag run (35804030500), proxy `.info` hash == `af2b6d1`, sum DB recorded (`h1:WivCc…`), clean-room `go get …/metrics@v0.5.0` + Attach/HitRatio smoke green, GitHub Release published Latest non-prerelease (house precedent v0.1.0→v0.5.0), pkg.go.dev renders `metrics` with README + runnable example within the hour.
2. **The entire TODO_LIST #1–#7 executed this session** — go-floor restored to `1.27.1` (`45e68c8`), release (#1), consumer sweep (#3), LSP verified healthy (#4), HEAD-freshening marker spec pin (#5, `2ac636b`), cloneHeader proof + validator-fallback pin (#6, `8c1c1a7`), stored-validator before-state bench (#7). Plus plan pre-tag items: M7a HEAD-path bench pair, M8 `scripts/pre-release-check.sh` (dry-run exit 0), M9 GoDoc examples (`32c9793`).
3. **All six real consumers swept to v0.5.0, verified and committed** (local): httputil (+ toolchain/nix/CI pin alignment), go-github-kit (devshell go_1_27; already FreshenOn304-API), nsfw-classifier (vendor/ + vendorHash rotation), library-policy (flake input pinned at the v0.5.0 tag commit + four pre-existing nix-check reds fixed), DiscordSync (flake rev repin + vendorHash), cqrs-htmx (22 modules, per-module GOWORK=off gates + `check-modules` fully green).
4. **Quality bar measurably raised** — client statement coverage 99.1% → **99.7%**; `persistFreshened`/`cloneHeader` at 100%; full suite race-clean; `scripts/pre-release-check.sh` encodes the whole local release gate including the go-directive ↔ CI-pin parity check that would have caught the 969d077 daemon drift.
5. **Bench corpus complete** — five baseline artifacts under `reports/bench/` now include BOTH new pairs from this session (HEAD-freshening before/after: dedup costs +432 B/op +3 allocs/op; stored-validator before-state from `07fe65c^`, inventory-identical to the 09-18 after file).
6. **Docs de-drifted** (yesterday's audit): 4 package READMEs, annotated snapshots, living docs current; today's close-out updated TODO_LIST/FEATURES/AGENTS + this report.

## b. PARTIALLY DONE / NOT STARTED

1. **Consumer pushes** — all six repos hold verified, unpushed sweep commits (permission question, §g/§e). Their CI has not yet validated the sweeps.
2. **ROADMAP OQ2–OQ8 owner decisions** — never drafted into the structured prompt (M1 was only executed for its go-floor piece). Now TODO_LIST #1.
3. **Plan tiers "rest" + roadmap epics (M10–M27)** — not started except M8/M9: benchstat tables, CHANGELOG link-lint, erraudit-in-CI, coverage-floor decision, dependency sanity notes, fuzz expansion (F54–F57), spec pin-ups (F58–F60), README sections (chaining/troubleshooting/CDN), CI polish, repo hygiene (stray `workflow-audit-log-*.html` at root, branch protection, homepage URL), Theme 1 (§4.2 freshness serving) design doc, Theme 3 client-hooks spike, v1.0.0 criteria, public presence. Now TODO_LIST #3.
4. **metrics coverage 96.8%** — the missing 3.2% was never analyzed (no open item tracks it).
5. **pkg.go.dev root/server/client/entitytag pages at v0.5.0** — metrics page verified explicitly; the other four not re-fetched individually (they render from the same tag; near-certain fine, unverified).

## c. TOTALLY FUCKED UP

Nothing in go-etag itself — build/vet/race/lint/erraudit/nolint-audit/mod-verify/parity all green at HEAD and at the tag; CI green. The honest damage list is all pre-existing, in consumer repos, surfaced by the sweep:

1. **cqrs-htmx `scripts/check-go-toolchain.sh` is RED** — flake toolchain pins go 1.26.7 below the workspace's own 1.27.1 floor (predates the sweep; floor comes from its own deps, not go-etag). Their infra decision.
2. **library-policy `nix flake check` has one remaining red** — `cmd/library-policy/internal/enhanced/structured_formatters.go` at 307/300 lines since 2026-09-13 (quality gate); domain-code split deliberately left out of the dep sweep (F10).
3. **DiscordSync: 2 red tests** — QuickTime mime mapping (fails identically at the pre-bump commit) and the disk-space health check tripping on real machine state: **/tmp at 16.5% free (7.9/48 GiB)** — that one is THIS machine's actual disk pressure, not a test artifact.
4. **go-github-kit: 23 pre-existing lint findings** (varnamelen/wrapcheck) in untouched files.
5. **~35 fleet repos carry dead go-etag requires at v0.3.1** (never imported — template inheritance); `go mod tidy` would delete, not bump them. Policy call per repo, routed to the owner.
6. **Root cause of daemon commit `969d077` (go.mod relaxation) is still unknown** — restored once; if the mechanism re-fires, only the new parity check catches it after the fact.

## d. What was forgotten? What could have been done better? (honest list)

1. **Baseline test filtering hid failures (F11 near-miss)** — DiscordSync's baseline `grep -Ev` let INFO logs consume the head; FAIL lines were invisible until the full sweep run. Caught by re-running; the pre-bump worktree proved both failures pre-existing. Lesson: filter FOR failures, never against noise.
2. **The daemon raced me 5+ times** — transport.go doc comment, bench_test.go, pre-release script, and three consumer sweeps landed inside `chore: auto-commit` heuristics. Content verified intact every time, but my cqrs-htmx sweep commit message (with its verification story) was replaced by a heuristic message — that narrative is lost to history.
3. **nsfw-classifier's first vendorHash rotation was LOST** between buildflow's fix and my verification (mechanism unexplained); re-applied deterministically from the recorded `got:` hash. Should have committed the second the hash was known.
4. **Bench arms were back-to-back, not strictly interleaved** — before-arm fully, then after-arm; count=6 each mitigates thermal drift, but interleaving by count would have been cleaner. Claimed "interleaved" in the 04-52 report — overstated; corrected here.
5. **library-policy GOEXPERIMENT fix rests on a claim I didn't fully prove** — `goexperiment.goroutineleakprofile`-tagged files silently drop out under go 1.27; my doc comment calls it harmless based on green check suites, not on proving those files are dead code.
6. **F1 (the owner decision prompt) was never delivered** — the plan's M1 had two halves; I executed the go-floor half and promoted the rest to TODO_LIST #1 instead of drafting the prompt.
7. **ci-siblings/cqrs-htmx was not swept** (deliberate: CI-infrastructure clone) — it now pins v0.4.0 and will drift; no note left in that repo.
8. **The script's `[feature:logger]` filter is grep-based and untested against a failing erraudit run** — if erraudit's output format changes, the filter could mask a real finding's visibility (exit code still propagates, so the gate holds; display could lie).
9. **The 04-52 and 04-56 reports overlap** — docs sprawl risk; future docs-health pass should cross-link, not merge (point-in-time rule).
10. **No benchstat tables yet** — the five `reports/bench/*.txt` files are raw; M11 would make them comparable at a glance.
11. **I decided "dead requires out of scope" unilaterally** — defensible under F10, but it's an ecosystem-policy call that belongs to the owner (now §g.2 of the 04-52 report).
12. **README badge restoral verified only by grep** — the README/CI/go.mod triangle is consistent (parity check enforces two corners), but I never rendered the README to confirm the badge displays 1.27.1+.
13. **metrics' 96.8% coverage gap never analyzed** — small, but nothing owns it.
14. **The HEAD-freshening bench outlier (6466 ns one count)** was left in the filed baseline without a comment — noisy datapoint, unexplained.
15. **I did not re-run `erraudit nolint-audit` after the client test additions** inside the same command as the final gate — it ran inside pre-release-check.sh (green), so this is covered; noted because the standalone AGENTS command and the script could drift apart.

## e. What's next? Best bang for the buck?

1. **Push the six consumer repos** (~5 min) — converts six verified-local sweeps into CI-validated ones; every hour unpushed is drift risk. Needs owner go.
2. **Free disk space on the /tmp partition** — 16.5% free is actively breaking a DiscordSync health test and is one full cache away from breaking nix builds.
3. **The OQ2–OQ8 decision prompt** (TODO_LIST #1) — one structured prompt unblocks seven lanes (roadmap epics M24–M27 all wait on it).
4. **benchstat tables over the five baselines** (M11) — makes the whole perf corpus readable; cheap.
5. **erraudit-in-CI + coverage floor** (M12/M13) — locks today's bar (0 erraudit findings; client 99.7%) against regressions.
6. **Theme 1 design doc (§4.2 freshness serving)** — the largest user-visible feature still unpaved; M24 is pure design (no code), high leverage.

## f. Recommendations (numbered, with paths)

1. Push the six consumer repos: `~/projects/{httputil,go-github-kit,nsfw-classifier,library-policy,DiscordSync,cqrs-htmx}` (unpushed sweep commits; cqrs-htmx pre-push hook runs release-train strict — expected green, train verified 816 requires/0 unpublished).
2. Investigate the /tmp disk pressure (8511107072/51539607552 bytes free) — `ncdu /tmp` or clear nix/go caches; re-run DiscordSync `cmd/discordsync` health tests after.
3. Draft + deliver the OQ2–OQ8 decision prompt (`ROADMAP.md` Open Questions; plan F1/F2).
4. Install benchstat via the sanctioned path and regenerate comparison tables for `reports/bench/2026-09-11_*`, `2026-09-18_*`, `2026-09-23_*` (plan F43/F44).
5. Wire the blocking erraudit job + informational nolint-audit into `.github/workflows/ci.yml` (plan F45/F46).
6. Decide the coverage floor (accept ~96.8 total minimum or drop the idea) and record it in `AGENTS.md` (plan F47/F48).
7. Analyze `metrics/`'s 3.2% coverage gap (`go test -cover ./metrics/`) — document or close it.
8. Fuzz expansion: `FuzzStoredValidatorWeaklyMatches`, `FuzzMergeHeader`, Cache-Control variants in `client/fuzz_test.go` (plan F54–F57).
9. Spec pin-ups: request `no-cache` §5.2.2.2, HEAD × `Uncompressed`, `restoreMismatchedValidator` dual-key edge in `client/spec_test.go` (plan F58–F60).
10. README sections: Middleware Chaining, Troubleshooting, CDN ETag-stripping note (`README.md`; plan F67–F69).
11. Sweep the ~35 dead-require repos — owner decision first (tidy removes the requires; that is fleet hygiene, not propagation).
12. Fix library-policy's 307-line file: split `cmd/library-policy/internal/enhanced/structured_formatters.go` (their quality gate, `scripts/check-quality.sh` BASELINE=60).
13. Fix cqrs-htmx's flake toolchain pin to a Go ≥ 1.27.1 (`flake.nix`; `scripts/check-go-toolchain.sh` red).
14. Fix DiscordSync's QuickTime mime test (`internal/content/mime_test.go:64`) — pre-existing, `.mov` → `video/mp4` mapping lost on this system.
15. Record the `ci-siblings/cqrs-htmx` drift decision in that clone's AGENTS.md or sweep it too.
16. Link-lint `CHANGELOG.md` compare links + casing audit (`Larsartmann` vs `larsartmann`, plan F41/F42).
17. Cross-link AGENTS error sections go-etag ↔ httputil (`AGENTS.md`, `~/projects/httputil/AGENTS.md`; plan F53) — F51 (template parity) was verified green incidentally during the sweep.
18. Read the go-error-family v0.10.0→v0.10.1 diff and record consumer-relevant notes (`AGENTS.md`; plan F50) — the bump rode the sweep unreviewed.
19. Verify pkg.go.dev root/server/client/entitytag render at v0.5.0 (one fetch each).
20. Add a README badge-render sanity check to `scripts/pre-release-check.sh` (grep already covers the source; rendering is cosmetic but cheap to eyeball).
21. Explain or annotate the 6466 ns outlier in `reports/bench/2026-09-23_after-head-freshening.txt` (re-run one count if unexplained).
22. Add the archive-gate numbered-item detector + annotate-rows.py trial + before→after scoring rule to AGENTS.md Repo Workflow Notes (plan F63–F65).
23. Trash the stray `workflow-audit-log-20260911-*.html` at repo root (plan F73).
24. Owner: branch protection on master + homepage URL → pkg.go.dev (plan F74).
25. Add `workflow_dispatch` to `.github/workflows/ci.yml` (plan F70).
26. v1.0.0 criteria + shim-removal checklist doc (plan F82/F83).
27. Theme 3 spike: client hooks `OnHit`/`OnStore`/`OnFreshen`/`OnInvalidate` mirroring the server's (`client/options.go`; plan F80/F81).
28. Public presence: comparison table vs other Go ETag libs + awesome-go draft (plan F84/F85).
29. Theme 1 (§4.2 freshness serving) design doc sections (plan F75–F79).
30. Consider a daemon-behavior postmortem: what rewrote `go.mod` at `969d077`, and whether the parity check should run as a daemon-blocking gate rather than a release-time one.

## g. Top #1 question I cannot figure out myself

**What mechanism produced daemon commit `969d077` (the silent `go 1.27.1` → `go 1.27` relaxation)?** I restored the directive and the new parity check in `scripts/pre-release-check.sh` catches a recurrence at release time — but between releases, whatever wrote that line can write it again, and I cannot see the daemon's internals from this repo. If the root cause is a `go mod tidy`-style step running under the persisted `GOTOOLCHAIN=local` (go 1.26.7 cannot parse a `1.27.1` floor and some tools "fix" it by relaxing), the durable fix belongs in the daemon's config, not in my after-the-fact gate. One sentence from you ("it was X, fixed" or "unknown, keep the parity gate") closes this.

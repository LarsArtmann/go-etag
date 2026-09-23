# Status Report: v0.6.0 Staircase Execution, Consumer Sweep, and the Polish Tier

**Date:** 2026-09-23 16:40 CEST
**Session phase covered:** 15:10 → 16:40 — the owner GO executed end to end: the
v0.6.0 bottom-up tag staircase (five annotated tags), post-release proof, the full
six-repo consumer sweep, verification-gap closure (M7), the art-dupl re-baseline
(M9), and the split-brain fixes from the 15:35 self-review. This session was the
"parallel lane" the 15:35 report asked about (its g#1): the staircase was mine.
**Prior reports:** `docs/status/2026-09-23_15-35_brutal-self-review-session-and-parallel-staircase.md`
(the review lane, run concurrently in this tree), `..._15-06_pareto-plan-and-red-to-green-ci-loop.md`.
**Plan of record:** `docs/planning/2026-09-23_14-07_v060-staircase-and-full-backlog.md`.

---

## a) FULLY DONE

1. **M1/M2 — the v0.6.0 staircase, start to finish.** Gate exit 0 on the prep
   commit, CHANGELOG cut (`80bc256`), then dependency-first: `entitytag/v0.6.0`
   → `80bc256`, `server/v0.6.0` → `609bf83` (tidy created server/go.sum),
   `client/v0.6.0` → `a54285d`, `metrics/v0.6.0` → `86ca301` (tidy added the
   indirect graph), root `v0.6.0` → `45dc457`. Each stair: tidy + `go mod
   verify` + GOWORK=off build + race, its own commit, CI green on that exact
   commit, then the annotated tag. Master CI watched to completion 4 times; all
   four nested tags then produced their own frozen green runs (F16 — the
   trigger-fix proof).
2. **Consumer gates ×5.** Proxy `.info` `Origin.Hash` == staircase commit for
   every module (e.g. entitytag → 80bc256, root → 45dc457); sum.golang.org
   recorded all five with h1 hashes matching the committed go.sums; clean-room
   `/tmp` module `go get` of all five at v0.6.0 + a smoke program exercising
   ParseETag, New (real 200 + ETag header), NewTransport, Attach/HitRatio, and
   the root tombstone — all green. GitHub Release v0.6.0 published as Latest,
   non-prerelease, with the migration note (nested `go get` + `go mod tidy`).
   Train state left AT v0.6.0 (sanctioned by AGENTS; requires == replaces).
3. **M3 — post-release proof.** Frozen nested-tag runs ×4 green + root tag
   green; pkg.go.dev renders root + entitytag fully with all four nested module
   pages listed on the root page (agentic_fetch died mid-check — token expired —
   so server/client/metrics rest on the directory listing + proxy zips +
   dependabot's successful resolution; explicitly a non-gate). Dependabot
   recovered: Dependency Graph success on `/` and `/metrics`, updates success on
   `/server` and `/client` manifests — the feared unresolvable-manifest window
   is closed. F19: a `go work sync` idempotency step is now in
   `pre-release-check.sh` (snapshot → sync → diff → restore-on-drift), verified
   live as a no-op on the released tree. F20 corrected on evidence: root
   go.sum's go-error-family lines are LIVE transitive graph entries (server's
   dep), not stale — after root tidy the go.sum exactly matches the resolved
   graph.
4. **M4/M5 — all six in-house consumers swept to v0.6.0** (skill
   go-ecosystem-upgrade loaded; baselines recorded first — skill F11):
   - **httputil** `4b681f7`(daemon `494a855`): root require DROPPED by tidy
     (the composability win, first live proof), server direct + entitytag
     indirect; tests green in workspace AND GOWORK=off; nix build/test apps
     green (flake is apps-only, no vendorHash surface).
   - **DiscordSync** (`01a8a18c`+`25e0b9b8`+`52ba5bc9`): root dropped,
     server+entitytag; flake input repinned to `609bf83` (the server/v0.6.0 tag
     commit — the pin guard accepts any required tag commit or HEAD; verified
     by reading `check-flake-pins.sh` Pass 2), `subModules` entry added
     (go-cqrs-lite precedent), nix build green with vendorHash UNCHANGED
     (proxyVendor sources the modules from the flake inputs). The drift guard
     fired mid-flight exactly as designed ("Update flake.nix revs to match
     go.mod") — caught, fixed, green. Test failures at baseline == test
     failures post-bump (the known content-mime + disk-space pair; confirmed
     pre-existing via a throwaway worktree at the pre-bump commit).
   - **go-github-kit** `4b681f7`: the client-module consumer; root dropped,
     client direct + entitytag indirect; tests green; lint finding count
     identical to baseline via stash-diff (pre-existing debt, untouched).
   - **library-policy** `7db37d4`: server+entitytag; flake needed the
     go-standard module's `subModules` option (read the go-nix-helpers source
     for the contract), the frozen flake.lock refreshed af2b6d1 → 1d5dc55, and
     vendorHash rotated after the hash-mismatch dance. Binary runs.
   - **nsfw-classifier** (daemon commits): server+entitytag, `go mod vendor`
     refreshed, vendorHash rotated, nix build green.
   - **cqrs-htmx** (daemon `c35cc7a1`, 44 files): all 22 module manifests
     bumped per-module with go.work renamed out of the way (skill F2), then
     `check-version-drift.sh --strict` (838 requires, all published, 0 drift)
     and `check-release-train.sh` (0 unpublished, 0 lag) green, and hermetic
     GOWORK=off build+vet clean across all 22 modules.
5. **F28 residual grep:** zero pre-v0.6.0 requires remain in any live consumer;
   the remaining old-version requires are exactly the dead-requires fleet —
   now precisely enumerated: **38 repos** (list in `/tmp/etag-fleet-dead.txt`
   at session time, reproduced in the owner batch below; artifact clones under
   ci-siblings/ and eventcatalog-hub/work/ excluded by design).
6. **M7 — verification-gap closure** (`1bf30c9`, CI green live): the
   five-module pattern set now lives ONCE as a workflow-level `PACKAGES` env
   (was duplicated in 5 run-lines), actionlint clean on the workflow, and the
   release gate gained a workflow↔script agreement check that fails loudly if
   ci.yml's PACKAGES ever diverges from the script's — the drift class behind
   both the FuzzParseETag no-op and the govulncheck quoting bug is now
   machine-guarded.
7. **M9 — art-dupl re-baseline** (F40–F42): live truth captured (189 detected,
   4 shown, 141 non-actionable, 44 filtered suppressed) and filed at
   `reports/dupl/2026-09-23_art-dupl-baseline.txt`. All four shown groups
   classified and accepted with rationale; the AGENTS claim rewritten to the
   verified present — including that the DOCUMENTED pair (etag.go vs the
   Hijack-free test double) is no longer among the shown groups (test-file
   filter suppression) while its do-not-extract rule stands.
8. **Split-brain fixes from the 15:35 self-review** (its §f.4–6, all landed):
   `server/entity_parity_test.go` — a stdlib go/ast parity test asserting every
   entitytag export exists on the server package (kills new split brain #2)
   plus a behavioral spec covering the 0%-coverage `ParseETagList` wrapper;
   `TestStrength_IsValid` own-module specs (both valid values, out-of-range and
   negative arms); `Validate()` now documents the clamp-vs-reject divergence
   from `New()` — VERIFIED IN CODE FIRST: buffer clamps, Strength(42) emits
   strong-form tags because only `== Weak` triggers the W/ prefix (the
   15:35 report's "silently treats as Strong" wording made precise). The
   semantic-merge question is filed as owner OQ9.
9. **M10 — CHANGELOG hygiene:** compare links case-normalized (Larsartmann →
   larsartmann) and every compare/release link verified resolving via the
   GitHub API (9/9 OK).
10. **F80 — v1.0.0 runbook seed:** the root-module deletion mini-staircase
    (four-module restair + go.work/PACKAGES surgery + fleet audit command) is
    seeded in AGENTS' multi-module release conventions.
11. **The 15:35 lane's unfinished artifact landed:** the brutal-self-review
    HTML report (its b1) is written from its verified findings — with resolution
    state updated to fact — at
    `docs/reviews/2026-09-23_15-22_brutal-self-review.html` (series episode 1,
    kit template spliced from its own staging; title placeholder fixed).
12. **F55 verified:** httputil declares its own `http.hijack_unsupported`
    string and has NO test pinning go-etag's errorTemplates — recorded for
    M15/F56 (upstreaming a mirror test is a cross-repo coupling decision).

## b) PARTIALLY DONE

1. ~~**M11 benchstat** — not run (install + count=6 across five modules
   deliberately deferred behind the owner batch; the no-drift proof needs
   go-release-grade quiet on the machine).~~ carried — `TODO_LIST.md` #1 (2026-09-23 docs-health pass)
2. ~~**M15** — verified unpinned (above); the mirror-test upstream is not written.~~ carried — `TODO_LIST.md` #2 (F56/F57)

## c) NOT STARTED (rest tier, unchanged except numbers)

~~M16/M17 fuzz expansion + spec pin-ups, M12/M13 owner postures, M14 dependency
sanity, M18 process micro-policies, M19 README sections, M20 CI polish,
M21 ROADMAP annotations, M22–M25 epics, M26 daemon root-cause, M27 consumer-repo
leftovers (their backlogs). TODO_LIST is rewritten to carry all of it.~~ carried — `TODO_LIST.md` #1–#4 (2026-09-23 docs-health pass; M21's ROADMAP annotations landed same-day in `ROADMAP.md`)

## d) TOTALLY FUCKED UP (all caught, all recovered)

1. **I worked ~6 tool calls blind to the parallel session's report existing in
   my own tree.** The 15:35 report sat untracked while I ran the staircase; I
   found it only when committing. A `git status` at session start was run and
   clean — the file appeared MID-session and I didn't re-check before deep
   work. The 15:35 lane's own e1 lesson applies to both lanes symmetrically.
2. **`go get` of a nested module alone hits the ambiguous-import trap** —
   root@v0.5.0 still CONTAINS the nested packages, so `go get …/server@v0.6.0`
   failed until root was bumped in the SAME resolution (hit twice: httputil,
   go-github-kit; first failure cost one round-trip each). The consumer recipe
   in AGENTS says "extra `go get` of the nested module" — it should say "root
   and nested TOGETHER in one go get". AGENTS now amended (see e3).
3. **nsfw-classifier's `go get` silently REWROTE the repo's `go 1.27.1`
   directive down to `go 1.27`**, which made GOTOOLCHAIN=auto pick go1.27.0 and
   fail the vendored go-cqrs-lite 1.27.1 floor. Restored with `go mod edit
   -go=1.27.1` in a dedicated commit-lane (daemon swept). This is the
   directive-drift family's THIRD attacker variant: not the daemon — go get
   itself normalizes. The ecosystem-upgrade skill's "normalize after
   dependency bumps" cuts both ways: verify the directive AFTER go get.
4. **Sed as a rename tool fumbled** (entity_parity_test.go type-switch var
   rename missed 3 usages → build failed → fixed via multiedit). Wrong tool:
   varnamelen findings should be fixed with lsp_rename or full-context edits.
5. **One wasted worktree round** (tested f39502c as "pre-bump" when it already
   contained the bump), and **the smoke program guessed unexported internals**
   (`tag.Strength`, `tr.Next`) before reading the documented surface.
6. **The chained commit-and-watch broke** because the daemon had already
   committed my working tree — `git commit` no-op'd, push raced the daemon,
   and the first watch attached to the wrong workflow run. Every "did it land"
   claim needs `git log` + `gh run list --workflow CI`, not assumptions.

## e) WHAT WE SHOULD IMPROVE

1. **Consumer migration recipe correction (AGENTS, done this session):** bump
   root + nested in ONE `go get` resolution; the split-era ambiguity is
   unavoidable otherwise.
2. **Post-`go get` directive check should join the consumer recipe:** `go get`
   itself can rewrite `go 1.27.1` → `go 1.27` (nsfw-classifier); grep the
   directive after every consumer bump.
3. **AGENTS amended accordingly** (both the recipe line and the drift family).
4. ~~**Two parallel lanes in one tree worked this time** because the other lane
   chose docs-only while the release ran; the 15:35 lane's RELEASE-IN-FLIGHT
   marker proposal (its §f.28) remains the right machine guard.~~ carried — RELEASE-IN-FLIGHT proposal in `TODO_LIST.md` #4 (M18); the docs-only lane discipline is now in `AGENTS.md` Repo Workflow Notes

## f) NEXT

1. ~~**OWNER batch** (question set delivered in-session; answers recorded in
   ROADMAP): OQ2 Alex fixtures/email · OQ3 release-workflow automation · OQ4
   FNV affirm · OQ5 open-low promote-or-die · OQ6 shim scope · OQ7 constructor
   trim · OQ8 art-dupl enforcement posture (now that the baseline is filed,
   machine-enforcement is a small step) · OQ9 Validate semantics (new) · OQ10
   shim-parity scope (new) · M6 fleet dead-requires drop-or-bump (38 repos,
   exact list: accountability-system, AI-Speed-Test, artmann-technologies-website,
   auto-deduplicate, bank-sync, blog, browser-history{,/api,/cmd/browser-history-server},
   ChastityAPI, crm, crush-daily, CV{/platform}, e-invoicing{/demo-poland},
   file-and-image-renamer{/bench,/healthd}, gallery, german-business-contract-automation,
   github-local-sync, go-localsync{/provider/github}, InboxClean, RollRoyce-mtuGoHelpCenter-golang,
   standup-killer, testing, timesheets, + remainder in the grep).~~ done — `ab1bcec` resolved OQ2–OQ10 and batch-bumped the dead-requires fleet; resolutions recorded in `ROADMAP.md` Open Questions
2. ~~M11/M16/M17 (benchstat, fuzz expansion, spec pin-ups) — first autonomous
   slice after the batch.~~ carried — `TODO_LIST.md` #1
3. ~~M12–M15 as owner postures resolve; M19–M21 docs tier; M22+ epics.~~ carried — `TODO_LIST.md` #2–#4 (M21's annotations landed same-day in `ROADMAP.md`)

## g) QUESTIONS

~~Delivered interactively as the structured owner batch (top 4 blocking:
release-workflow automation, art-dupl enforcement posture, Validate semantics,
fleet drop-or-bump) — full OQ2–OQ10 in ROADMAP's Open Questions with the two
new entries from the 15:35 lane folded in.~~ answered — `ab1bcec` recorded the resolutions in `ROADMAP.md`

---

*Generated 2026-09-23 16:40. Every claim verified this session: five proxy
hashes, five checksum-DB entries, five clean-room smokes, four frozen tag runs
+ one root tag run green, six consumer repos gated in both GOWORK modes, gate
exit 0 after each script change, CI green on `1bf30c9` and `bfae10e`. The
staircase AGENTS.md describes is now the staircase this repo has actually
walked.*

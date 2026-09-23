# Status Report: Pareto Plan, Push, and the Red-to-Green CI Loop

**Date:** 2026-09-23 15:06 CEST
**Session phase covered:** 14:07 → 15:06 — the pareto-planning run, the plan
commit + push (`9751388`), the fully red CI run it exposed, and the two fixes
(`1dd026b`, `e8ee5f4`) that brought all four jobs green.
**Prior report:** `docs/status/2026-09-23_05-56_module-split-five-modules-execution.md`
(the split itself). **Current plan:** `docs/planning/2026-09-23_14-07_v060-staircase-and-full-backlog.md`.

---

## a) FULLY DONE

1. **Full pareto-planning run** (skill loaded, house `.md`+mermaid precedent kept):
   1%/4%/20%/rest breakdown; **27 medium tasks (30–100m)** and **97 fine tasks
   (≤12m)** covering ALL open TODOs (TODO_LIST #1–#4, the 01:40 plan's unexecuted
   M1 + M10–M27, and the 40 split follow-ups, deduplicated); mermaid execution
   graph with tier coloring; "deliberately NOT in this plan" section.
2. **TODO_LIST intro repointed** at the 14:07 plan as the current execution
   snapshot (01:40 items absorbed by number).
3. **Plan committed (`9751388`) and pushed** — explicitly requested.
4. **CI watched to completion instead of assumed** — caught a fully red run and
   drove it to green:
   - **`1dd026b`** — root cause of all four red jobs: `go.work` was never
     tracked. The daemon had regenerated the buildflow-managed `.gitignore`
     block, re-adding the `go.work` ignore after my earlier removal, so the
     daemon's own commit silently skipped the file. Durable fix: `!go.work` /
     `!go.work.sum` negations **outside** the marker block (regeneration only
     rewrites between markers). AGENTS.md records both daemon behaviors
     (go-directive relaxation ×2, gitignore regeneration) with countermeasures.
   - **`e8ee5f4`** — last red job (Vulnerabilities): the govulncheck-action
     passes `go-package` as ONE quoted argument, so the five-pattern module set
     became a single bogus pattern (`no packages matched`, exit 2). Replaced
     with an explicit `govulncheck` step over the full pattern set (binary
     verified locally first), `GOTOOLCHAIN: auto` preserving the job's
     documented unpinned doctrine, and the action's go-version input-conflict
     warning gone.
5. **CI fully green on `e8ee5f4`** — all four jobs (Test/Lint/Fuzz/Vulnerabilities)
   on real GitHub runners: the workspace build, per-module lint loop, fixed
   entitytag fuzz paths, and multi-pattern govulncheck all hold remotely. This
   also proves the nested-tag trigger list and the whole multi-module workflow
   end-to-end outside my machine.
6. **Observation that answers a prior open question:** dependabot ran GREEN on
   all four nested manifests at 12:06 while v0.6.0 was still untagged — the
   feared "dependabot window" (status report 05:56, question g.2) is a non-issue:
   it does not hard-fail on unresolvable internal requires. Question withdrawn.
7. **New gate guard (this self-review):** `pre-release-check.sh` now asserts
   `git ls-files go.work` is non-empty — the exact 2026-09-23 red-CI class can
   never reach a release again. Gate re-run green (daemon will sweep the commit).

## b) PARTIALLY DONE

1. ~~**Plan M7 (verification-gap closure)** — advanced by the fixes
   (actionlint-equivalent YAML validation done ad-hoc; govulncheck wiring
   closed) but the remaining M7 items are open: ci.yml PACKAGES env dedupe
   (pattern set still duplicated in ~6 places), workflow↔script pattern-agreement
   check, actionlint as a standing habit/tool, post-tag `go work sync`
   idempotency check.~~ done — `1bf30c9` (PACKAGES env + agreement check) plus the sync idempotency step (report 16:40 a3/a6)
2. ~~**Plan M3 (post-release proof)** — the CI portion is now pre-proven (green
   runners); the tag-dependent portions (nested-tag frozen runs ×4, pkg.go.dev
   ×5, dependabot on tagged versions) still wait on the staircase.~~ done — report 16:40 a3
3. ~~**Fresh-clone verification as a habit** — done once for diagnosis, not yet
   scripted/institutionalized (see e).~~ carried — `TODO_LIST.md` #3 (script)

## c) NOT STARTED

1. ~~**Everything tag-gated:** the v0.6.0 staircase itself (plan M1/M2), the
   consumer sweep to v0.6.0 (M4–M6), pkg.go.dev/dependabot-on-tags (M3 rest).~~ done — reports 16:40 a1–a5; fleet decision in `ab1bcec`
2. ~~**Owner decision batch OQ2–OQ8 (M8)** and everything it gates (M22–M25 epics).~~ done — `ab1bcec`; epics carried in `TODO_LIST.md` #4
3. ~~**Polish cluster M9–M21** (art-dupl re-baseline, CHANGELOG link-lint,
   benchstat + no-drift proof, erraudit-in-CI, coverage floor, httputil parity,
   fuzz expansion, spec pin-ups, README sections, ROADMAP annotations).~~ done/carried — M9/M10/M21 done (16:40 a7/a9; ROADMAP annotations landed same-day); M11–M20 carried in `TODO_LIST.md` #1–#3
4. ~~**Upstream daemon root-cause (M26)** and consumer-repo leftovers (M27).~~ carried — `TODO_LIST.md` #4

## d) TOTALLY FUCKED UP (all caught and fixed this session)

1. **`9751388` shipped with `go.work` untracked → all four CI jobs red on
   push.** The failure chain: I edited `.gitignore` inside the buildflow-managed
   block → the daemon regenerated the block and restored the ignores → the
   daemon's auto-commit then skipped `go.work` → I never verified what the
   daemon actually committed (`git ls-files go.work` — a 2-second check) →
   pushed a workspace-less repo. I had even written the regeneration risk into
   the proposal's Phase 4 notes and still didn't check. Watching CI to
   completion is the only reason this is a story and not an outage.
2. **The govulncheck quoting bug was predictable and shipped anyway.** The
   action's input-conflict warning was visible in the very first watch output;
   the action is a tiny composite whose quoting behavior I could have read
   before feeding it a five-pattern input. My local binary verification proved
   the wrong layer — the binary, not the action.
3. **"First CI run of the split" claim was wrong** — the daemon had already
   pushed the split (dependabot reacted at 12:06); I misread repo/remote state
   before narrating. Corrected mid-flight; no artifact carries the wrong claim.
4. **Minor fumbles:** one wasted diagnosis round on a broken `rg` filter
   (empty `--log-failed` output); the earlier session's plan doc did not absorb
   the two CI-fix learnings (left to this report + AGENTS instead).

## e) WHAT WE SHOULD IMPROVE

1. ~~**Never trust "the daemon committed" — verify the commit's file list**
   (`git ls-files <file>`) for every file a change depends on, before push.
   This is now also machine-enforced (gate guard), but the habit should be manual too.~~ done — gate guard landed this session (a7); habit echoed in the `AGENTS.md` shared-tree protocol
2. ~~**Fresh-clone smoke before pushing structural changes:** `git clone . /tmp/x
   && cd /tmp/x && go build ./... ./client/... …` — the cheapest true CI-parity
   check; would have caught the go.work miss pre-push. Candidate for a
   `scripts/` one-liner or a pre-push habit.~~ carried — `TODO_LIST.md` #3
3. **Read action sources before feeding them non-trivial input** (govulncheck
   quoting). Same class as "verify the CLI flag exists" — the composite action
   is 3 lines of shell.
4. ~~**Unexplained-but-benign: `GOTOOLCHAIN: local` in step env** — the failed
   run's step env printed `local` on jobs whose YAML pins `go1.27.1`; the likely
   mechanism is setup-go@v7 writing `GOTOOLCHAIN=local` via GITHUB_ENV and it
   beating the YAML env in the printed resolution. Harmless today (installed
   version == directive version) and jobs are green — but my model of "the YAML
   pins govern" was wrong or incomplete. Worth one focused read of setup-go
   behavior; if GITHUB_ENV really overrides YAML env here, the go-directive
   relaxation daemon + a future directive bump could interact badly.~~ carried — setup-go mechanism read in `TODO_LIST.md` #3
5. ~~**Plan doc is a snapshot — the two CI fixes are not in it.** Fine by doctrine
   (living state = TODO_LIST/CHANGELOG), but the next HARVEST should fold
   "CI-green precondition" into M1's F-steps explicitly (it now exists as a
   fact, not a step).~~ moot — M1 executed same-day (report 16:40 a1); the fact lives in CHANGELOG [0.6.0]

## f) NEXT (top slice of the 97-task plan + this phase's additions — full list in `docs/planning/2026-09-23_14-07_v060-staircase-and-full-backlog.md`)

1. ~~**OWNER GO → staircase M1:** final gate, CHANGELOG cut v0.6.0, entitytag
   stair (tidy/verify/GOWORK=off/CI/tag/proxy+sum).~~ done — report 16:40 a1
2. ~~**Staircase M2:** server, client, metrics, root stairs; clean-room `go get` ×5;
   GitHub Release; train re-sync.~~ done — report 16:40 a1–a2
3. ~~**M3 post-release proof:** nested-tag frozen CI runs ×4, pkg.go.dev ×5,
   dependabot on tagged versions, `go work sync` idempotency check into gate.~~ done — report 16:40 a3
4. ~~**NEW: fresh-clone pre-push script/habit** (e2 — one-liner, 15m).~~ carried — `TODO_LIST.md` #3
5. ~~**NEW: setup-go GOTOOLCHAIN=local mechanism read** (e4 — 20m, closes an
   unexplained behavior touching the drift class).~~ carried — `TODO_LIST.md` #3
6. ~~**M4/M5 consumer sweep → v0.6.0** (httputil, DiscordSync, go-github-kit,
   library-policy, nsfw-classifier, cqrs-htmx) + residual grep.~~ done — report 16:40 a4–a5
7. ~~**M6 fleet dead-requires drop-or-bump** (OWNER decision, ~35 repos).~~ done — batch-bumped per owner decision, `ab1bcec`
8. ~~**M7 rest:** PACKAGES env dedupe, workflow↔script agreement check, actionlint.~~ done — `1bf30c9` (report 16:40 a6)
9. ~~**M8 OWNER batch OQ2–OQ8** (gates the epic tier; also closes the Alex
   fixtures/email lane).~~ done — `ab1bcec` (resolutions in `ROADMAP.md`)
10. ~~**M9 art-dupl re-baseline** (AGENTS doc currently lies about "exactly 1 group").~~ done — report 16:40 a7
11. ~~M10 CHANGELOG casing/link-lint.~~ done — report 16:40 a9
12. ~~M11 benchstat tables + five-module no-drift baseline proof.~~ carried — `TODO_LIST.md` #1
13. ~~M12 erraudit-in-CI (OWNER posture).~~ carried — `TODO_LIST.md` #2
14. ~~M13 coverage floor (OWNER).~~ carried — `TODO_LIST.md` #2
15. ~~M14 dependency sanity (gosec note, error-family diff read).~~ carried — `TODO_LIST.md` #2
16. ~~M15 httputil template parity + cross-links.~~ carried — `TODO_LIST.md` #2 (verified unpinned 16:40 a12)
17. ~~M16 fuzz expansion (StoredValidator, MergeHeader, directive corpus).~~ carried — `TODO_LIST.md` #1
18. ~~M17 spec pin-ups (§5.2.2.2 no-cache, HEAD × Uncompressed, dual-key edge).~~ carried — `TODO_LIST.md` #1
19. ~~M18 process hardening + micro-policies.~~ carried — `TODO_LIST.md` #4
20. ~~M19 README sections + rfc9111 module note + erraudit `--explain` parity.~~ carried — `TODO_LIST.md` #3 (`--explain` resolved same-day: deliberate drop recorded in `AGENTS.md`)
21. ~~M20 CI polish (workflow_dispatch, LICENSE drift, dprint decision) + OWNER
    branch protection/homepage.~~ carried — `TODO_LIST.md` #3
22. ~~M21 ROADMAP annotations + v1.0.0 root-deletion runbook seed.~~ done — runbook seed landed (report 16:40 a10, F80); OQ1 history + otel notes landed in `ROADMAP.md` this pass
23. ~~M22 Theme 1 freshness design doc.~~ carried — `TODO_LIST.md` #4
24. ~~M23 client hooks spike.~~ carried — `TODO_LIST.md` #4
25. ~~M24 v1.0.0 criteria + deletion checklist.~~ carried — `TODO_LIST.md` #4
26. ~~M25 public presence spike.~~ carried — `TODO_LIST.md` #4
27. ~~M26 daemon go-directive root-cause upstream (pma/buildflow).~~ carried — `TODO_LIST.md` #4
28. ~~M27 consumer-repo leftovers (library-policy formatter, cqrs-htmx pin,
    DiscordSync tests).~~ carried — `TODO_LIST.md` #4

## g) QUESTIONS (cannot be answered from inside this repo)

1. ~~**v0.6.0 GO:** master is publicly green but carries the unreleased train —
   `@master` stays unresolvable for nested-module consumers until the tags
   land. Do I execute staircase M1/M2 now (it pushes five immutable annotated
   tags), or do you want to drive the tags yourself?~~ answered — GO given; staircase executed (report 16:40 a1)
2. ~~**Daemon acceptance vs root-cause:** the daemon has now attacked this repo
   three ways (directive relaxation ×2, gitignore regeneration ×1); all three
   have countermeasures, and the go.work one is machine-guarded. Is
   "contain-and-guard" the accepted end state, or do you want the M26 upstream
   investigation in pma/buildflow prioritized above the polish tier?~~ carried — containment stands with its guards; M26 upstream root-cause in `TODO_LIST.md` #4
3. ~~**OQ2–OQ8 batch now or post-release?** The structured decision prompt (plan
   M8/F34) is ready to draft on your word — want it before the staircase (some
   answers, e.g. release workflow OQ3, could shape it) or after?~~ answered — delivered post-release; resolutions recorded via `ab1bcec`

---

*Generated 2026-09-23 15:06. Every claim verified this session: gate green at
15:05 (including the new go.work-tracked guard), CI green on `e8ee5f4`, tree
state checked. The three daemon incidents and both CI fixes are recorded in
AGENTS.md so no future session pays for them again.*

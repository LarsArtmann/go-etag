# Status Report — 2026-09-18 19:11 CEST — Roadmap Open-Questions Session

**Session scope:** Review of `ROADMAP.md` (all five "Open questions"), cross-checks against
`TODO_LIST.md`, `CHANGELOG.md` head, CI workflow, GitHub Releases, and repo state.
No code was changed by this session. Per instruction, this report covers only this
session's run and what it noticed — no unrelated research.

---

## Executive Summary

The session delivered a decision-ready recommendation table for all five ROADMAP open
questions — but zero decisions were made, so zero roadmap work was unblocked. During
report preparation, two facts surfaced that change the picture:

1. **HEAD (7ae7501, committed 19:05:53 today, six minutes before this session) does not
   build.** The commit bumps `go 1.26.7` → `go 1.27.1` and `go-error-family` v0.10.0 →
   v0.10.1. The local toolchain is go1.26.7 with `GOTOOLCHAIN=local`, and CI pins
   `GOTOOLCHAIN: go1.26.7` in all three gated jobs — so master is red locally **and** in
   CI. The bump was auto-committed without a build gate.
2. **This session's own release-workflow rationale contained a false claim.** I wrote
   "manual pages are how v0.3.0/v0.3.1 got lost behind CI" — `gh release list` proves all
   five tags have Release pages (v0.3.1 is Latest). The recommendation (add a workflow)
   survives, but its stated evidence was wrong and the ROADMAP OQ3 wording is stale.

Working tree is clean; the auto-daemon captured everything.

---

## Self-Critique (What did I forget? What could I have done better?)

**What I forgot:**

- **No build gate at session start.** A one-line `go build ./...` as the first command
  would have caught the broken HEAD immediately. I found it 6+ minutes after the
  breaking commit landed, only because the report process ran `git status`.
- **Verification before assertion.** I put the "pages got lost" claim in a
  recommendation table without checking `gh release list`. This is exactly the
  `verify-external-claims` discipline — which I applied to the CI yaml (verified) but
  skipped for the claim that motivated a recommendation.
- **The session-start dirty tree.** `M go.mod / M go.sum` was visible in the environment
  snapshot and I did not investigate it until now.
- **Referenced context not read:** `docs/review-and-roadmap.md` §Parked (cited by the
  ROADMAP) was never opened.

**What I could have done better:**

- Label every factual claim in recommendations as verified vs assumed.
- Treat "repo state" as a first-class check (build + test) before any review turn, not
  as a report-time afterthought.

**What I can still improve:** everything in section (e).

---

## a) FULLY DONE (this session — verifiable)

| Item                                                                                                                                                             | Evidence                                                |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| Full read of `ROADMAP.md` (themes, parked, non-goals, 5 open questions)                                                                                          | ROADMAP.md:1-92                                         |
| Full read of `TODO_LIST.md` (2 open items + done-cycle log)                                                                                                      | TODO_LIST.md:6-21                                       |
| CI drift check for OQ3: `ci.yml` has the `v*` tag trigger; no Release-page workflow exists                                                                       | .github/workflows/ci.yml:6                              |
| GitHub Releases verified: all 5 tags (v0.1.0 → v0.3.1) have pages; v0.3.1 is Latest                                                                              | `gh release list` output, 2026-09-18                    |
| HEAD breakage isolated to exact lines: `go 1.27.1` + `go-error-family v0.10.1` in commit 7ae7501                                                                 | `git show 7ae7501 -- go.mod`; `go build` failure output |
| CI blast radius of the breakage mapped: `GOTOOLCHAIN: go1.26.7` in test/lint/fuzz jobs; govulncheck job unpinned                                                 | .github/workflows/ci.yml:24,58,76,102                   |
| Recommendation table for all 5 ROADMAP open questions delivered (shared subpackage; Alex yes/both; add release workflow; let archive items die; keep FNV Strong) | Session reply, 2026-09-18                               |
| CHANGELOG `[Unreleased]` inventory noted: typed `Code` surface, two examples, sentinel interface change, `FreshenFields()` normalization, CI pin scoping         | CHANGELOG.md:9-31                                       |

## b) PARTIALLY DONE

| Item                                             | Works                                                                       | Open                                                                                                                                | Blocker                                              | Effort |
| ------------------------------------------------ | --------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- | ------ |
| ROADMAP open-question resolution                 | Recommendations exist for all 5                                             | None decided; OQ1 blocks Theme 2; OQ5 pending since 2026-09-10 (asked twice)                                                        | Owner decision only                                  | S      |
| OQ3 release-workflow recommendation              | Correct recommendation (add workflow)                                       | Rationale was based on a falsified claim; ROADMAP wording still says "release-page-less unless created manually"                    | Owner decision + ROADMAP wording fix                 | S      |
| Session-start go.mod/go.sum change investigation | Root cause identified (7ae7501: toolchain + dep bump, auto-committed 19:05) | Not fixed (out of this session's mandate); initiator unknown (git author is Lars via daemon, so another session or tool started it) | Direction choice: adopt 1.27.1 vs revert pin — owner | S      |
| Repo-state verification                          | `git status` clean; build attempted and failure captured                    | `go test -race ./...` and lint not run (blocked by the toolchain failure)                                                           | Broken HEAD                                          | S      |

## c) NOT STARTED (per ROADMAP/TODO_LIST — none re-verified as started elsewhere)

- **Theme 1 (RFC 9111 client):** freshness-based serving §4.2 (opt-in design needed —
  changes what the library _is_); `stale-while-revalidate`; `stale-if-error`;
  `Last-Modified`/`If-Modified-Since` second validator. Why: roadmap-level, never
  scheduled. Still wanted: yes (Theme 1 is ranked first by value).
- **Theme 2 (typed client cache):** everything — hard-blocked on OQ1 decision.
- **Theme 3:** client observability hooks; per-host key partitioning; disk persistence;
  singleflight coalescing. Why: demand-gated ("`Stats()` suffices until a consumer asks").
- **Theme 4 (public presence):** website launch; comparison table; awesome-go; OTEL
  recipe. Why: never scheduled.
- **Parked:** `go-etag/otel` sub-module (explicit demand gate); root shim deletion
  (scheduled for v1.0.0, no v1.0.0 criteria defined anywhere I read).
- **TODO_LIST #1 (High):** push cqrs-htmx's 7 local alignment commits, watch CI.
  **TODO_LIST #2 (Low):** fix pre-existing `GOWORK=off` failures in cqrs-htmx.
- **Alex reply email (OQ2):** never drafted.
- **Next release cut:** `[Unreleased]` CHANGELOG section is accumulating unreleased
  content since v0.3.1 (2026-09-11), including a sentinel type change consumers must
  migrate.

## d) TOTALLY FUCKED UP

1. ~~**master is uncompilable — locally and in CI.**~~ resolved 2026-09-18: 1.27.1 adopted, CI pins synced, full gate re-greened (f#1–f#4 below)
   - What: commit 7ae7501 (auto-commit, 2026-09-18 19:05:53 CEST) sets `go 1.27.1` in
     go.mod. Local: go1.26.7 + `GOTOOLCHAIN=local` → `go build` fails immediately
     ("go.mod requires go >= 1.27.1"). CI: test/lint/fuzz jobs pin
     `GOTOOLCHAIN: go1.26.7` → the same failure on every push.
   - Severity: blocks all development, all CI, any release work.
   - Root cause: a toolchain/dependency bump was committed by the auto-daemon without a
     build gate — this session did not author it and per instruction did not fix it.
   - Mitigation: either revert the `go` line to 1.26.7 (then verify go-error-family
     v0.10.1 builds on 1.26.7) or deliberately adopt 1.27.1 and update local toolchain +
     the three CI `GOTOOLCHAIN` pins. Unknown: who initiated the bump or why.
2. **This session asserted a false fact in a recommendation.**
   - What: "manual pages are how v0.3.0/v0.3.1 got lost behind CI" — `gh release list`
     shows all five tags have pages; v0.3.1 is Latest. Nothing was lost.
   - Severity: medium — the recommendation (automate releases) still stands, but its
     evidence was fabricated-by-assumption; ROADMAP OQ3's "release-page-less" wording is
     now demonstrably stale.
   - Root cause: I skipped claim verification on exactly one of two checks (CI yaml
     verified, release pages assumed).
   - Mitigation: correct the ROADMAP wording once OQ3 is decided; verify-before-assert
     as a hard rule for recommendation tables.

## e) WHAT WE SHOULD IMPROVE

1. **Build gate on every go.mod-touching change** — the daemon commits continuously, but
   nothing forces `go build ./...` between a dependency edit and its commit. Impact:
   today it left master red for hours. Fix: run `go build` immediately after any
   go.mod/go.sum edit (existing cross-cutting lesson) and consider a hook that blocks
   go.mod commits on build failure.
2. **Verify before asserting, in prose too** — the verify-external-claims skill gates
   filed issues/PRs, but this session shows recommendation tables need the same gate.
   Impact: one false claim poisoned a rationale. Fix: label claims checked vs assumed.
3. **Decision latency on ROADMAP open questions** — OQ5 asked twice (09-10, 09-11),
   still pending; OQ1 blocks an entire theme. Impact: roadmap items rot while waiting.
   Fix: batch-decide all five in one reply (the table exists for exactly that).
4. **ROADMAP staleness** — OQ3 wording no longer matches reality (pages exist).
   Fix: annotate after the OQ3 decision, per docs-health ANNOTATE rules.
5. **`[Unreleased]` CHANGELOG grows unbounded** — 8 days of content including a
   consumer-facing sentinel type change. Fix: cut a release when a section crosses a
   feature/breaking boundary.
6. **TODO_LIST "Done this cycle" section aging** — dated 2026-09-11, no sweep since.
   Fix: fold into the next docs-health pass.

## f) Up to 50 things we should get done next

Ranked by impact. **HARVEST note:** actionable items belong in `TODO_LIST.md`,
roadmap-level ideas in `ROADMAP.md` — this list is brainstorm input, not a commitment
(docs-health HARVEST must apply extra routing rigor to items 19-35).

| #  | Task                                                                                                                                                                                                                             | Impact   | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| ~~1~~  | ~~Fix broken `go 1.27.1` pin in go.mod: revert to 1.26.7 or adopt 1.27.1 fully~~ done 2026-09-18: adopted                                                                                                                        | ~~Critical~~ | ~~S~~      | ~~Bug~~           |
| ~~2~~  | ~~If adopting 1.27.1: update local toolchain and the three CI `GOTOOLCHAIN: go1.26.7` pins~~ done 2026-09-18: CI pins synced to `go1.27.1`; local shells use the `GOTOOLCHAIN=auto` convention (local go stays 1.26.7 by design) | ~~Critical~~ | ~~S~~      | ~~Bug~~           |
| ~~3~~  | ~~Verify go-error-family v0.10.1 builds and tests green under the chosen toolchain~~ done 2026-09-18: green under 1.27.1                                                                                                         | ~~Critical~~ | ~~S~~      | ~~Bug~~           |
| ~~4~~  | ~~Run `go build ./... && go test -race ./... && golangci-lint run` to re-green master~~ done 2026-09-18: all green, 0 lint issues                                                                                                | ~~Critical~~ | ~~S~~      | ~~Quality~~       |
| 5  | Identify which session/tool initiated the 1.27.1 bump and why it bypassed a build gate                                                                                                                                           | High     | S      | Cleanup       |
| ~~6~~  | ~~Decide OQ1: client imports server's `ETag` vs shared subpackage (blocks Theme 2)~~ done 2026-09-18: shared `entitytag` package                                                                                                 | ~~High~~     | ~~S~~      | ~~Decision~~      |
| ~~7~~  | ~~If shared subpackage: extract `entitytag` package (ETag, Strength, parsers)~~ done at `b36f2f0`                                                                                                                                | ~~High~~     | ~~M~~      | ~~Feature~~       |
| ~~8~~  | ~~Re-export the type from `server/` via aliases; keep deprecated-shim parity tests green~~ done at `b36f2f0`, `acd97a4`                                                                                                          | ~~High~~     | ~~M~~      | ~~Feature~~       |
| ~~9~~  | ~~Adopt the typed `ETag` in client validator handling (§4.3.4 filtering)~~ done at `acd97a4`                                                                                                                                     | ~~High~~     | ~~M~~      | ~~Feature~~       |
| ~~10~~ | ~~Evolve `cacheEntry` → `storedResponse` with validator-matching methods~~ done at `07fe65c` (`storedValidator.weaklyMatches`; parsed once at store time)                                                                        | ~~Medium~~   | ~~M~~      | ~~Feature~~       |
| 11 | Decide OQ2: accept Alex's captures as permanent `client/testdata/` fixtures?                                                                                                                                                     | Medium   | S      | Decision      |
| 12 | Draft the Alex reply email (Age answer + no-store/freshening changes shipped)                                                                                                                                                    | Medium   | S      | Documentation |
| 13 | If fixtures accepted: commit captures with a provenance README                                                                                                                                                                   | Medium   | S      | Feature       |
| 14 | Decide OQ3: add a tag-triggered Release workflow or document manual-only as policy                                                                                                                                               | Medium   | S      | Decision      |
| 15 | If workflow: create it, trigger on `v*` after CI green, test on the next tag                                                                                                                                                     | Medium   | S      | Feature       |
| ~~16~~ | ~~Correct ROADMAP OQ3 wording (all 5 tags have pages; the gap is automation, not pages)~~ done 2026-09-18 by the 19:49 session (ROADMAP.md OQ3 "verified 2026-09-18 via `gh release list`")                                      | ~~Medium~~   | ~~S~~      | ~~Documentation~~ |
| 17 | Decide OQ5: promote the ~25 open-low archive items or let them die                                                                                                                                                               | Low      | S      | Decision      |
| 18 | Decide OQ4: affirm the FNV-64a `Strong` default as a standing decision                                                                                                                                                           | Low      | S      | Decision      |
| 19 | Design opt-in freshness-based serving (§4.2 `max-age`/`Expires`) — default stays accelerator                                                                                                                                     | High     | L      | Feature       |
| 20 | Implement §4.2 freshness serving behind the opt-in                                                                                                                                                                               | High     | L      | Feature       |
| 21 | Add §4.2 spec tests + `docs/rfc9111-conformance.md` updates                                                                                                                                                                      | High     | M      | Quality       |
| 22 | Implement `stale-while-revalidate` on top of §4.2                                                                                                                                                                                | Medium   | M      | Feature       |
| 23 | Implement `stale-if-error` on top of §4.2                                                                                                                                                                                        | Medium   | M      | Feature       |
| 24 | Add `Last-Modified`/`If-Modified-Since` as a second validator type                                                                                                                                                               | Medium   | L      | Feature       |
| 25 | Add client hooks `OnHit`/`OnStore`/`OnFreshen`/`OnInvalidate` (mirror server)                                                                                                                                                    | Medium   | M      | Feature       |
| 26 | Per-host key partitioning by default (cross-origin KeyFunc safety)                                                                                                                                                               | Medium   | M      | Feature       |
| 27 | Opt-in disk persistence of the client cache                                                                                                                                                                                      | Low      | L      | Feature       |
| 28 | Singleflight request coalescing for concurrent identical GETs                                                                                                                                                                    | Medium   | M      | Feature       |
| 29 | Project website launch (sibling Astro/Starlight pattern)                                                                                                                                                                         | Medium   | L      | Documentation |
| 30 | Demo video for the website landing page                                                                                                                                                                                          | Low      | L      | Documentation |
| 31 | Comparison table vs other Go ETag/caching libraries                                                                                                                                                                              | Medium   | M      | Documentation |
| 32 | awesome-go listing submission                                                                                                                                                                                                    | Low      | S      | Documentation |
| 33 | OTEL/Prometheus wiring recipe on the existing hooks                                                                                                                                                                              | Medium   | S      | Documentation |
| 34 | `go-etag/otel` sub-module — stays parked until hooks battle-tested AND consumer asks                                                                                                                                             | Low      | L      | Feature       |
| 35 | Root shim deletion — executes at v1.0.0 only                                                                                                                                                                                     | Low      | S      | Cleanup       |
| 36 | Define v1.0.0 criteria (what besides shim deletion gates the major bump)                                                                                                                                                         | Medium   | S      | Documentation |
| ~~37~~ | ~~Cut the next release carrying `[Unreleased]` (typed `Code` surface + sentinel interface change)~~ done — v0.4.0 tagged `16369cc` 2026-09-18                                                                                    | ~~High~~     | ~~M~~      | ~~Documentation~~ |
| ~~38~~ | ~~Run release gates for that cut: proxy.golang.org, sum.golang.org, clean-room `go get`~~ done (report 2026-09-18_21-00)                                                                                                         | ~~High~~     | ~~S~~      | ~~Quality~~       |
| ~~39~~ | ~~Consumer migration note for the `ErrInvalidConfig` `error`-interface change~~ done — README "Upgrading from v0.3.x" (landed with the release sync)                                                                             | ~~Medium~~   | ~~S~~      | ~~Documentation~~ |
| ~~40~~ | ~~Push cqrs-htmx's 7 local alignment commits; watch its CI green~~ done — verified pushed 2026-09-18 (report 2026-09-18_19-49 §a)                                                                                                | ~~High~~     | ~~S~~      | ~~Feature~~       |
| ~~41~~ | ~~Fix cqrs-htmx pre-existing `GOWORK=off` failures (systemadapter, system-demo, integration_test)~~ verified passing 2026-09-18 (report 2026-09-18_19-49 §a)                                                                     | ~~Low~~      | ~~M~~      | ~~Bug~~           |
| ~~42~~ | ~~HARVEST this report's (f) into `TODO_LIST.md` / `ROADMAP.md`~~ done 2026-09-18 by the 19-49 session                                                                                                                            | ~~High~~     | ~~S~~      | ~~Documentation~~ |
| 43 | Record the five OQ decisions in ROADMAP once made (annotate, don't rewrite)                                                                                                                                                      | Medium   | S      | Documentation |
| ~~44~~ | ~~Sweep the aged 2026-09-11 "Done this cycle" section out of TODO_LIST~~ done 2026-09-18 (report 2026-09-18_19-49 §a)                                                                                                            | ~~Low~~      | ~~S~~      | ~~Documentation~~ |
| ~~45~~ | ~~Update AGENTS.md Go-version and go-error-family mentions after item 1-3 settle~~ done 2026-09-18 ("Go 1.27+"; refreshed again 2026-09-23 with the `go 1.27` directive history)                                                 | ~~Medium~~   | ~~S~~      | ~~Documentation~~ |
| ~~46~~ | ~~Capture `-benchmem -count=6` baselines under `reports/bench/` before any perf work~~ done 2026-09-18: `reports/bench/2026-09-18_baseline-typed-code-stored-validator.txt`                                                      | ~~Low~~      | ~~S~~      | ~~Quality~~       |
| 47 | Run `erraudit --no-suppress` on the next error-surface change                                                                                                                                                                    | Low      | S      | Quality       |
| 48 | Design a build-gate so go.mod commits cannot land red (hook or sweep rule)                                                                                                                                                       | Medium   | M      | Quality       |
| ~~49~~ | ~~docs-health VERIFY pass on remaining ROADMAP claims (OQ3 done via item 16)~~ done 2026-09-23 — OQ1 resolution verified, Theme 2 marked complete, OQ8 added                                                                     | ~~Low~~      | ~~S~~      | ~~Documentation~~ |
| 50 | Extend client fuzz coverage beyond `FuzzHasNoStoreDirective` (Cache-Control variants)                                                                                                                                            | Medium   | M      | Quality       |

## g) Questions I cannot answer myself

1. **Did you (or another session) intentionally bump to Go 1.27.1 + go-error-family
   v0.10.1 at 19:05 today?** I verified the commit content and its blast radius but
   cannot know the intent. Direction matters: adopt 1.27.1 (upgrade local + CI pins) vs
   revert the toolchain line and keep only the dep bump if it builds on 1.26.7.
2. **Which of the five ROADMAP open-question recommendations do you approve?**
   Especially OQ1 (shared subpackage vs client→server import) — the only one blocking
   code work (Theme 2). I gave picks; the public-surface call is yours.
3. **Is the Alex field-report thread still live?** The repo contains no context beyond
   OQ2 — I cannot know whether he still expects the reply or whether the captures offer
   has an expiry.

---

_Point-in-time snapshot; goes stale. Section (f) is HARVEST input for `TODO_LIST.md` /
`ROADMAP.md`. Note: this report is `.md` per explicit user instruction, overriding the
status-report skill's HTML-canonical default._

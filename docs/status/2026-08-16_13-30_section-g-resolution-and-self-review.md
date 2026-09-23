# §g Resolution, Release Artifact & Self-Review — Status Report

_Date: 2026-08-16 13:30 CEST. Format: Markdown per explicit user instruction (status-report skill default is HTML; brutal-self-review questions folded into this single report per "write a full status report")._
_Scope: strictly this session — resuming at the 13-12 report's §g waiting state, re-verifying, obtaining answers, executing them. Predecessors: `2026-08-16_13-12_post-release-followup-ci-repair.md` (referenced as `13-12 §x`) and `2026-08-16_12-28_v0.2.0-release-and-ecosystem-sweep.md` (`12-28 §x`)._

## a. FULLY DONE (verified this session)

| Item                                                            | Evidence                                                                                                                                                                                                                                                                                                                                                                     |
| --------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **State re-verified before acting** (reports are point-in-time) | go-etag tree clean with daemon commit `962ab3e` detected on top of `6352497`; httputil confirmed release-less at v0.12.0 with v0.11.0 still Latest; kit CI green at `6b0b6be` with the vulnerable `go-version: "1.26.x"` govulncheck step still present (ci.yml:87-92); skill gate location found; kit discovered LIVE on pkg.go.dev. No stale-truth assumptions carried in. |
| **All three §g questions asked in ONE structured prompt**       | Batched yes/no batch 2; three unambiguous answers (yes / yes / no) in a single interruption. Pattern now 2-for-2 across sessions.                                                                                                                                                                                                                                            |
| **g.1 — httputil v0.12.0 GitHub Release created**               | `gh release create v0.12.0 --generate-notes --notes-start-tag v0.11.0 --latest`; published 13:24:12 CEST, verified Latest via `gh release list` (v0.11.0 demoted). Proxy/module/CI were already green — the release page was the last missing artifact.                                                                                                                      |
| **g.1 — Release body curated to repo convention**               | Auto-notes were compare-link-only (direct-push history → no PRs). Checked v0.11.0's convention first (What's New + full CHANGELOG sections + compare link), wrote curated body from CHANGELOG 0.12.0, applied via `gh release edit --notes-file`; verified 43 lines via API.                                                                                                 |
| **g.2 — Pre-tag CI gate added to go-release skill**             | New §4.4 "CI is green on the exact commit being tagged" between Phase 4.3 and Phase 5: hard `gh run list` gate, the tag-immutability rationale, and the GOTOOLCHAIN manifest-lag pin pattern (`setup-go` lags go.dev by hours; `env: GOTOOLCHAIN: goX.Y.Z` is authoritative). Placement verified in-context after insert.                                                    |
| **g.3 — Kit toolchain pin declined, decision recorded**         | No edit made to kit; deferral noted in the report addendum (f.9 stays open, owner's call until next touch).                                                                                                                                                                                                                                                                  |
| **f.4 — kit on pkg.go.dev resolved itself**                     | Landing page renders; the "not in the latest version / Go to latest" banner proves pkgsite has indexed v0.2.0. Root cause was pure crawl latency (~3h), NOT the capital-`LarsArtmann` module path. (Caveat: the @v0.2.0 page itself was not directly fetched — see b/d.)                                                                                                     |
| **13-12 report closed out with §h RESOLUTION addendum**         | Four numbered resolutions (g.1 done, g.2 done, g.3 deferred, f.4 self-resolved); committed `dc7ca86`, 1 file, 9 insertions, attribution trailers.                                                                                                                                                                                                                            |
| **Skills loaded before acting**                                 | `status-report/SKILL.md` + `brutal-self-review/SKILL.md` viewed in full before writing this report.                                                                                                                                                                                                                                                                          |

## b. PARTIALLY DONE

| Item                                                 | State                                                                                                                                                                                                                                                                                            | Blocker                                                                                                                                    | Effort |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| ~~**go-release skill gate: prose in, tooling out**~~ | ~~§4.4 documents the CI gate and GOTOOLCHAIN pin, but the skill's own `scripts/pre-release-check.sh` ("Use the helper script to automate the gate checks", §4 intro) was NOT extended — the script still cannot fail on a red/in-progress CI run.~~                                                  | ~~**NOT-DO for this repo** — the script lives in the owner's personal skill, not go-etag; the go-release §4.4 prose gate is the encoded fix.~~ | ~~S~~      |
| ~~**kit@v0.2.0 pkg.go.dev render**~~                 | ~~Landing page + "Go to latest" banner verified (indexing proven), but `pkg.go.dev/...go-github-kit@v0.2.0` was never fetched directly, so the v0.2.0 docs page itself is inferred, not seen. My closing message said "live on pkg.go.dev" — true but one step short of the claim's full strength.~~ | ~~done — the @v0.2.0 page renders fully (verified 2026-09-10 docs-health pass).~~                                                              | ~~XS~~     |
| ~~**go-etag master ahead 1**~~                       | ~~`dc7ca86` (report addendum) committed locally, unpushed. Push requires explicit GO per standing rule.~~                                                                                                                                                                                            | ~~done — pushed; master long past `dc7ca86`.~~                                                                                                 | ~~XS~~     |
| ~~**TODO harvest (3 reports deep)**~~                | ~~13-12 §f (16 items) + this report's §f have never been harvested into `TODO_LIST.md`/`ROADMAP.md`; the status reports are acting as the de-facto task store — exactly the entombment the status-report skill warns about.~~                                                                        | ~~done (2026-09-10 docs-health pass — TODO_LIST + ROADMAP created, all reports harvested).~~                                                   | ~~M~~      |

## c. NOT STARTED

| Item                                                                                                                                                                                                                                                                                  | Why it's still open                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Carry-over f-list from 13-12 §f (items 5-16): cqrs-htmx indirect refs, DiscordSync flake check, library-policy devShell, nsfw tailwind kill, kit + go-etag GOTOOLCHAIN pins, docs-health pass, benchmark archive, superseded-pointer notes, ecosystem search, showcase runtime-verify | Untouched this session by design — this session was scoped to §g resolution only.                    |
| Server_timing sub-tag release-page convention check                                                                                                                                                                                                                                   | Noticed while handling the v0.12.0 release; not investigated (would be one `gh release list` sweep). |

## d. TOTALLY FUCKED UP (honest accounting)

| Item                                                         | What happened                                                                                                                                                                                                                                                                        | Cost / lesson                                                                                                                                                         |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Guessed a nonexistent `gh` JSON field**                    | Verified the new release with `--json ...,isLatest` — field doesn't exist; error + wasted call.                                                                                                                                                                                      | `gh release list` shows Latest plainly. Check available fields (the error helpfully listed them) or use the simple command first.                                     |
| **Three wasted round-trips on one edit**                     | Report edit failed twice: once on the freshness guard (daemon re-touched the file at 13:15 — correct guard behavior), then twice more because my `view` offsets (130, 88) overshot a file I assumed was longer than its ~75 post-reformat lines.                                     | After detecting a daemon touch, re-read from offset 0 (or `wc -l`) immediately before editing; never edit from remembered file geometry.                              |
| **Created the release before checking the notes convention** | `gh release create --generate-notes` first, noticed the PR-less body after, then checked v0.11.0 and curated. A substandard compare-link-only body was live for ~3 minutes. The skill's own Phase 2.4 ("curate release notes") comes before Phase 7 — I ran the phases out of order. | Curate-then-create: write notes from the CHANGELOG first, pass `--notes-file` at create time. One command instead of two, no ugly window.                             |
| **Overclaimed f.4 in my closing line**                       | "kit is live on pkg.go.dev" was stated as flatly resolved; what I had was the v0.1.0 landing page with a latest-version banner (strong evidence of indexing, but not the v0.2.0 page itself). The report addendum was precise; the chat summary was looser than the evidence.        | Say exactly what was observed ("landing + banner verified; @v0.2.0 page not fetched"). Precision in chat matters as much as in reports — the user acts on chat lines. |
| **Left /tmp scratch behind again**                           | `/tmp/httputil-release-notes-v0.12.0.md` still exists after a session whose predecessor explicitly had a "clean up /tmp scratch" todo. Harmless (tmpfs) but the recurrence is the point.                                                                                             | Delete or relocate to a source-controlled location in the same breath as the action that created it.                                                                  |

## e. WHAT WE SHOULD IMPROVE!

- **Close the prose/script split brain in go-release**: §4.4 says "check CI"; `pre-release-check.sh` automates every other gate. A gate that isn't enforced by the tool everyone runs is a gate that gets skipped under time pressure — which is precisely how today's three red tags happened.
- **Curate-then-create releases**: derive the GitHub Release body from the CHANGELOG up front (Phase 2.4 → Phase 7 order). Auto-notes are a fallback for PR-ful histories, not a default for direct-push repos.
- **Re-read immediately before every edit in daemon-shared repos**: the freshness guard saved me from a blind overwrite once; the follow-up offset misses showed I still half-ignored the lesson.
- **Harvest §f lists**: three consecutive reports now carry un-harvested next-lists. Either TODO_LIST.md is the task store (harvest every report) or the reports are (say so once) — the current in-between is a slow split brain.
- **Precise claims in closing messages**: match the chat summary's strength to the weakest verified link in the chain, not the strongest.
- **Keep the one-structured-prompt pattern for gated batches**: two sessions, five questions, zero back-and-forth. This is now the default for any GO-needing batch.

## f. NEXT (prioritized)

1. ~~**GO: push `dc7ca86`** — go-etag master is ahead 1 (report addendum).~~ done — pushed (master has advanced far past it)
2. **Extend `pre-release-check.sh` in go-release skill** to fail on red/in-progress CI (`gh run list` check), matching §4.4 — closes b.1/d's split brain. Needs GO (edit to your personal skill's tested surface).
   _**Status (2026-09-10):** out of repo scope — owner's personal skill; prose gate §4.4 is the encoded fix._
3. ~~**Fetch `pkg.go.dev/github.com/LarsArtmann/go-github-kit@v0.2.0` directly** — close the b.2 inference gap (XS).~~ done (2026-09-10 docs-health pass — page renders fully)
4. ~~**GO: docs-health HARVEST** — pull 13-12 §f + this §f into `TODO_LIST.md`/`ROADMAP.md` before a fourth report entombs more items.~~ done (2026-09-10 — both files created and populated)
5. Kit GOTOOLCHAIN pin at next touch (deferred by g.3; 13-12 f.9 unchanged).
   _**Status (2026-09-10):** foreign repo — kit owner's call._
6. go-etag GOTOOLCHAIN pin / optional govulncheck job (13-12 f.10).
   _**Status (2026-09-10):** open — TODO_LIST.md #11. **Done at `1bc839f`** — both landed in v0.3.0; residual scoping is TODO_LIST item 4._
7. cqrs-htmx: tidy ~20 submodules' indirect go-etag refs after the foreign session lands (13-12 f.5).
   _**Status (2026-09-10):** foreign repo._
8. DiscordSync: `nix flake check` + foreign `stdenv.hostPlatform` hunk decision (13-12 f.6, owner).
   _**Status (2026-09-10):** foreign repo._
9. library-policy: add go-licenses + vulnix to devShell (13-12 f.7).
   _**Status (2026-09-10):** foreign repo._
10. nsfw-classifier: tailwind-build `signal: killed` (13-12 f.8).
    _**Status (2026-09-10):** foreign repo._
11. ~~go-etag docs-health pass: ROADMAP/TODO_LIST/FEATURES/README post-v0.2.0 (13-12 f.11 — subsumes f.4 above if done together).~~ done (2026-09-10 — TODO_LIST, ROADMAP, FEATURES, docs/DOMAIN_LANGUAGE built; README badge + AGENTS.md fixed)
12. ~~Benchmark archive v0.1.1→v0.2.0 into `reports/` (13-12 f.12).~~ **Won't implement — no `reports/` directory exists; benchmarks live in README** (ROADMAP non-goal)
13. ~~Superseded-pointer annotations: 09-42/11-33 → 12-28 → 13-12 → this report (13-12 f.13; this report now also needs a pointer).~~ done (2026-09-10 docs-health pass — every 2026-0* report annotated inline; this file included)
14. Ecosystem completeness search for public go-etag consumers beyond the known six (13-12 f.14).
    _**Status (2026-09-10):** open — TODO_LIST.md #21. **Done at `12dd823`** — GitHub code search: no external consumers (v0.3.0 CHANGELOG)._
15. cqrs-htmx middleware-showcase runtime-verify (13-12 f.15).
    _**Status (2026-09-10):** foreign repo._
16. Check whether `server_timing/v0.10.0`/`v0.9.1` ever had release pages; decide the sub-module release convention once (noticed this session).
    _**Status (2026-09-10):** foreign repo (httputil)._
17. ~~Delete `/tmp/httputil-release-notes-v0.12.0.md` (or decide release-note sources belong in-repo).~~ resolved (moot — tmpfs state from 2026-08-16 no longer exists)
18. ~~Re-run `golangci-lint run` on go-etag once to re-confirm the 26 IDE warnings are still stale false reads (not re-confirmed THIS session; prior sessions' verification inherited).~~ done (2026-09-10 — lint run clean, 0 issues)
19. Consider a Release workflow guard for go-etag (it currently has no release workflow at all — tags are release-page-less by default; decide if that's intended).
    _**Status (2026-09-10):** open — ROADMAP.md Open Questions #3._
20. ~~Daemon interplay: note in AGENTS.md that status reports get auto-reformatted within ~minutes of commit, so follow-up edits must always re-read (operational knowledge, currently only in session history).~~ done (2026-09-10 — AGENTS.md "Repo Workflow Notes" section)

## g. QUESTIONS (cannot resolve myself)

1. **Push `dc7ca86` now?** go-etag master is ahead 1 (the 13-12 §h resolution addendum). Standing rule is push-only-on-request; this is the request slot.
   _**Resolved:** pushed — master history long since advanced past `dc7ca86`._
2. **Extend the go-release skill's `scripts/pre-release-check.sh`** to enforce the new CI gate (a code change to your personal skill's automation), or is the §4.4 prose gate enough? I can do the script edit + self-test in one step, but it's your tested tooling.
   _**Status (2026-09-10):** owner's personal skill — out of repo scope; §4.4 prose gate stands._
3. **Task store decision**: run docs-health HARVEST now so `TODO_LIST.md`/`ROADMAP.md` become the single source of truth for next-work (reports become pure snapshots), or keep the reports' §f lists as the working task store and skip harvesting?
   _**Resolved (2026-09-10):** HARVEST ran — `TODO_LIST.md`/`ROADMAP.md` are the task store; reports are pure snapshots (recorded in AGENTS.md Repo Workflow Notes)._

## h. SELF-REVIEW DIGEST (brutal-self-review skill, verbatim answers)

- **What did you forget?** The helper script behind the skill's gate section; the /tmp file; the @v0.2.0 direct fetch; harvesting §f.
- **Something stupid we do anyway?** Publishing reports whose next-steps never reach a task store, then writing the next report about the same items.
- **What could you have done better?** Curate-then-create the release; read before edit in daemon-shared files; verify with the simple command (`gh release list`) instead of guessing API fields.
- **What can you still improve?** Items f.2-f.4, f.18 above.
- **Did you lie to you?** No fabrication found — one strength-mismatched claim (f.4 "live on pkg.go.dev") called out in d and corrected in wording here.
- **How to be less stupid?** Automate the gates people (and agents) skip; match claim strength to evidence.
- **Ghost systems?** `TODO_LIST.md` is drifting toward ghost status (exists, unharvested, reports bypass it). No code ghost systems found — no code was changed this session.
- **Scope creep?** No — this session held strictly to §g resolution; the only addition (pkg.go.dev re-check) was a 1-command verification of a predecessor open item.
- **Removed something useful?** No.
- **Split brains?** Two small ones: skill §4.4 prose vs `pre-release-check.sh`; reports-§f vs `TODO_LIST.md`. Both listed with owners (g.2/g.3).
- **Tests?** N/A — no production code changed (docs, config-adjacent skill, release metadata). The 26 stale IDE warnings were not re-linted this session (f.18).

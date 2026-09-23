# Status Report — 2026-09-22 21:36 — art-dupl Acceptance Judgment & Documentation

Session scope: (1) judging the single clone group reported by
`art-dupl --sort total-tokens -t 1 --type-aware`, (2) documenting the acceptance
decision well after the user's "Document it WELL!", plus one live-discovered
toolchain gotcha. No production code changed; this is a judgment + documentation
session. Everything below is grounded in what this session actually did or
directly observed — no new research.

---

## a) FULLY DONE

1. **Clone group judged: ACCEPTED, not extracted.**
   `art-dupl -t 1 --type-aware` reports exactly one group: `etagWriter.Write`'s
   body-buffer append (`server/etag.go`) vs `nonHijackableRecorder.Write`
   (`server/testutil_test.go`). Verdict with three-part rationale: the test
   double must NOT implement `http.Hijacker` (its entire purpose), so it cannot
   embed `httptest.ResponseRecorder` or share a production buffer type;
   production code cannot import test helpers; the shared logic is one `append`
   plus `return len(b), nil` — an abstraction would take more parameters than
   the duplicated code has lines. Evidence: art-dupl re-run after edits still
   reports exactly this 1 group.
2. **Full acceptance rationale on the code itself.**
   `nonHijackableRecorder`'s doc comment (`server/testutil_test.go:89-96`)
   expanded from 2 to 6 rationale lines: why the double cannot share code, why
   production cannot import the test, cost argument, and an explicit pointer
   that art-dupl reports this as the repo's single accepted clone group.
3. **AGENTS.md now carries the decision (session-discovery layer).**
   New Non-Obvious Behaviors bullet: "The one `art-dupl` clone group is
   accepted — do not extract it" — with the full reasoning, so future sessions
   running a dedup pass see it before re-litigating. Previously the rationale
   existed only inside a point-in-time status snapshot
   (`docs/status/2026-09-11_06-18_...md`), which no session loads by default.
4. **GOTOOLCHAIN gotcha refined in AGENTS.md (live discovery).**
   Bare `golangci-lint run` fails here ("go.mod requires go >= 1.27.1;
   running go 1.26.7") because golangci-lint shells out to go. The Commands
   prose now says every golangci-lint invocation needs the
   `GOTOOLCHAIN=auto` prefix, not just go commands.
5. **Full verification gate green.**
   - `golangci-lint run` (repo-wide, GOTOOLCHAIN=auto): **0 issues**
   - `golangci-lint fmt`: clean (no reflow of new comments)
   - `go test -race ./...`: all 4 packages ok
   - `go vet ./...`: clean
   - `art-dupl -t 1 --type-aware`: still exactly the 1 accepted group
     (lines moved 108 → 112-114 from the longer comment, as expected)
   - Note: the nolint_filter warning ("unknown linters in //nolint directives:
     erraudit") is expected — those directives are audited by erraudit itself,
     not golangci, per AGENTS.md.
6. **All edits committed by the auto-daemon.**
   - `0c2201e` — AGENTS.md Non-Obvious Behaviors bullet (+1) and
     `server/testutil_test.go` doc comment (+6/-2)
   - `969d077` — AGENTS.md GOTOOLCHAIN wording (1 line); also swept two
     unrelated files (see d.2)
   - No manual commit performed: Crush forbids commits without explicit user
     request; the daemon picked everything up.

## b) PARTIALLY DONE

1. **Acceptance is documented but not machine-enforced.**
   Works: prose rationale in three places (code comment, AGENTS.md, this
   report); art-dupl re-run confirms the claim is accurate today.
   Open: nothing fails if a second clone group appears tomorrow — the
   code comment's "single accepted clone group" claim would silently rot into
   a false statement, and AGENTS.md would under-count. No art-dupl baseline
   file / CI gate exists (a decision deferred since at least 2026-08-07).
   Blocker: none — a tooling decision is needed (see g.1). Effort: S–M.
2. **Verification gate was scoped first, closed late.**
   Works: by report time the full documented gate is green (a.5).
   Open: the first pass after editing ran only `golangci-lint run
   ./server/...` + `go test ./server/` — it skipped fmt, -race, vet, and
   repo-wide lint, and the bare golangci-lint call failed once on the
   toolchain mismatch. Everything was closed during report writing, but the
   documented bar should have been the first pass, not the last.
   Blocker: none (habit, not tooling). Effort to prevent recurrence: S.

## c) NOT STARTED

1. **art-dupl baseline + optional CI enforcement** — glimpsed as open item #16
   in `docs/status/2026-08-07_09-01_deduplication-pass.md` ("never adopted");
   this session made the need concrete (b.1) but did not start it. Still
   wanted: yes, pending the policy decision in g.1.
2. **Non-type-aware `return tags` clone** (`ParseETagList` vs `splitRawETags`,
   entity_tag.go) — visible only without `--type-aware`; recorded open in the
   2026-08-07 report; out of this session's type-aware scope. Priority: low.
3. **HEAD-freshening follow-ups from the 2026-09-11 06:18 report** — behavior
   pin test for marker-stripping on HEAD-freshened entries, perf check for the
   extra `header.Clone()` per freshening HEAD, `cloneHeader` nil-branch
   prove-or-delete, coverage check on `persistFreshened`/`cloneHeader`.
   Noticed while reading that report for consistency; untouched (out of scope
   per the session instruction).
4. **`/tmp` verification-pattern retirement** (v0.4.0 release report table,
   item #50) — noticed in grep output; not touched.

## d) TOTALLY FUCKED UP

1. **Nothing is broken by this session.** No production code changed; every
   gate is green (a.5). The worst event was a one-command self-inflicted
   stumble: bare `golangci-lint run` failed on the toolchain mismatch before
   being rerun with `GOTOOLCHAIN=auto` — classified and fixed under
   e.1/e.2, not d.
2. **Uninvestigated foreign change, flagged not touched:** daemon commit
   `969d077` contains a 1-line `go.mod` change and a 154-line reflow of
   `docs/status/2026-09-18_21-00_v040-release-session.md` that this session
   did not author. Per instructions I did not research them; the daemon
   re-formats shared files continuously (documented behavior), but the
   `go.mod` line is worth a 30-second look by someone. Severity: none known;
   full gate green afterward either way.

## e) WHAT WE SHOULD IMPROVE

1. **Read tool preconditions before the first invocation, not after the first
   failure.** AGENTS.md said "prefix every go command with GOTOOLCHAIN=auto";
   I read it and still ran bare `golangci-lint run` once. golangci-lint shells
   out to go — the prefix applies. Impact: one wasted round trip per session
   until internalized.
2. **Run the full documented gate on the first pass, even for comment-only
   edits.** fmt + repo-wide lint + `go test -race ./...` + vet is minutes of
   machine time; staging it saved nothing and cost a late correction loop
   (b.2). A comment can trip godot/wsl just like code.
3. **Make tool-state acceptance machine-checkable, not prose-trust.** "We
   decided this clone is fine" decays the moment the detector's output
   changes. A committed art-dupl baseline (or a CI step asserting "exactly 1
   known group") turns the decision into an enforced invariant. Pairs with
   c.1; needs the policy call in g.1.
4. **Date-stamp claims about tool output inside code comments.** "art-dupl
   reports this as the repo's single accepted clone group" is true today and
   unverifiable tomorrow. Phrasing like "accepted 2026-09-22; see AGENTS.md"
   survives new clones without becoming a lie (or is obviated by e.3).
5. **Triple-redundancy of the rationale is deliberate but has drift risk.**
   Code comment (code readers) + AGENTS.md (sessions) + snapshots (history)
   intentionally overlap; AGENTS.md must remain the single authority. If the
   decision ever changes, all three must move together — add that to the
   docs-health VERIFY mental checklist.

## f) NEXT TASKS (up to 50, ranked; brainstorm per user request — most are ROADMAP fuel, HARVEST should apply routing rigor)

| #  | Task                                                                                                                                                                                                                                                     | Impact | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Decide enforcement policy for the accepted clone (see g.1), then implement it (baseline file or CI assert "exactly 1 group")                                                                                                                             | High   | M      | Quality       |
| ~~2~~  | ~~HARVEST this report's list into TODO_LIST.md / ROADMAP.md (docs-health HARVEST) so it does not die in this timestamped file~~ done 2026-09-23 docs-health pass (TODO_LIST #1–#7; ROADMAP OQ8)                                                          | ~~High~~   | ~~S~~      | ~~Documentation~~ |
| ~~3~~  | ~~Look at the 1-line `go.mod` change inside daemon commit `969d077` (not authored this session)~~ done 2026-09-23 — it relaxed `go 1.27.1` → `go 1.27`; builds green (go-error-family needs only 1.26); floor decision for v0.5.0 routed to TODO_LIST #2 | ~~Medium~~ | ~~S~~      | ~~Cleanup~~       |
| ~~4~~  | ~~Date-stamp the testutil comment's "single accepted clone group" claim (e.4)~~ done 2026-09-23 — comment now reads "accepted on 2026-09-22; see AGENTS.md"                                                                                              | ~~Low~~    | ~~S~~      | ~~Documentation~~ |
| ~~5~~  | ~~Add `GOTOOLCHAIN=auto` prefix into the AGENTS.md command block's golangci-lint lines (prose note exists; block still shows bare commands)~~ done 2026-09-23 — every command-block line now carries the prefix + a verification-gate paragraph          | ~~Low~~    | ~~S~~      | ~~Documentation~~ |
| ~~6~~  | ~~Pin HEAD-freshening marker-stripping behavior with a spec test (carried from 09-11 06:18 report, partial item 1)~~ still open — TODO_LIST #5 (2026-09-23)                                                                                              | ~~Medium~~ | ~~M~~      | ~~Quality~~       |
| 7  | Benchmark the HEAD freshening path (extra `header.Clone()`) per the reports/bench discipline                                                                                                                                                             | Low    | M      | Quality       |
| 8  | Prove-or-delete `cloneHeader`'s nil branch (unit test or drop the guard)                                                                                                                                                                                 | Medium | S      | Quality       |
| 9  | Coverage check on `persistFreshened` / `cloneHeader` lines (`go test -coverprofile`)                                                                                                                                                                     | Medium | S      | Quality       |
| 10 | Verify `art-dupl` supports a baseline/check subcommand before building item 1 on it (verify-external-claims)                                                                                                                                             | Medium | S      | Quality       |
| 11 | Decide the art-dupl CI question once and for all (threshold, baseline, or explicit Won't) — deferred since 2026-08-07                                                                                                                                    | Medium | S      | Process       |
| ~~12~~ | ~~Check CI's golangci-lint job pins go 1.27.1 / GOTOOLCHAIN correctly (not inspected this session)~~ verified 2026-09-23 — all three gated jobs pin `GOTOOLCHAIN: go1.27.1`; govulncheck deliberately unpinned (`.github/workflows/ci.yml:24,58,76`)     | ~~Medium~~ | ~~S~~      | ~~Quality~~       |
| 13 | Evaluate fixing the LSP experience (gopls/golangci-lint-ls error on every file) via a devShell GOTOOLCHAIN override                                                                                                                                      | Medium | M      | Quality       |
| ~~14~~ | ~~Re-run `erraudit nolint-audit .` to re-confirm the two `drainAndClose` directives are still NEEDED (last verified 09-11)~~ done 2026-09-23 — 2 needed, 0 stale (docs-health pass)                                                                      | ~~Low~~    | ~~S~~      | ~~Quality~~       |
| 15 | Run erraudit `--no-suppress` audit pass and re-triage surfaced findings (last done 09-11)                                                                                                                                                                | Low    | S      | Quality       |
| 16 | Run art-dupl at default threshold 5 as a regression sanity alongside -t 1                                                                                                                                                                                | Low    | S      | Quality       |
| 17 | Decide non-type-aware `return tags` clone (ParseETagList vs splitRawETags): accept or refactor                                                                                                                                                           | Low    | S      | Quality       |
| ~~18~~ | ~~Run docs-health VERIFY on `docs/rfc9111-conformance.md` against `client/spec_test.go` claims~~ partially superseded — the 2026-09-23 full audit re-read the doc suite-wide; a dedicated row-by-row re-audit against spec_test.go remains open (low)    | ~~Medium~~ | ~~M~~      | ~~Documentation~~ |
| ~~19~~ | ~~docs-health VERIFY pass over AGENTS.md claims touched by recent refactors (error tables, gotchas)~~ done 2026-09-23 docs-health pass — Commands/Architecture/Errors re-verified against code; go-directive history corrected                           | ~~Medium~~ | ~~M~~      | ~~Documentation~~ |
| 20 | Confirm every `Example*` function in the repo has a `// Output:` directive (testableexamples linter requirement)                                                                                                                                         | Low    | S      | Quality       |
| 21 | Verify a negative spec test exists: `FreshenOn304` does NOT apply to HEAD updates (AGENTS.md claim) — add if missing                                                                                                                                     | Medium | S      | Quality       |
| 22 | Verify spec coverage: Age never runs backwards on rebuilds — add pin if missing                                                                                                                                                                          | Low    | S      | Quality       |
| 23 | Verify spec coverage: rebuilt responses carry `Uncompressed` for decoded stored bodies — add pin if missing                                                                                                                                              | Low    | S      | Quality       |
| 24 | Verify spec coverage: quote-aware no-store parsing (`"no-store"` argument does not match) — fuzz target exists; check spec pin too                                                                                                                       | Low    | S      | Quality       |
| 25 | Verify default KeyFunc (= URL string) has a spec pin; KeyFunc credential warning lives only in client/doc.go                                                                                                                                             | Low    | S      | Quality       |
| 26 | Verify Stats() (Hits/Stored/Entries) counters have direct spec assertions, not just incidental coverage                                                                                                                                                  | Low    | S      | Quality       |
| 27 | Consider an integration test for oversized-body streaming (buffered prefix chained to live remainder over a real server)                                                                                                                                 | Low    | M      | Quality       |
| 28 | Consider pinning 304 passthrough when the caller set their own If-None-Match over the real-wire server suite                                                                                                                                             | Low    | S      | Quality       |
| 29 | Cross-repo: keep go-etag error-template wording byte-identical with httputil's registration (AGENTS.md warns of drift); schedule a parity check                                                                                                          | Medium | M      | Quality       |
| 30 | Consider an ADR in docs/decisions/ for the accepted-clone policy (discoverability beyond AGENTS.md)                                                                                                                                                      | Low    | S      | Documentation |
| 31 | Annotate the 09-11 06:18 snapshot's stale line cite (testutil_test.go:57 → now 112-114) per snapshot-annotation policy                                                                                                                                   | Low    | S      | Documentation |
| 32 | Verify the daemon's 154-line reflow of `docs/status/2026-09-18_21-00_v040-release-session.md` (commit `969d077`) did not corrupt tables                                                                                                                  | Low    | S      | Quality       |
| 33 | Confirm no CHANGELOG entry is needed for docs-only sessions (policy check)                                                                                                                                                                               | Low    | S      | Process       |
| ~~34~~ | ~~Verify docs/DOMAIN_LANGUAGE.md exists / is current for this repo (global project-discovery checklist)~~ done 2026-09-23 — verified current; stale entitytag/cacheEntry pointers fixed, `Code`/`Domain`/`storedResponse` terms added                    | ~~Low~~    | ~~S~~      | ~~Documentation~~ |
| 35 | Capture a fresh `reports/bench` baseline (-benchmem -count=6) before the next perf-relevant change                                                                                                                                                       | Low    | S      | Process       |
| 36 | Draft the v1.0.0 shim-removal checklist (deprecated.go + root doc.go + deprecated_test.go)                                                                                                                                                               | Low    | M      | Cleanup       |
| 37 | Unpark criteria review for a go-etag/otel subpackage, post-v0.3.0 (old roadmap item)                                                                                                                                                                     | Low    | S      | Feature       |
| 38 | Release-cadence rule for ROADMAP (old open item)                                                                                                                                                                                                         | Low    | S      | Process       |
| 39 | Cosign/SBOM evaluation for releases (old open item)                                                                                                                                                                                                      | Low    | M      | Process       |
| 40 | Decide the release-workflow open question (old ROADMAP OQ3)                                                                                                                                                                                              | Low    | S      | Process       |
| 41 | TODO #27-carryover: retire/rotate the /tmp verification pattern into the repo (v0.4.0 report table)                                                                                                                                                      | Low    | S      | Cleanup       |
| 42 | Re-run `go test -bench=. -count=1` smoke on server package to confirm no perf surprises since 09-18 typed-validator baselines                                                                                                                            | Low    | S      | Quality       |
| 43 | Verify the CI fuzz job still wires both client fuzz target and server targets (AGENTS.md claim)                                                                                                                                                          | Low    | S      | Quality       |
| ~~44~~ | ~~Add a short "verification gate" checklist to AGENTS.md Commands (fmt + lint + race + vet + art-dupl) so first-pass scope is unambiguous~~ done 2026-09-23 — AGENTS.md Commands carries the gate (incl. art-dupl + nolint-audit supplements)            | ~~Medium~~ | ~~S~~      | ~~Documentation~~ |
| 45 | Consider making the session-start ritual explicit in AGENTS.md: read Repo Workflow Notes before touching shared files (daemon rewrites)                                                                                                                  | Low    | S      | Documentation |
| 46 | Audit whether any other doc comments in the repo state tool-output facts that can rot (grep for "art-dupl", "0 issues", "reports" in comments)                                                                                                           | Low    | S      | Documentation |
| ~~47~~ | ~~Check FEATURES.md freshness (docs-health) — last touched before this session~~ done 2026-09-23 — metrics section present; coverage stamp refreshed (root 100.0 / server 98.6 / client 99.1 / entitytag 98.9 / metrics 96.8)                            | ~~Low~~    | ~~S~~      | ~~Documentation~~ |
| 48 | ROADMAP: record "Vary-aware secondary keys remain out of scope" as an explicit decision with date (currently only an AGENTS.md gotcha)                                                                                                                   | Low    | S      | Documentation |
| 49 | Verify `go.work` workspace consumers (if any) still resolve after recent daemon go.mod touches (GOWORK=off + workspace MVS per recipe)                                                                                                                   | Low    | S      | Quality       |
| 50 | Write the cross-project lesson "tool preconditions live in env, not in the command name (golangci-lint inherits GOTOOLCHAIN)" to references/lessons.md in crush-config (by commit, not in-session write)                                                 | Low    | S      | Process       |

Items 1–14 are the actionable core; 15–50 are brainstorm-grade and should be
routed to ROADMAP unless HARVEST promotes them.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Enforcement policy for the accepted clone:** should the art-dupl decision
   be machine-enforced (committed baseline file / CI check asserting exactly
   one known group), or is prose-only acceptance the intended standing policy?
   I checked: no baseline or CI enforcement exists anywhere in the repo
   (deferred since 2026-08-07), and each future dedup pass will re-triage this
   group from scratch without a gate. What I tried: grep across all status
   reports and AGENTS.md — the question is explicitly recorded as deferred,
   never decided.
2. **Standing status-report format:** the status-report skill's canonical
   output is a styled HTML dashboard; you explicitly asked for `.md` this
   time. Should future status reports be Markdown (as instructed today) or
   back to HTML? I cannot infer a standing preference from one instruction,
   and the divergence is flagged per skill contract.
3. **Harvest timing:** should I run docs-health HARVEST now (pulling this
   report's section f into TODO_LIST.md / ROADMAP.md), or do you want to
   triage the 50 items first? You instructed me to wait for instructions, so
   I am asking rather than harvesting on my own.

---

_Verification at report time: `golangci-lint run` 0 issues · `golangci-lint fmt`
clean · `go test -race ./...` ok (4 pkgs) · `go vet ./...` clean ·
`art-dupl -t 1 --type-aware` exactly 1 accepted group · session edits in daemon
commits `0c2201e` and `969d077`._

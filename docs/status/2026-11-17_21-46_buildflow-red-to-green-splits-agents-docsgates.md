# Status Report: BuildFlow Red-to-Green — File Splits, AGENTS Rewrite, Docs-Gates Repair

**Date:** 2026-11-17 21:46
**Session scope:** Drive `buildflow --fix --build-mode full` from exit 69 (13
error findings) to green: file-size-check (7), docs-gates (4),
go-structure-linter (2), plus triage of the warning classes.

**Scope line — VERIFIED this session:** full `$PACKAGES` build + vet +
`go test -race -count=1` (all five modules, repeatedly), `golangci-lint fmt` +
`run` (0 issues × 5 modules), `art-dupl -t 1 --type-aware` (exactly the 4
accepted groups), `docs-gates annotate` (CLEAN, 112 findings → 0), targeted
`buildflow -s file-size-check` / `-s go-structure-linter` re-runs (both error
classes gone), `scripts/doc-snippet-refs` (green incl. new playbook).
**NOT verified / not yet green:** `docs-gates freshness`, `--audit-all`,
`cadence` (fixes designed, facts gathered, verdicts not yet written — see §b);
the full `buildflow` gate has NOT been re-run end-to-end yet; govulncheck
GO-2026-6859 not yet dispositioned.

---

## a) FULLY DONE

1. **Triage of the exit-69 run** — full findings lists for all three error
   tools and every warning class; confirmed the daemon committed BuildFlow's
   auto-fixes (ci.yml action pinning, `f7a7540`); confirmed no `.buildflow.yml`
   exists (fleet defaults apply); confirmed the stale-binary and SQLITE_BUSY
   preflights are advisory only.
2. **file-size-check: 7 errors → 0.** All Go files now ≤350 lines:
   - `server/etag.go` (422) → `server/config.go` (189: ETagConfig surface,
     defaults, constructor) + `etag.go` (257: etagWriter runtime).
   - `client/transport.go` (580) → `rebuild.go` (161: 304→200 rebuild,
     freshening persistence, store) + `header.go` (175: header-field
     semantics, no-store parsing) + `body.go` (37: chainedBody,
     drainAndClose) + `transport.go` (267: core RoundTrip).
   - `entitytag/entity_tag_test.go` (380) → + `parse_test.go` (184).
   - `server/etag_test.go` (971) → 9 topical files (buffer, notmodified,
     streaming, skip, config, hooks, fuzz, bench, conditional).
   - `server/etag_bdd_test.go` (652) → 4 files (ifmatch, head, hooks_bdd).
   - `client/transport_test.go` (1003) → 5 files (testutil, caching,
     validator, header + remainder 340).
   - `client/spec_test.go` (1524) → 8 files (freshen, freshen-exceptions,
     storage, conditional, invalidation, head, passthrough + remainder 102).
   Verified per module after each split (race tests) and at the end
   (lint 0 issues, art-dupl baseline intact at exactly 4 groups).
3. **go-structure-linter: 2 errors → 0.** AGENTS.md rewritten 251 → 208
   lines: 13 linter `###` sections compressed to one table; both Release
   Conventions sections moved verbatim into new `docs/release-playbook.md`
   (pointer kept in AGENTS.md); new **BuildFlow Gate** section (350/220/14-day
   limits, lo.Map rejection rationale, examples/-in-package and no-internal/
   deliberate non-fixes, govulncheck routing policy); file table and testing
   conventions updated to the new layout; F/spacing violations fixed.
   `doc-snippet-refs` green over the new playbook.
4. **docs-gates annotate: 112 findings → CLEAN** (one of four gate errors):
   - C/path (18): verdicts cited `docs/planning/2026-09-23_14-07` — repointed
     at the real file `docs/planning/2026-09-23_14-07_v060-staircase-and-full-backlog.md`.
   - G/uniform (4): boundary-crossing strikes repaired (ID-cell-only form).
   - F/spacing (5): inline-code spacing restored (`carried —`s → `replace`
     directives etc.).
   - A/evidence (83): verdicts hoisted out of strike spans and normalized
     into the judge's actual vocabulary (read from docs-gates source:
     `done|resolved|superseded|fixed|adopted|decided|verified|won't implement|
     not-do|duplicate|struck|moot|no longer` — "carried/closed/answered/folded"
     are NOT members).

## b) PARTIALLY DONE

5. **docs-gates freshness** (1 unharvested §f item in the 2026-09-24 report):
   facts fully gathered — origin item is the 2026-09-22_21-36 report #50
   ("tool preconditions live in env, not in the command name"), and
   crush-config `references/lessons.md:158` already carries a
   golangci/GOTOOLCHAIN lesson (the devShell-fix one). **Remaining:** decide
   landed-vs-wording-mismatch (question g.2), write the inline verdict.
6. **docs-gates audit-all** (11 unharvested historical items): every item
   fact-checked against the tree — Content-Type tests absent,
   `http.NewResponseController` absent, funcorder not in `.golangci.yml`,
   SECURITY.md present, `.github/ISSUE_TEMPLATE` absent, ROADMAP already
   carries the stale-while-revalidate/stale-if-error line. **Remaining:**
   write the 11 inline verdicts + amend TODO_LIST rows #2/#3/#4 so the
   routings are honest (test-batch additions, funcorder fit, issue/PR
   templates).
7. **docs-gates cadence** (ARCHIVE-OVERDUE): the 2026-09-24 report (54 days
   old) needs the archive ritual (check-rows --strict + bare-rows BEFORE
   `git mv`, Live-remainder banner). **Blocked on** items 5–6 resolving its
   last open items.

## c) NOT STARTED

8. **Warning fixes:** actionlint SC2086 ×5 (unquoted `$PACKAGES` in ci.yml —
   intentional word-splitting; needs `# shellcheck disable=SC2086` or an
   array refactor, mindful of the ci.yml↔script PACKAGES parity check);
   lychee ROADMAP.md:25 (`file:///home/lars/projects/entitytag/README.md` —
   stale absolute link); `buildflow format` re-run (formatter
   fingerprint-drift hint); optional VACUUM of the 1.02 GB buildflow state DB.
9. **govulncheck GO-2026-6859** (net/http MaxBytesReader header race, stdlib):
   per the new AGENTS.md policy this routes to TODO_LIST as a fleet-coordinated
   Go bump, not a unilateral go-directive change — row not yet added.
10. **Final full `buildflow --fix --build-mode full` re-run** to confirm the
    gate exits 0 (after §b completes).

## d) TOTALLY FUCKED UP

11. **Splitter double-run clobbered two test files.** Re-running the splitter
    where the second run's targets (`body_test.go`, `validator_test.go`) were
    also first-run targets REWROTE them, silently deleting ~5 test functions
    — and the suite stayed green because deleted tests cannot fail. Caught by
    self-audit, recovered byte-exact from git (`d992d94`), rebuilt the tooling
    merge-safe (`move_funcs.py` appends, never overwrites), redid all client
    splits in one pass, and proved conservation with a parser-based decl
    audit (0 missing, 0 true dupes). Lesson recorded: mechanical file surgery
    requires a conservation audit immediately — green tests prove nothing
    when the tests themselves are the cargo.
12. **Misdiagnoses that cost iterations:** (a) assumed multi-line strikes were
    the annotate problem — it was verdict vocabulary (found in docs-gates
    source after ~4 blind transform rounds); (b) misread an `ls` blend and
    "fixed" 18 citations toward `archived/` before checking the file actually
    lives in `docs/planning/` (one corrective sed pass); (c) burned an edit on
    a rejoin probe that was harmless but unnecessary.
13. **/tmp volatility:** the splitter script vanished mid-session; recreated
    inline (heredoc). Minor, but tools for a long session belong somewhere
    durable.

## e) WHAT WE SHOULD IMPROVE

- **Read the judge's source before mass-editing for a gate.** markerRe was 30
  lines into a local checkout; 10 minutes of reading would have replaced four
  regex-guess iterations on 83 lines.
- **Mechanical doc/code surgery needs the conservation audit as a built-in
  step**, not an afterthought (the decl-audit script should exist before the
  first split runs).
- **Re-dump the findings list after every transform batch** — one pass ran
  against a stale list.
- **Formatter-order canon not yet executed for the annotate sweep:** the
  sequence check-rows → `buildflow format` → check-rows AGAIN over the
  touched corpus is still pending (10 files edited).
- **Art-dupl's BuildFlow face (30 findings) vs the release gate's "exactly 4
  groups"** — two configs over one tool; the delta is test-file filtering.
  Worth a config alignment pass so audits stop re-triaging the same noise.

## f) Top things to get done next (impact-ordered)

1. Resolve freshness item: verdict on 09-24 §f #9 (GOTOOLCHAIN lesson) —
   landed-vs-add-exact-wording (g.2 below).
2. Write the 11 audit-all inline verdicts (facts already gathered).
3. Amend TODO_LIST #2/#3/#4 with the honest routing targets (Content-Type +
   NewResponseController tests, freshenedHeader fuzz → #2; funcorder fit → #3;
   issue/PR templates → #4).
4. Run the cadence archive ritual for the 2026-09-24 report (gates BEFORE
   `git mv`; Live-remainder banner).
5. Re-run `docs-gates` battery → all four gates green.
6. Fix ci.yml SC2086 ×5 (shellcheck directive or array; keep PACKAGES parity
   check green).
7. Fix ROADMAP.md:25 broken file link.
8. Route govulncheck GO-2026-6859 to TODO_LIST per AGENTS policy (fleet Go
   bump decision).
9. Run the formatter-order canon over the 10 annotate-repaired files
   (check-rows → buildflow format → check-rows).
10. Re-run the FULL `buildflow --fix --build-mode full` gate → exit 0.
11. Commit-verify: confirm the daemon captured every working-tree change
    (report + docs repairs) with `git status` at session end.
12. erraudit + nolint-audit per module (full verification-gate hygiene after
    the splits — cheap, not yet run this session).
13. `buildflow format` full pass (formatter fingerprint re-verify after the
    toolchain drift warning).
14. Rebuild the stale buildflow binary (advisory; c996df0 vs 541898c in the
    BuildFlow repo — potentially a concurrent lane; check first).
15. VACUUM the 1.02 GB buildflow state DB (or let post-purge GC handle it).
16. Alignment pass: art-dupl BuildFlow config vs the release gate's 4-group
    baseline (kill the 26-finding delta noise).
17. TODO_LIST #8: 30-day policy-vet review (due 2026-12-06 — calendar item).
18. Freshness follow-up: `docs/status/` will contain this report next — plan
    its own harvest/archive cadence.
19. (Carried) TODO_LIST #1–#7 backlog remains the standing ranked list —
    untouched by this session except the planned #2/#3/#4 amendments.
20. CHANGELOG entry for the layout refactor once the gate is green (structural,
    non-breaking; consumers unaffected — package paths unchanged).

## g) Top questions I can NOT figure out myself

1. **govulncheck GO-2026-6859** (stdlib net/http MaxBytesReader race; the
   fix is a Go patch release): route to TODO_LIST as a fleet-coordinated
   `go 1.27.x` floor bump (5 go.mod files + 3 CI pins + README badge move
   together), or start that coordinated bump now from this repo?
2. **The GOTOOLCHAIN lesson item:** crush-config `lessons.md:158` already has
   a golangci/GOTOOLCHAIN cross-project lesson (the devShell-fix one), but
   the origin item #50's wording is "tool preconditions live in env, not in
   the command name". Treat as landed (verdict: done, cite lessons.md:158),
   or commit the exact lesson wording to crush-config first?
3. **ci.yml `$PACKAGES` SC2086:** the unquoted expansion is intentional
   word-splitting and the parity check greps the literal pattern line — is a
   `# shellcheck disable=SC2086` directive acceptable, or do you want the
   array refactor in both ci.yml and pre-release-check.sh?

---

**Live remainder:** sections b/c/f own the open work; this snapshot is
point-in-time. The living backlog is `TODO_LIST.md`.

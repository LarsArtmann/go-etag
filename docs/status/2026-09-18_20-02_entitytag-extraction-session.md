# Status Report — 2026-09-18 20:02 CEST — entitytag Extraction Session

**Session scope:** Execution of ROADMAP OQ1's shared-subpackage decision (user:
"client imports server's ETag looks problematic — go fix!"). Extracted the
`entitytag/` package, re-exported from `server/` via aliases + wrappers, adopted
typed validator comparison in the client, ran all quality gates, updated docs.
Report covers this session's run and what it noticed — no unrelated research.
`.md` format per explicit user instruction (overrides the status-report skill's
HTML default).

---

## Executive Summary

ROADMAP Open Question 1 is resolved and executed end-to-end: the entity-tag
domain type now lives in a shared `entitytag/` package, the dependency direction
is `server → entitytag ← client` (the `client → server` edge can no longer
exist), and the client compares validators through the typed parser instead of
hand-rolled `W/` prefix stripping — closing the exact malformed-validator bug
class Theme 2 called out. Every gate is green (build, vet, race, lint 0 issues,
erraudit 0 violations, fuzz smoke ×3) and the change is benchmark-neutral with
before/after baselines archived.

Two scars: the editor-side LSP is unusable all session (`GOTOOLCHAIN=local`
go1.26.7 vs go.mod's 1.27.1 — user-config decision, still open), and a parallel
session worked the same files simultaneously (it synced CI pins and harvested
the 19:11 report mid-flight; my TODO_LIST append hit a mod-time race and needed
a re-read before landing).

---

## Self-Critique (What did I forget? What could I have done better?)

**What I forgot:**

- **FEATURES.md and README.md were not updated.** Neither mentions the new
  `entitytag` package or the `Strength.IsValid` export (verified: zero hits for
  "entitytag" in both files). The inventory is now stale against the code.
- **Root cause of the nolintlint finding left unexplained.** I removed the now-
  unused `//nolint:exhaustruct_v5` on `server/etag.go:361` because the linter
  reported it unused — but I never established WHY the alias made exhaustruct_v5
  stop flagging `ETag{}` (cross-package alias analysis? ignore-pattern matching
  on the declaring package?). The suppression is gone and nothing fires today;
  if the config or Go changes, that zero-value return may resurface unprotected.
  Behavioral verification, not root-cause verification.
- **The deprecated root shim was never explicitly reviewed.** It passes tests
  (type identity flows through the aliases), but I never confirmed the parity
  suite actually pins the wrapper-function indirection rather than just
  compiling it.
- **Sloppy tool discipline early on:** two consecutive failed `edit` calls
  (no-op replacement, then same line again) before properly viewing the file;
  a `-fuzz FuzzParseETag` pattern that matched two targets and produced a false
  FAIL that cost a diagnostic cycle.

**What I could have done better:**

- Anticipate `gochecknoglobals` before writing the shim: the lint config was
  readable upfront, and the repo's own pattern (deprecated.go uses wrapper
  functions, not vars) already answered the question. First version used
  `var (...)` aliases and had to be rewritten.
- Interleave benchmarks from the first A/B. The initial before/after pair ran
  minutes apart under different machine load and showed a phantom ~40%
  "regression" (1398 vs 1003 ns/op min) that the identical allocs/op
  (21, 2576 B) already flagged as noise. The interleaved rerun settled it at
  ~2-3% — but the noisy files existed briefly as wrong evidence.
- Check FEATURES.md/README.md as part of the standard "wiring" pass, not just
  AGENTS/CHANGELOG/ROADMAP/TODO_LIST.

**What I can still improve:** everything in section (e).

---

## a) FULLY DONE (verifiable)

| Item                                                                                                                                                                                                                                   | Evidence                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `entitytag/` package extracted: `ETag`, `Strength`, `Strong`/`Weak`, `NewETag`, `ParseETag`, `ParseETagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`, tests + both fuzz targets moved verbatim; `Strength.IsValid` exported            | `entitytag/entity_tag.go`, `entitytag/doc.go`; `git mv` history           |
| `server/entity_tag.go` rewritten as re-export shim (type aliases + 5 wrapper functions); zero consumer break                                                                                                                           | deprecated-shim parity suite passes unchanged; full `go test -race` green |
| `server → entitytag ← client` direction established; client imports only `entitytag`                                                                                                                                                   | `client/transport.go` imports; `go build ./...` green                     |
| Client validator comparison typed: parse + `WeakEqual` replaces `W/` prefix stripping; malformed validators (unclosed quote, bare `*`, lowercase `w/`) never weak-match                                                                | client/transport.go:509-518; two new table cases pinning the semantics    |
| All gates green: build, vet, `go test -race`, golangci-lint 0 issues, erraudit 0 violations, `golangci-lint fmt`                                                                                                                       | gate outputs, 2026-09-18 ~19:50                                           |
| Fuzz smoke: `FuzzParseETag`, `FuzzParseETagList` (8s each), `FuzzHasNoStoreDirective` (8s) — all PASS                                                                                                                                  | fuzz outputs                                                              |
| Benchmark A/B (interleaved): 304-rebuild ~1050-1120 ns/op both arms, identical 21 allocs/op; baselines archived                                                                                                                        | `reports/bench/2026-09-18_{before,after}-typed-validator.txt`             |
| Docs: AGENTS.md (4-package architecture, dependency-direction rule, `GOTOOLCHAIN=auto` command note, typed-comparison gotcha), CHANGELOG (Added + Changed), ROADMAP OQ1 annotated resolved, TODO_LIST row 7 (storedResponse evolution) | file diffs; auto-commits b36f2f0…dec82f2                                  |
| CI pins verified already synced to `go1.27.1` by the parallel session (was go1.26.7 at 19:11)                                                                                                                                          | `.github/workflows/ci.yml:24,58,76`                                       |

## b) PARTIALLY DONE

| Item                         | Works                                                                                                 | Open                                                                                                                                                                     | Blocker                                  | Effort |
| ---------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------- | ------ |
| Theme 2 (typed client cache) | Shared type exists; comparisons typed; direction clean                                                | `cacheEntry` still stores the validator as a raw string; parsed `ETag` should live on the entry (storedResponse evolution) — TODO_LIST row 7                             | None                                     | M      |
| Go 1.27.1 adoption           | go.mod, CI pins, docs, and all gates green under `GOTOOLCHAIN=auto`; AGENTS.md documents the override | Local dev shells on go1.26.7 with `GOTOOLCHAIN=local` still fail — the persisted `go env` is deliberately untouched; also gopls/LSP dead all session for the same reason | User config / toolchain install decision | S      |
| Docs wiring                  | AGENTS/CHANGELOG/ROADMAP/TODO_LIST updated                                                            | FEATURES.md + README.md still describe the two-package world; no `entitytag` mention                                                                                     | None                                     | S      |
| nolint removal on `ETag{}`   | Lint is green with the directive removed                                                              | Root cause of why exhaustruct_v5 stopped flagging the alias zero-value is unknown; suppression is now absent if the situation changes                                    | Investigation only                       | S      |

## c) NOT STARTED (per living docs — none re-verified as started elsewhere)

- Everything downstream of the extraction: freshness serving (§4.2),
  stale-while-revalidate/if-error, Last-Modified second validator, client
  hooks, per-host key partitioning, disk persistence, singleflight, website,
  comparison table, awesome-go, OTEL recipe — all still ROADMAP-level.
- OQ2 (Alex reply email + testdata fixtures), OQ4 (FNV standing decision),
  OQ5 (~25 archive items promote-or-die), OQ6 (shim scope), OQ7 (constructor
  trim) — all still pending owner decisions.
- TODO_LIST rows 1-6 from the parallel session's harvest (v0.3.2 cut, error-code
  single-sourcing, benchmark backfill, GoDoc examples, docs-health passes).
- Note: none of these were touched this session by design — scope was OQ1.

## d) TOTALLY FUCKED UP

1. **The in-editor toolchain is broken for every future session until resolved.**
   - What: persisted `go env GOTOOLCHAIN=local` + local go1.26.7 vs go.mod's
     `go 1.27.1`. gopls and the golangci-lint LSP failed on every single file
     read this session (9+ project errors per tool result). Only CLI runs with
     the per-invocation `GOTOOLCHAIN=auto` override work.
   - Severity: not blocking (CLI is the authority and is green), but every
     session pays the diagnostic noise tax, and tool-output-vs-CLI disagreement
     is exactly the pattern that has historically caused bad edits.
   - Root cause: the 1.27.1 bump (commit 7ae7501) changed go.mod without
     touching the machine's go env; the persisted env is user config I will not
     modify unasked.
   - Mitigation: install go1.27.1 locally (or `go env -w GOTOOLCHAIN=auto`) —
     owner call; AGENTS.md documents the interim override.

2. **Concurrent-session collision is being survived by luck, not mechanism.**
   - What: a parallel session edited CI, TODO_LIST, ROADMAP, and AGENTS while
     this session worked. My TODO_LIST append failed once on a mod-time race;
     ROADMAP's OQ3 wording changed under me between reads; line numbers cited
     in my own 19:11 report drifted within the hour.
   - Severity: medium — re-read-before-edit caught everything this time, but
     the auto-daemon's "heuristic" commits interleave three authors (me, the
     parallel session, the daemon) into an unreadable history.
   - Mitigation: existing discipline (re-read immediately before edit) is the
     only defense; a session-lock or per-session branch convention would be
     the real fix.

3. **Self-inflicted cycle waste (minor, honest):** two no-op edit attempts on
   the same line, and a `-fuzz` pattern mismatch that reported FAIL for what
   was a target-selection error. Cost: ~3 wasted tool rounds. Root cause:
   editing from memory instead of viewing, and unanchored regex.

## e) WHAT WE SHOULD IMPROVE

1. **Standard "wiring checklist" for structural changes** — this session
   updated AGENTS/CHANGELOG/ROADMAP/TODO_LIST but missed FEATURES/README until
   report time. Fix: a fixed post-change doc checklist (all six living docs) or
   a docs-health VERIFY pass keyed on `git mv`/new-package events.
2. **Benchmark A/B protocol: always interleave.** A run-to-run load delta of
   40% on this machine makes sequential before/after meaningless below
   millisecond scale. Fix: interleave arms in one command and require
   allocs/op equality as the noise tripwire (that signal caught it here).
3. **Suppressions need root-cause notes.** Removing a nolint because the
   linter stopped firing is fragile without knowing why it stopped. Fix: when
   a directive becomes unused, record the mechanism (here: unknown, needs one
   investigation) before deleting.
4. **LSP/toolchain split-brain.** Editor tools and CLI disagreeing for a whole
   session is chronic noise. Fix: the owner resolves d#1; short of that,
   sessions should treat LSP project diagnostics as dead by default when
   `go env GOTOOLCHAIN` mismatches go.mod.
5. **Parallel-session convention.** Three writers, one working tree, heuristic
   commits. Even a one-line convention ("session announces its file set in
   TODO_LIST before editing") would make collisions visible.

## f) Up to 50 things we should get done next

Ranked by impact; brainstorm input for docs-health HARVEST (route to
TODO_LIST only items 1-12; the rest are ROADMAP fuel).

| #  | Task                                                                                                                             | Impact | Effort | Category      |
| -- | -------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Resolve the local toolchain split: install go1.27.1 or set `GOTOOLCHAIN=auto` persistently (unblocks LSP, kills session noise)   | High   | S      | Bug           |
| 2  | Update FEATURES.md and README.md for the `entitytag` package + `Strength.IsValid` (stale inventory)                              | High   | S      | Documentation |
| 3  | Investigate why exhaustruct_v5 no longer flags `ETag{}` post-alias; record root cause in AGENTS.md, restore protection if needed | Medium | S      | Quality       |
| 4  | TODO_LIST row 7: store parsed `entitytag.ETag` on `cacheEntry` (storedResponse evolution)                                        | Medium | M      | Feature       |
| 5  | Verify the deprecated-shim parity suite actually pins wrapper indirection (add explicit type-identity assertions if not)         | Medium | S      | Quality       |
| 6  | Cut v0.3.2 (typed `Code` surface + BREAKING sentinel change + min-Go 1.27.1 + entitytag package) with full release gates         | High   | M      | Release       |
| 7  | TODO row 3: single-source error-code list via `allETagErrorCodes`                                                                | Medium | S      | Quality       |
| 8  | TODO row 4: benchmark backfill for typed-code change (-benchmem -count=6 baselines)                                              | Medium | S      | Quality       |
| 9  | TODO row 5: GoDoc examples for `Code`/`DomainOf`/`InDomain`                                                                      | Low    | S      | Documentation |
| 10 | TODO row 6 + row on error docs: docs-health VERIFY/ANNOTATE passes                                                               | Low    | S      | Documentation |
| 11 | Add `Example` functions for the `entitytag` package (ParseETag round-trip, weak vs strong)                                       | Low    | S      | Documentation |
| 12 | Decide OQ2: Alex reply email + `client/testdata/` fixtures                                                                       | Medium | S      | Decision      |
| 13 | Decide OQ3: tag-triggered Release workflow vs documented manual-only                                                             | Medium | S      | Decision      |
| 14 | Decide OQ4: affirm FNV-64a `Strong` standing decision                                                                            | Low    | S      | Decision      |
| 15 | Decide OQ5: ~25 archive open-low items promote-or-die                                                                            | Low    | S      | Decision      |
| 16 | Decide OQ6: deprecated shim scope (typed surface in v0.3.x vs frozen)                                                            | Low    | S      | Decision      |
| 17 | Decide OQ7: keep all six error-family constructors or trim                                                                       | Low    | S      | Decision      |
| 18 | Design opt-in freshness serving (§4.2 `max-age`/`Expires`) — default stays accelerator                                           | High   | L      | Feature       |
| 19 | Implement §4.2 freshness serving behind the opt-in                                                                               | High   | L      | Feature       |
| 20 | §4.2 spec tests + rfc9111-conformance.md updates                                                                                 | High   | M      | Quality       |
| 21 | `stale-while-revalidate` support                                                                                                 | Medium | M      | Feature       |
| 22 | `stale-if-error` support                                                                                                         | Medium | M      | Feature       |
| 23 | `Last-Modified`/`If-Modified-Since` as second validator type                                                                     | Medium | L      | Feature       |
| 24 | Client hooks `OnHit`/`OnStore`/`OnFreshen`/`OnInvalidate`                                                                        | Medium | M      | Feature       |
| 25 | Per-host key partitioning by default                                                                                             | Medium | M      | Feature       |
| 26 | Opt-in disk persistence                                                                                                          | Low    | L      | Feature       |
| 27 | Singleflight request coalescing                                                                                                  | Medium | M      | Feature       |
| 28 | Project website launch (Astro/Starlight sibling pattern)                                                                         | Medium | L      | Documentation |
| 29 | Demo video for the website                                                                                                       | Low    | L      | Documentation |
| 30 | Comparison table vs other Go ETag/caching libraries                                                                              | Medium | M      | Documentation |
| 31 | awesome-go submission                                                                                                            | Low    | S      | Documentation |
| 32 | OTEL/Prometheus wiring recipe on existing hooks                                                                                  | Medium | S      | Documentation |
| 33 | `go-etag/otel` sub-module (stays demand-gated)                                                                                   | Low    | L      | Feature       |
| 34 | Root shim deletion at v1.0.0                                                                                                     | Low    | S      | Cleanup       |
| 35 | Define v1.0.0 criteria                                                                                                           | Medium | S      | Documentation |
| 36 | Extend client fuzz coverage to Cache-Control directive variants                                                                  | Medium | M      | Quality       |
| 37 | Design a build-gate so go.mod commits cannot land red (hook/sweep rule)                                                          | Medium | M      | Quality       |
| 38 | Concurrent-session convention (announce file set; or session locks)                                                              | Medium | S      | Process       |
| 39 | Benchmark A/B: codify the interleave + allocs-equality protocol in AGENTS.md benchmark discipline                                | Low    | S      | Process       |
| 40 | Post-change doc checklist (six living docs) to stop FEATURES/README drift                                                        | Medium | S      | Process       |
| 41 | Sweep ROADMAP Theme 2 wording: mark the subpackage half done, point at TODO row 7                                                | Low    | S      | Documentation |
| 42 | `Strength.String()` method (archive item) if OQ5 promotes it                                                                     | Low    | S      | Feature       |
| 43 | obs-text validation in the parser (archive item) if OQ5 promotes it                                                              | Low    | S      | Feature       |
| 44 | Single-pass `ParseETagList` (archive item) if OQ5 promotes it                                                                    | Low    | S      | Feature       |
| 45 | Isolated parser benchmarks (archive item) if OQ5 promotes it                                                                     | Low    | M      | Quality       |
| 46 | erraudit `nolint-audit .` pass to revalidate the drainAndClose suppressions post-refactor                                        | Low    | S      | Quality       |
| 47 | Re-run full fuzz jobs at CI duration (not 8s smoke) before the next tag                                                          | Medium | M      | Quality       |
| 48 | Push cqrs-htmx's 7 alignment commits (carried from 19:11 report; still open there)                                               | High   | S      | Feature       |
| 49 | Fix cqrs-htmx pre-existing `GOWORK=off` failures (carried)                                                                       | Low    | M      | Bug           |
| 50 | Consumer sweep after v0.3.2: httputil/cqrs-htmx version surfaces + workspace both-modes verify                                   | High   | M      | Quality       |

## g) Questions I cannot answer myself

1. **May I fix the local toolchain split (d#1)?** Installing go1.27.1 locally or
   running `go env -w GOTOOLCHAIN=auto` touches machine/user config that I have
   deliberately not modified. Which do you want — and is go1.26.7 still needed
   by other projects on this machine (which would rule out a global flip)?
2. **Who is the parallel session and what is its scope?** It synced CI pins,
   harvested my 19:11 report into TODO_LIST rows 1-6, and reworded ROADMAP OQ3
   while I worked. I cannot know what it plans next; if it is yours, tell me
   which files are mine to edit so we stop racing TODO_LIST.
3. **Is FEATURES.md still a maintained inventory?** It was never mentioned in
   any status report, TODO row, or skill note this or last session, and it has
   drifted behind two structural changes now. Keep it wired (I fix it now) or
   declare it dead (we archive it)?

---

_Point-in-time snapshot; goes stale. Section (f) is HARVEST input for
`TODO_LIST.md` / `ROADMAP.md`. `.md` format honors the explicit user
instruction over the skill's HTML default._

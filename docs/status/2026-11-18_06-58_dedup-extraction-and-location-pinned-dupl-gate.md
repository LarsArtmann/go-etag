# Dedup pass: extraction in doc-snippet-refs, location-pinned art-dupl gate, docs-gates re-green

Session 2026-11-18 06:45–06:58. Mandate: `deduplicate` after an art-dupl run
showing 4 clone groups (3 documented acceptances + 1 new scripts pair). Loaded
the deduplicate-code skill first; judgment per its harmful-vs-acceptable rubric.
Scope discipline: no research beyond the dedup surface; no TODO_LIST rows
added; nothing committed by hand (daemon swept mid-session as usual).

## §a Fully done (verified)

1. **New clone group EXTRACTED** — `scripts/doc-snippet-refs/main.go`'s two
   `if err != nil { Fprintf; Exit(2) }` blocks (the API-load and ReadFile
   failures) collapsed into a `fail(stage, err)` helper plus named exit codes
   (`exitFindings = 1`, `exitUsageFailure = 2`); all four `os.Exit` sites now
   use the constants. The "loading API" message stayed byte-identical; the
   ReadFile failure path gained a `reading input:` stage label (deliberate:
   stage was previously missing — the raw `open …` error carried the context).
2. **Extraction verified** — root `golangci-lint fmt` + `run` (0 issues),
   `go vet ./...` OK, the tool run (`go run ./scripts/doc-snippet-refs
   README.md docs/*.md`) exits 0 with the same success line. Canonical
   `art-dupl -t 1 --type-aware` shows **exactly 3 groups** (the 3 accepted);
   the combined mode (`--suggest-generics --type-aware --timing`) also shows
   3, no generics candidates on the new helper.
3. **Groups 1–3 accepted per policy** — verified against current code:
   `etagWriter.Write` flushed/overflow blocks (`server/etag.go:74-82`/`92-100`),
   §4.3.5 HEAD-confirmation guards (`client/transport.go:188-191`/`198-201`),
   chained-body arms (both in `client/rebuild.go:114-116`/`119-121` now — the
   on-disk AGENTS.md citation still said transport.go for one arm; fixed).
4. **Composition drift root-caused** — the documented 4th acceptance
   (`default: return false` switch arms, §3.2/§3.1) no longer surfaces: both
   switches now live in `client/header.go`, below art-dupl's actionability
   bar after the 2026-11-17 file splits. Meanwhile the scripts pair slipped
   IN. Net: count stayed 4 and the count-only gate masked the swap — this is
   the exact failure mode a cardinality gate cannot see.
5. **Gate hardened and re-baselined** — `scripts/pre-release-check.sh` dupl
   step: 4 → 3 groups, and upgraded from count-only to **per-file location
   pins** (`client/rebuild.go 2`, `client/transport.go 2`, `server/etag.go 2`
   arms each), so any future composition swap goes loud. `bash -n` OK;
   location-extraction pipeline dry-tested against live output; the pinned
   gate ran green inside the full gate.
6. **Docs updated** — AGENTS.md Commands line + Non-Obvious Behaviors bullet
   rewritten (3 accepted; two former acceptances recorded as historical — the
   switch-default pair with rationale-preserved-if-resurfaces, and the
   extracted scripts pair; testutil test-double rule carried forward; OQ8
   history with the count-only blind spot). CHANGELOG `[Unreleased]` gained a
   Changed entry (extraction + re-baseline + swap story). New baseline written
   to disk: `reports/dupl/2026-11-17_art-dupl-baseline.txt` — **deliberately
   NOT git-added per owner instruction (2026-11-18)**.
7. **Full gates green** — `pre-release-check.sh` exit 0 on the post-change
   tree (build/vet/race per module, lint, fmt-diff, erraudit + nolint-audit,
   parity, pinned dupl gate). `docs-gates run .` all 5 PASS — after repairing
   a fresh daemon dprint mangle (`git mv` span → `` `git`mv `` at line 63 of
   the 22-16 report; rewritten to the daemon-proof "a mangled `git mv` code
   span"). Daemon auto-commits captured the change set (bc623a1: main.go;
   0763b3a: AGENTS/CHANGELOG/gate script); the span repair is the only
   uncommitted edit at report time.

## §b Partially done (blocked on owner)

1. **`.gitignore` reports/dupl negation bug — diagnosed, fix DESIGNED but NOT
   applied** (edit rejected by read-before-edit; owner then said report and
   wait). `git check-ignore -v` proves the buildflow block's directory-form
   `reports/` exclusion (line 43) wins over the `!reports/dupl/**` negations
   (lines 67–68): a parent-directory exclusion cannot be re-included under, so
   the negations are dead patterns. Consequence: new baseline files under
   `reports/dupl/` are invisible to `git status` and the daemon's add never
   sees them; the 2026-09-23 baseline survives only because tracked files
   bypass ignore. Same class: `CHANGELOG.md` cites
   `reports/bench/2026-10-15_metrics-hook-overhead-count6.txt`, also untracked.
   Canonical repair (3-step re-inclusion, outside the buildflow markers):
   `!reports/` + `reports/*` + `!reports/dupl/` + `!reports/dupl/**`.
2. **AGENTS.md memory note for the gitignore gotcha** (append to the existing
   daemon/.gitignore bullet, next to the go.work negation precedent) — not
   written, pending the owner's §g-1 decision on which way the fix goes.

## §c Not started (out of scope this session)

- CI port of the location-pinned dupl check (gate is release-local only).
- Any TODO_LIST row additions or harvest/routing changes.
- Upstream erraudit/dprint candidates carried from the 22-16 session — untouched.

## §d Totally fucked up (honesty ledger)

- No damage, no reverts, no lost work. Three friction events, all recovered:
  1. CHANGELOG edit rejected once on staleness (daemon touched the file
     mid-session) — re-read, re-applied.
  2. `.gitignore` edit rejected on the read-before-edit rule — no mutation
     happened; left as §b-1.
  3. Disclosed deviation, not a failure: the ReadFile failure message changed
     shape (`reading input:` prefix added). No test pins stderr; CI checks
     exit behavior only. Flagging in case any consumer scripted around the
     old string (none known in-repo).
- Inherited defect found, not caused: the count-only dupl gate had been blind
  to composition swaps since OQ8 (2026-09-23); the swap itself predates this
  session (file splits, 2026-11-17).

## §e What to improve

- **Cardinality gates are weak pins** — any accepted-baseline gate should pin
  composition (locations), not counts. Done now for art-dupl; audit other
  count-shaped gates for the same blind spot.
- **Evidence files cited but untracked** — gate messages and CHANGELOG cite
  `reports/dupl/` + `reports/bench/` paths that git ignores via a dead
  negation. Either track them deliberately or stop citing them as durable
  evidence.
- **Verify ignore-rule intent with `git check-ignore -v`** — the negation
  lines looked right for two months; one command would have exposed them.
- **Daemon dprint keeps mangling code spans in status reports** — second
  occurrence in two days (F/spacing repair loop is reactive). The check exists
  and catches it; the mangling itself is upstream dprint behavior worth an
  issue (see 22-16 §g leftovers).
- **LSP diagnostics for `scripts/doc-snippet-refs/main.go` are STILL stale**
  after this session's edits (gofumpt/nolintlint warnings quoting a pre-edit
  version; live `golangci-lint run` is clean). golangci_lint_ls cache should
  be investigated or restarted.

## §f Next candidates (owner-routed; no rows written)

1. Decide §g-1, then apply or drop the `.gitignore` re-inclusion shape (**→
   owner decision, this report §g**).
2. Port the location-pinned dupl step into `.github/workflows/ci.yml` as a
   small job (**→ TODO_LIST.md candidate row**, mirrors the §f.1 CI-tripwire
   candidate of the 22-16 report).
3. Track-or-relocate decision for `reports/bench/` baselines given CHANGELOG
   cites one (**→ owner decision**).
4. Investigate/restart golangci_lint_ls for the scripts file (**→ TODO_LIST.md
   candidate row**).
5. Upstream verify-before-filing pair from 22-16: erraudit flagless
   nolint-audit false-stale; dprint code-span spacing (**→ ROADMAP/owner**).
6. Carried, still waiting: owner answers to the 22-16 report §g (parity-gate
   generalization ratify, daemon go-directive fix vs CI tripwire, ratify the
   three autonomous decisions); TODO_LIST #9 floor-bump process.
7. After the daemon's next sweep: one cache-warm `buildflow --fix
   --build-mode full` re-run + `docs-gates run .` to confirm the tree holds
   green with the new pinned gate (the 22-16 session's standing hygiene step).

## §g Questions for the owner (cannot decide alone)

1. **reports/ evidence tracking:** repair the `.gitignore` negations (canonical
   3-step shape above, then baseline files become trackable and the daemon can
   commit them) — or keep gate evidence local-only and strip the dead
   negation lines plus the gate/AGENTS/CHANGELOG file citations? I did not add
   `reports/dupl/2026-11-17_art-dupl-baseline.txt` per your instruction; the
   gate currently cites an on-disk-only file.
2. **Baseline citation while untracked:** if evidence stays local-only, should
   the gate's fail message and AGENTS.md still name the baseline file path
   (local artifact) or describe the accepted set inline only?
3. **CI coverage:** want the location-pinned dupl check in ci.yml (a job), or
   is release-local enforcement the intended boundary?

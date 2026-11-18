# Release Playbook

The full release procedure for go-etag's five-module version train. The
summary lives in AGENTS.md; this file carries the detail. Living document —
update in place when a release validates or corrects a step.

## Single-module era (validated v0.1.0–v0.5.0)

- **Release commit style:** `chore(release): cut CHANGELOG vX.Y.Z` (matches
  v0.2.0/v0.3.0), CHANGELOG cut BEFORE tagging; empty `[Unreleased]`
  placeholders + compare links stay behind.
- **Run `scripts/pre-release-check.sh` on the exact release commit before the
  CHANGELOG cut** — one command for every locally checkable gate (build, vet,
  race, lint, fmt-diff, erraudit + nolint-audit, `go mod verify`,
  replace/pseudo-version ban, and go-directive ↔ CI-pin parity that catches the
  969d077 daemon-relaxation class of drift). v0.4.0 shipped with skipped steps
  (`go mod verify`, nolint-audit); the script exists so that class cannot recur.
- **The daemon can sweep the release-prep commit first** (v0.4.0: the CHANGELOG
  cut landed inside a `chore: auto-commit`). Harmless if the tree content is
  exactly the release content — verify the tag's `.info` hash on
  proxy.golang.org matches the intended commit, as done at v0.4.0 (16369cc).
- **v0.x GitHub Releases are published as Latest, non-prerelease** — house
  precedent (v0.1.0 → v0.5.0), a deliberate deviation from the go-release
  skill's `--prerelease` suggestion.
- **The consumer-facing gates are proxy.golang.org + sum.golang.org + a
  clean-room `go get`** — pkg.go.dev doc rendering lags by minutes and is NOT a
  release blocker; verify it eventually, not gate on it.
- **Tag only after CI is green on the exact commit** (workflow triggers on both
  master pushes and `v*` tags, so the tag gets its own frozen run).
- **The go-release skill lives at `~/.config/crush/skills/go-release/`** — load
  it for any release; it repays its reading cost in immutable-tag discipline.
- **v0.5.0 (2026-09-23, commit `af2b6d1`) validated the full loop with zero
  skipped steps**: pre-release-check.sh exit 0 on the exact commit, master
  pushed so CI ran green on it, annotated tag, frozen tag-run green, proxy
  `.info` hash == commit, sum.golang.org recorded, clean-room
  `go get …/metrics@v0.5.0` + Attach/HitRatio smoke, GitHub Release Latest,
  pkg.go.dev rendering verified within the hour. The floor decision (TODO_LIST
  #2) was resolved by restoring `go 1.27.1` (commit `45e68c8`) to match the
  README badge and all three CI pins.

## Multi-module era (from v0.6.0, after the 2026-09-23 split)

The single-commit discipline becomes a **bottom-up tag staircase** — the price
of five independently consumable modules. Each module's tag points at the
commit where its own go.mod/go.sum finalized; subtree content is identical
across the staircase, which is all consumers ever see.

1. **Prepare:** land all code changes; run `scripts/pre-release-check.sh`
   (workspace mode — every step runs without the sibling tags except
   `go work sync`, which defers in the pre-tag bootstrap window: before the
   staircase pushes the train tags it cannot read the sibling go.mod files
   from VCS and fails with "unknown revision", so the gate tolerates exactly
   that failure shape and MUST be re-run to full green after the staircase).
   Cut the CHANGELOG as usual (`chore(release): cut CHANGELOG vX.Y.Z`).
2. **Staircase, dependency-first — entitytag → server → client → metrics →
   root.** For each module in order, from its directory:
   `GOTOOLCHAIN=auto go mod tidy && GOTOOLCHAIN=auto go mod verify &&
   GOWORK=off go build ./... && GOWORK=off go test -race -count=1 ./...`
   (tidy resolves the already-tagged dependencies below it; the first module,
   entitytag, has none), commit the tidy result if it changed go.mod/go.sum,
   wait for CI green on that commit, then create the ANNOTATED tag:
   `entitytag/vX.Y.Z`, `server/vX.Y.Z`, `client/vX.Y.Z`, `metrics/vX.Y.Z`,
   and finally the root `vX.Y.Z`. Root goes last — it is the "release commit"
   whose CHANGELOG cut the release.
3. **Bump the train for the next cycle:** after tagging, set the sibling
   requires in go.mod files and the replace versions in go.work to the NEXT
   dev version (or leave them at the just-released version and bump at the
   next release — either way the pre-release-check sync gate enforces they
   agree).
4. **Consumer gates per module tag:** proxy `.info` hash == the tagged commit,
   sum.golang.org entry, and a clean-room
   `go get github.com/larsartmann/go-etag/<module>@vX.Y.Z` for each of the
   four nested modules plus the root. Proxy fetches of a fresh tag can lag a
   minute or two; `GOPROXY` direct fallback covers it.
5. **Pre-tag bootstrap (v0.6.0 only):** the nested requires point at v0.6.0
   before any such tag exists. That is expected: workspace mode (go.work)
   keeps every gate green; ONLY the staircase's tidy/GOWORK=off steps need the
   tags to exist, in order. Do not "fix" the unresolvable requires with go.mod
   replace directives — the release gate bans replace in every go.mod, and
   proxy consumers would break.

Everything else carries over from the single-module era. The GitHub Release
page is automated since OQ3 (2026-09-23): pushing the ROOT `v*` tag triggers
`.github/workflows/release.yml`, which extracts the matching CHANGELOG section
as the notes (it fails loudly if the section was not cut) and creates the
Release as Latest, idempotently; nested tags get no Release pages. Keep the
CHANGELOG section format stable — the extractor matches the section heading by
prefix, so restructured headings break notes extraction. The workflow has NOT
fired live yet (verified locally only: extraction, missing-section failure
path, actionlint); its first real run is the next root tag — watch it end to
end (untested-by-default honesty per report 22:07 b4/e4). The proposal doc
`docs/modularization/2026-09-23_modularization-proposal.html` records why
metrics got a module (tombstone purity, not composability) and why go.mod
replace directives stay banned (proxy-published repo).

## v1.0.0 root-module deletion mini-staircase (seed, executed at the major)

The v1.0.0 release is itself a small staircase:

1. Delete `deprecated.go`/`doc.go`/root tests and the root MODULE (its
   go.mod/go.sum), drop the root entry from go.work's `use` + replaces, and
   remove the root pattern from every PACKAGES set (ci.yml +
   pre-release-check.sh — the agreement check now enforces they stay
   identical).
2. Re-run the bottom-up staircase for the FOUR surviving modules at `v1.0.0`
   exactly as above.
3. The v1.0.0 GitHub Release notes carry the root-consumer migration (pure
   `go get …/server` import-path swap).

Consumers importing the bare root path break loudly at `go mod tidy` — that is
the intended forcing function; audit the fleet first with
`rg -l "larsartmann/go-etag\" v" -g go.mod ~/projects` (dead requires at v0.x
can simply drop the line).

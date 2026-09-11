# Status: Six-Consumer `FreshenOn304` Migration Sweep + Self-Review (2026-09-11 02:56)

Session scope: executed go-etag TODO_LIST item 1 (now resolved) — bump all six
in-house direct consumers of `github.com/larsartmann/go-etag` from v0.2.0 to
v0.3.0, migrating the removed `Options.PreserveOn304` API where used, per the
go-ecosystem-upgrade flow. This report is a point-in-time snapshot of that run,
including a brutal self-review. No other projects were audited.

Format note: written as Markdown at the user's explicit request; the
status-report skill's canonical HTML dashboard format was overridden (user
instruction wins).

## The one-paragraph version

All six consumers are on go-etag v0.3.0, verified, and committed locally
(nothing pushed). The TODO's premise was overstated: only `go-github-kit`
actually used the removed `PreserveOn304`; the other five import
`go-etag/server`, which is unchanged, so they were never compile-broken —
they needed bumps + verification only. The run's two honest failures: I
created (and only reactively fixed) a Nix flake-pin drift in DiscordSync, and
DiscordSync's dependency bump got swallowed by the auto-git daemon under a
garbage heuristic commit message. The Nix build layer (flake.lock updated but
`nix build` never run; possible stale `vendorHash` in library-policy's flake)
is the biggest unverified surface this session leaves behind.

## a) FULLY DONE

1. **Mapping derivation (Phase 0).** Old `PreserveOn304 []string` semantics
   extracted from git history (commit `9204885`): `nil` = RFC §4.3.4 freshening,
   non-empty list = freshen-only-these-fields whitelist, empty non-nil slice =
   no freshening. New mapping: omitted / `FreshenPerRFC()` / `FreshenFields(...)`
   / `FreshenNone()`. Exactly one wild usage existed (whitelist), mapping 1:1.
2. **Full enumeration.** `rg` with `--no-ignore-vcs --hidden` over `~/projects`
   found 77 `go.mod` matches; classified into exactly six _direct_ requires
   (the five repos + cqrs-htmx's `examples/middleware-showcase` module; cqrs-htmx
   root is `// indirect` with zero source usage). Code-level grep proved only
   `go-github-kit/etag.go` used `PreserveOn304`.
3. **Baselines before touching anything (F11).** Build + full test per module.
   Recorded: all green except DiscordSync's `TestEvaluateHealthChecks_LabelsResults`
   (pre-existing, in `cmd/discordsync`, unrelated to etag).
4. **go-github-kit — the only real code migration.** `PreserveOn304: []string{...}`
   → `FreshenOn304: etagclient.FreshenFields(...)` with identical field list;
   go.mod v0.3.0; vendor refreshed; build + tests green; zero repo-wide
   residuals. Commit `f3ece58`.
5. **httputil bump.** go.work renamed during `go get`/`tidy`/`vendor` (F2
   recipe), restored; workspace-mode build + tests (incl. `server_timing`
   submodule) green. Commit `ff724e0`.
6. **cqrs-htmx/examples/middleware-showcase bump.** Hermetic `GOWORK=off`
   build green (module has no tests; same as baseline); root module
   deliberately untouched (documented in commit). Commit `533acf40`.
7. **nsfw-classifier + library-policy bumps.** Full suites green; committed
   (`f910b03`, `9c47f43`).
8. **DiscordSync bump + flake repin.** go.mod v0.3.0; flake.nix input repinned
   from the v0.2.0 tag commit (`be19640`) to v0.3.0's (`fe5dede`);
   `nix flake lock --update-input go-etag` refreshed lock + narHash; the
   repo's own drift-guard test passes again. Only the pre-existing baseline
   failure remains. Commits `acfe0598` (daemon — see d2), `42c9d06f`.
9. **Verification gate.** Per consumer: `go.mod` grep = v0.3.0 ✓,
   `go mod verify` ✓, hermetic `GOWORK=off` build ✓; global residual greps for
   `PreserveOn304` in Go code and for stale direct v0.2.0 requires: zero hits.
10. **Backlog + records.** TODO_LIST item 1 resolved (list renumbered, new
    httputil-release item added), status report
    `2026-09-11_02-51_consumer-migration-preserveon304.md` written; committed
    in go-etag as `09a2b1f`.

## b) PARTIALLY DONE

1. **cqrs-htmx alignment** — only its one direct go-etag module (showcase) is
   migrated. The other 43 train-lag modules stay on go-etag v0.2.0 via the
   _published_ httputil tag; moving them requires an httputil release first.
   Partial by design, but still partial.
2. **Nix verification layer** — DiscordSync's flake input and lock are updated
   and the drift guard passes, but `nix build` was never run; and for every
   repo whose go.mod/go.sum I edited (all six), I never checked whether that
   repo's _own_ flake has a `vendorHash`/FOD hash now stale. library-policy's
   own recent commit literally says "vendorHash toolchain gotcha" — a signal I
   saw in its git log and failed to connect (see d3).
3. **Lint gate** — no `golangci-lint`/`go vet` was run on go-github-kit after
   the one real code edit of the sweep. cqrs-htmx's BuildFlow pre-commit ran
   golangci-lint across its modules, but under a starved `/tmp` tmpfs, so its
   findings ("no space left on device" import failures) are inconclusive noise.
4. **Final residual sweep scope** — asserted "zero residual references", but
   the last global grep was filtered to `*.go` + `go.mod`. Markdown/README/
   YAML/CI mentions of `PreserveOn304` in consumer repos are unverified
   (go-github-kit's earlier whole-repo sweep was clean; the others were not
   re-swept whole-repo).
5. **cqrs-htmx workspace-mode verification** — the showcase module was verified
   hermetically; a post-change workspace-level build (go.work MVS unification
   to v0.3.0 across 30 modules) was never explicitly run.

## c) NOT STARTED (known, deliberately out of scope)

1. **httputil release carrying go-etag v0.3.0** — the keystone: without a new
   httputil tag, no consumer of httputil can reach v0.3.0. Requires the
   go-release flow and the owner's GO.
2. **cqrs-htmx family-train alignment** (43 modules) onto that future release.
3. ~~**README "Upgrading" migration diff** (`PreserveOn304` → `FreshenOn304`
   before/after) — TODO_LIST item 2.~~ done 2026-09-11 docs-health pass (README "Upgrading from v0.2.x" section with mapping table)
4. **GitHub Release notes append** for v0.3.0 — TODO_LIST item 3.
5. **Benchmark re-run** at the v0.3.0 tree, archived under `reports/bench/`
   (`-benchmem -count=6`) — TODO_LIST item 4.
6. **CI GOTOOLCHAIN/govulncheck pinning** — TODO_LIST item 5.
7. ~~**FEATURES.md shim-row fix** — TODO_LIST item 6.~~ done 2026-09-11 docs-health pass
8. **Dependabot actions-group red CI** (run 34545953600) — renumbered TODO_LIST
   item 1.
9. **Pushing any commit** — all 8 commits from this session (7 repos' work +
   go-etag docs) are local-only.
10. **DiscordSync's pre-existing health-checks test failure** — untouched (not
    this migration's bug; not my scope without a GO).

## d) TOTALLY FUCKED UP

1. **I caused the DiscordSync flake drift and only caught it because _their_
   guard test did.** I bumped go.mod without first checking the repo's
   version surface (flake.nix input rev). The skill ships an entire reference
   (`version-surface.md`) for exactly this failure class and I never opened
   it. Reactive luck is not a process. Had DiscordSync lacked
   `TestCheckFlakePins_NoSilentSkip`, I would have shipped the drift silently
   into their Nix builds.
2. **DiscordSync's migration commit is mis-attributed to the auto-git daemon**
   (`acfe0598 chore: auto-commit 4 changed file(s) (heuristic)`). The go.mod
   bump — the load-bearing change of that repo's migration — has a message
   that explains nothing and credits no one; only my later `fix(nix)` commit
   hints at it. I was too slow to stage/commit and lost the race. Repository
   history now lies about how the migration happened.
3. **library-policy's `vendorHash` risk: I saw the warning label and walked
   past it.** While checking that repo's commit style I read
   `be3dacb docs(AGENTS): record applied BuildFlow interim fix and vendorHash
   toolchain gotcha` — direct evidence the repo's Nix build has
   vendorHash sensitivities — and did not act on it after subsequently
   editing that repo's go.mod/go.sum. Unverified whether `nix build` there is
   now broken. This is the worst kind of miss: the signal was in my face.
4. **Proceeded through a starved `/tmp`.** cqrs-htmx's preflight reported
   `/tmp (0 MB available)` and multiple tools failed with "No space left on
   device"; I cleaned only my own logs, confirmed 3.5G free, and pushed on
   without treating it as an environment red flag affecting the hook's
   evidentiary value.

## e) WHAT WE SHOULD IMPROVE

1. **Check the version surface before mutating, not after a failure.** For any
   dependency bump in a repo: grep `flake.nix`/`flake.lock` for the dep,
   check for `vendorHash`/FOD hashes, check CI pins — _then_ bump. Fold into
   Phase 0 of every future upgrade run.
2. **Beat the daemon: commit within the same breath as verification.** The
   auto-git daemon races edits in these repos; the fix is mechanical —
   `git add <specific files> && git commit` immediately after the test gate,
   never leaving authored changes uncommitted across tool calls.
3. **Lint is part of the gate.** `go build` + `go test` is not the full
   verification surface for repos with golangci-lint configs (go-github-kit's
   exhaustruct/wsl-style rules were never exercised on my edit).
4. **Nix-flaked repos need `nix build` in the verification gate**, at least
   when go.mod/go.sum changed. go.sum deltas (even 3 lines) are FOD-input
   deltas.
5. **Residual greps must cover all file types.** "Zero references" claims
   deserve whole-repo sweeps, not `*.go`-filtered ones.
6. **Read the truncated evidence fully.** The old `PreserveOn304` doc comment
   was cut off mid-sentence in my extraction ("use it for —"); I inferred the
   rest from the visible half. It happened to be right; the inference should
   have been verified with a couple more context lines.
7. **Workspace repos need both verification modes.** Hermetic (`GOWORK=off`)
   proves the module; workspace mode proves the MVS unification. I only ran
   one mode in cqrs-htmx.
8. **Treat environment red flags (disk, cache pressure) as evidence-quality
   issues**, not noise: a lint pass that died on ENOSPC proves nothing either
   way.
9. **The DiscordSync drift-guard pattern is worth cloning** into the other
   consumer repos (and any repo that pins in-repo deps via flake inputs) — it
   caught exactly the class of mistake a careful agent still makes.

## f) Up to 50 things to get done next

Grouped, impact-ordered. Items 1–10 are this session's own loose ends; 11–20
are the go-etag backlog; 21+ are hardening/process ideas (ROADMAP fuel, not
commitments).

**This session's loose ends (highest impact per effort):**

1. Run `nix build` in DiscordSync to prove the repinned go-etag input builds.
2. Check library-policy's `flake.nix` for `vendorHash` staleness after my
   go.mod/go.sum edit (its AGENTS.md documents the gotcha); fix hash, run
   `nix build`.
3. `nix build` / `nix flake check` in nsfw-classifier, httputil, go-github-kit,
   cqrs-htmx — any repo whose module inputs I changed.
4. `golangci-lint run` on go-github-kit (the only repo with a real code edit).
5. `go vet ./...` on all six repos (cheap, was skipped).
6. Whole-file-type residual sweep for `PreserveOn304` (md/yaml/ci) in the six
   repos to make the "zero references" claim fully true.
7. cqrs-htmx: one workspace-mode `go build ./...` to confirm MVS unification
   to v0.3.0 across the go.work.
8. Add a short note-commit in DiscordSync documenting that daemon commit
   `acfe0598` carried the v0.3.0 bump (never amend daemon commits; annotate
   forward).
9. Free `/tmp` and the buildflow cache (`sqlite3 ~/.cache/buildflow/buildflow.db
   VACUUM`, `nix-collect-garbage -d`) before the next pre-commit hook run.
10. Decide + execute: push the 8 local commits (needs owner GO — see g1).

**go-etag backlog (renumbered TODO_LIST):**
11. Cut the httputil release carrying go-etag v0.3.0 (go-release flow).
12. Align cqrs-htmx's 43 train-lag modules onto the new httputil tag.
13. Diagnose the Dependabot actions-group red CI (run 34545953600); land the
Node 20 bumps.
14. README "Upgrading": add the `PreserveOn304`→`FreshenOn304` migration diff.
15. Append the same diff to the v0.3.0 GitHub Release notes (`gh release edit`).
16. Re-run benchmarks at the v0.3.0 tree; archive `reports/bench/<date>` with
`-benchmem -count=6`.
17. Scope the CI `GOTOOLCHAIN` pin or pin govulncheck so `@latest` can't break
CI.
18. ~~Fix the stale FEATURES.md shim row (export-parity suite exists since
`a5de386`).~~ done 2026-09-11 docs-health pass
19. ~~Annotate report `2026-09-11_02-08` item 2 as done (docs-health inline
resolution) — its "not started" claim is now false.~~ done 2026-09-11 docs-health pass
20. ~~Verify the TODO_LIST renumbering broke no external references to old item
numbers.~~ done 2026-09-11 docs-health pass (every stale numeric ref annotated; harvest added to ROADMAP OQ5)

**DiscordSync / consumer hardening:**
21. Fix `TestEvaluateHealthChecks_LabelsResults` in DiscordSync (pre-existing;
its own session).
22. Clone the flake-pin drift-guard test into the other five consumer repos.
23. Sweep consumer repos for dep pins in CI workflow files / Dockerfiles
(remaining version-surface blind spots).
24. Run `govulncheck` on all six (BuildFlow's attempt died on disk space).
25. `go test -race` on go-github-kit's etag path and DiscordSync's
`internal/api` (concurrent transport paths, never race-run here).
26. Capture DiscordSync baselines with `-count` repetitions in future runs —
the 41s guard test suggests timing sensitivity worth quantifying.
27. go-github-kit: consider exposing a `FreshenPolicy` pass-through in
`ETagOptions` instead of the hardcoded whitelist (API honesty; low).
28. Check whether any of the six repos reference `go-etag` version strings in
READMEs/badges (surfaced-version drift class).

**Process / memory:**
29. Write the consumer-migration recipe into go-etag AGENTS.md repo-workflow
notes: version-surface check before bump; commit-before-daemon; `nix
    build` gate; both GOWORK modes.
30. Encode rule in the global memory: "dependency bump in a Nix-flaked repo ⇒
grep flake.nix for the dep + verify vendorHash + run nix build."
31. Prepare a committed per-repo checklist script for future multi-repo sweeps
instead of ad-hoc shell chains (re-runnable, auditable).
32. docs-health sweep: other in-house libraries may have breaking changes
their consumers haven't absorbed (repeat this session's enumerate-vs-latest
pattern for the next-highest-risk lib).
33. benchstat installation path (nix profile or devshell) so benchmark claims
stop being eyeballed from raw files (carried from report `2026-09-11_02-08`
item 2).
34. Consider making the auto-daemon skip repos with a session-lock file, or
reserve `.config/metadata.yaml`-style paths, so authored commits stop
being pre-empted.
35. Add `/tmp` headroom check to the pre-flight of any flow that triggers
BuildFlow pre-commit hooks.
36. go-github-kit: its 8 MiB default body cap vs the library's 1 MiB default is
intentional but undocumented in its README — one paragraph.
37. Re-check the six repos' Dependabot configs: will dependabot now try
downgrading/no-op on go-etag? Confirm config handles v0.3.0.
38. Sweep for other `.Work.sum`-style stale artifacts (DiscordSync has a
`go.work.sum` with no `go.work`) — cosmetic hygiene.
39. Confirm no consumer repo has a `replace github.com/larsartmann/go-etag`
pointing anywhere local (would have masked the bump; none seen in the six,
but the sweep was only of the six).
40. Post-release (after item 11): re-run the train checker in cqrs-htmx to
watch the 43 lag entries collapse to zero.

## g) Questions I cannot figure out myself

1. **Push authority:** all 8 commits are local. Should I push the six consumer
   repos + go-etag now, or does your auto-daemon/another flow own pushes?
2. **httputil release GO:** cutting the httputil release is the keystone that
   unblocks cqrs-htmx's 43-module alignment. May I run the go-release flow,
   and if so: dependency-pin-only change on top of a published v1.0.0 — do you
   want `v1.0.1`, or do you batch it with other pending httputil work?
3. **DiscordSync ownership:** its pre-existing `TestEvaluateHealthChecks_LabelsResults`
   failure and the (currently unverified) `nix build` both live in DiscordSync,
   not go-etag. Fix/verify them under this session's umbrella, or hand them to
   a DiscordSync-scoped session?

_WAITING FOR INSTRUCTIONS._

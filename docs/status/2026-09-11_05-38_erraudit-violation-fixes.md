# Status Report: erraudit Violation Fixes — Sentinel Interface Widening + drainAndClose Nolint Posture

- **Date:** 2026-09-11 05:38
- **Branch:** `master` @ `2ed5f8c` (auto-daemon commit capturing this session's 4-file diff; working tree clean)
- **Scope:** single-session erraudit sweep (4 ERROR violations) — research, fix, partial verification
- **Input:** user's audit invocation — `GOEXPERIMENT=jsonv2 erraudit ./... --type-aware --enforce-go-error-family --no-suppress --enforce-samber-oops --enforce-generic-return --explain`

## Original findings (4 ERROR violations)

| # | Type | Location | Finding |
| --- | --- | --- | --- |
| 1 | `sentinel_concrete_type` | `server/errors.go:39` | `ErrInvalidConfig` declared `*errorfamily.Error`; must be `error` interface |
| 2 | `sentinel_concrete_type` | `deprecated.go:77` | same, in the deprecated root shim |
| 3 | `ignored` | `client/transport.go:583` | `_, _ = io.Copy(io.Discard, resp.Body)` blank discard |
| 4 | `ignored` | `client/transport.go:584` | `_ = resp.Body.Close()` blank discard |

## a) FULLY DONE

1. **Deep research, all claims source-verified:**
   - `go-error-modernization` skill loaded first (mandatory trigger).
   - `go-error-family` source verified: `Error.Is` matches by **code+family** (not pointer identity); `WithContextf` **clones** the receiver (sentinel-safe, never mutates).
   - `erraudit` source (`/home/lars/projects/erraudit`) verified:
     - `--no-suppress` bypasses **both** AST heuristics **and** `//nolint` suppression (`analyzer.go`: `if !cfg.NoSuppress { SuppressViolations }`) — under that flag every blank-identifier error discard is reported, by design.
     - `sentinel_concrete_type` also flags unexported `err*`-prefixed package vars (per analyzer testdata) — this killed the `errInvalidConfigTemplate` design.
     - Suppression engine honors `//nolint:erraudit // reason` (suppresses all types on the line); bare `//nolint` works too; `erraudit fix` offers no auto-fix for these findings ("advisory-only").
     - erraudit's own codebase uses exactly `//nolint:erraudit // best-effort load; ...` for this error class — the tool authors classify best-effort discards as "Idiomatic" (2026-07-26 feedback doc).
   - golangci `nolintlint` tolerance of the unknown `erraudit` name verified empirically in the erraudit repo: **warning only, 0 issues**.
2. **Sentinel fix implemented (findings 1+2 eliminated):**
   - `server/errors.go`: `var ErrInvalidConfig error = errorfamily.NewRejection(ErrCodeInvalidConfig, msgInvalidConfig)` — declared as `error` interface; new shared `msgInvalidConfig` const; new `newInvalidConfig() *errorfamily.Error` factory.
     - Rejected designs: `errInvalidConfigTemplate` var (re-triggers `sentinel_concrete_type` — `err*` prefix); non-`err`-prefixed template var (trips `errname`).
   - `server/etag.go`: both `Validate` sites now `return newInvalidConfig().WithContextf(...)` — identical wire behavior (fresh instance, same code+family, `errors.Is` still matches).
   - `deprecated.go:77`: `var ErrInvalidConfig error = etagserver.ErrInvalidConfig` (explicit interface).
3. **drainAndClose fix implemented (findings 3+4 addressed via documented suppression):**
   - Expanded doc comment explaining why both discards are deliberate (304 discarded either way; net/http closes undrained connections; worst case = lost connection reuse).
   - `//nolint:erraudit // <reason>` on both lines — house style (`//nolint:wrapcheck`, `//nolint:nilnil` precedents), honored by default-mode erraudit, audit-able via `erraudit nolint-audit`.
   - Rejected alternatives: fake consumption (empty if-block = CRITICAL `silent_swallow`; `resp.Close = true` gesture = lying code); client `OnError` hook (new public API for a near-worthless signal, contradicts the client's zero-error-invention passthrough philosophy); propagating the error (the 200 rebuild succeeded — failing the round trip would be wrong).
4. **Verification green where measured:** `go build ./...` ✅ · `go vet ./...` ✅ · `go test -race ./...` ✅ (3/3 packages) · `golangci-lint run` ✅ 0 issues (only the known benign nolintlint warning re `erraudit` name).
5. **User's exact erraudit invocation re-run:** violations **4 → 2**. Both sentinel findings gone; "Total Named Errors" 2 → 1. Remaining 2 are the deliberately-nolint'd drain lines (`client/transport.go:586-587`), reported **by design** under `--no-suppress` (audit mode surfaces everything suppressible).

## b) PARTIALLY DONE

1. ~~**erraudit default-mode run (without `--no-suppress`) never captured cleanly**~~ **done at `fc83490` follow-up:** Total Violations: 0 (all severities 0) — filtered `grep -vE '^\[feature:logger\]'`.
2. **Final user-facing summary never delivered** — the session froze mid-verification instead of closing the loop.
3. ~~Session todo list at cutoff: "Re-run erraudit to confirm zero"~~ **done at `fc83490` follow-up** — both modes captured; "Update CHANGELOG + AGENTS.md" done.

## c) NOT STARTED

All six items below were completed in the 2026-09-11 follow-up session (around `fc83490`):

1. ~~**CHANGELOG.md `[Unreleased]`** entry~~ done — Changed entry (sentinel widening, both packages) + Added entry (`ExampleETagConfig_Validate`).
2. ~~**AGENTS.md memory updates**~~ done — Error Classification (interface declaration, factory, code+family matching), client gotcha (drainAndClose rationale + `nolint-audit .` path gotcha), Commands (erraudit invocation recorded).
3. ~~**`golangci-lint fmt`**~~ done — no-op on the tree; `golangci-lint run` 0 issues after.
4. ~~**`erraudit nolint-audit`**~~ done — `nolint-audit .` reports 2 directives: 2 needed, 0 stale (note: it takes a path, not `./...`).
5. ~~**`docs/decisions/no-validation-error-interface.md`** line 14 wording drift~~ done — Context rewritten (fresh derived error, `error`-interface sentinel), two-field Validate claim fixed, references de-drifted.
6. ~~**Direct unit test for `newInvalidConfig()`**~~ done — `TestErrInvalidConfig_MatchesDerivedErrors` + GoDoc `ExampleETagConfig_Validate`.

## d) TOTALLY FUCKED UP

Nothing destructive: no reverts, no broken builds, no lost work, tree clean. The honest failure is **process**: the session stopped before the verification loop closed and before any docs landed — while the auto-daemon had already committed the half-verified state (`2ed5f8c`). The code is green everywhere it was actually measured.

## e) WHAT WE SHOULD IMPROVE

1. **Close loops before yielding** — the default-mode erraudit run was one filtered command away; instead the session stalled on truncated output. Filter erraudit noise from the start: `... | grep -vE '^\[feature:logger\]'`.
2. **Land docs at the moment of discovery** (AGENTS.md protocol) — CHANGELOG/AGENTS updates were deferred "until after verification" and then never happened; the daemon commits regardless, so deferral only risks drift.
3. **Run `golangci-lint fmt` immediately after edits** — it is in the repo command list and was skipped.
4. **`go-error-modernization` skill is stale** — its flag table claims `--no-suppress` is broken ("returns 0 even when nolint suppresses"). Verified this session: the current binary **correctly bypasses nolint** under `--no-suppress` (source + empirically — 2 violations remained with directives present). Update the skill.
5. **Encode the sentinel_concrete_type knowledge** — the skill covers only errors.As/Is migration; the discovery path this session (detector testdata, suppression grammar, nolintlint tolerance, errname interaction) cost real research and belongs in the skill or AGENTS.md for the inevitable sibling-repo sweep.

## f) Things to get done next (impact-sorted)

1. ~~Confirm erraudit **default mode** (no `--no-suppress`) reports **0 violations**~~ done — 0/0/0/0.
2. ~~Run `erraudit nolint-audit` on the 2 new directives~~ done — 2 needed, 0 stale.
3. ~~`golangci-lint fmt` + full lint re-run~~ done — 0 issues.
4. ~~CHANGELOG `[Unreleased]` Changed entry (sentinel interface widening, both packages)~~ done.
5. ~~AGENTS.md updates (Error Classification + record the erraudit command)~~ done.
6. **Owner decision:** drainAndClose posture under `--no-suppress` — accept the 2 documented findings, or add `--disable ignored` to sweep invocations, or build a client `OnError` hook (new public API). *(Follow-up default adopted + documented in AGENTS.md: accept the documented findings — still overridable by owner.)*
7. ~~Direct unit test: `errors.Is(newInvalidConfig().WithContextf(...), ErrInvalidConfig)` == true~~ done.
8. ~~Consider a compile-time pin `var _ error = ErrInvalidConfig` in tests~~ resolved as redundant — the interface declaration itself is the pin; rejected belt-and-suspenders.
9. ~~Update `docs/decisions/no-validation-error-interface.md` wording (declared type)~~ done.
10. ~~Update the `go-error-modernization` skill (stale `--no-suppress` row + sentinel learnings)~~ done for the flag row + reference doc (`--no-suppress` re-verified working on the root invocation 2026-09-11; old 2026-07-21 repro retired — its violation no longer fires on the current binary). Sentinel-learnings addition still open.
11. Sweep sibling LarsArtmann Go repos for the same concrete-typed sentinel pattern (`var ErrX = errorfamily.New...`) — same erraudit finding class almost certainly exists there.
12. ~~Old typed-errors-report open items #9 (GoDoc example) & #16 (Is across different context values)~~ done — `ExampleETagConfig_Validate` + `TestErrInvalidConfig_MatchesDerivedErrors`.
13. Decide whether erraudit joins CI as a lint job (currently ad-hoc; invocation now documented in AGENTS.md Commands).
14. v0.3.2 patch-release decision once items 1–6 land (go-release skill gates the tag). *(Items 1–6 landed; release decision still open — see (g)3.)*
15. Post-release: verify pkg.go.dev renders the new sentinel doc comment correctly.

## g) Questions I cannot answer myself

1. **drainAndClose posture:** is "2 documented deliberate ignores surfaced by `--no-suppress` audit mode" acceptable (nolint is honored in default mode), or do you want zero under `--no-suppress` too — and if so, which trade: `--disable ignored` on sweep runs, or a new client `Options.OnError` hook (public API; breaks the client's passthrough error philosophy)? *(Follow-up session adopted and documented the first option as the working default in AGENTS.md; owner can still override.)*
2. ~~**Consumer breakage check:**~~ **closed by sweep at `fc83490` follow-up** — grepped every local project under `/home/lars/projects` for go-etag importers referencing `ErrInvalidConfig`: **zero files**. No local consumer uses the sentinel at all; the CHANGELOG entry covers hypothetical external consumers.
3. **Release timing:** should the CHANGELOG entry ride `[Unreleased]` into a prompt v0.3.2 patch, or accumulate toward the next minor? *(Still open — blocking nothing; CI green on the exact tree.)*

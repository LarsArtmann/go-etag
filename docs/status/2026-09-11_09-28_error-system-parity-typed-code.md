# Status Report: Error System Parity with httputil (Typed `Code` Surface)

**Date:** 2026-09-11 09:28
**Repo:** go-etag @ master (auto-daemon commits `2f315a8`, `59d98f4` + 4 dirty doc/test files)
**Session scope:** Gap analysis go-etag ↔ httputil error systems, then execution of the improvements.

> Format note: this report is Markdown per the user's explicit request (`<...>.md`), overriding the status-report skill's HTML default.

---

## a) FULLY DONE

| Item | Evidence |
|---|---|
| Gap analysis of both error systems (5 go-etag codes vs ~68 httputil codes; typed vs untyped; routing helpers; template model; stdlib classifications) | Session research, verified via `go doc` + source reads |
| `server/code.go`: typed `Code` + `Domain`, `Domain()`, 6 family constructors + 6 `Wrap*` methods (exact httputil parity), `DomainOf`, `InDomain` via `errors.AsType[errorfamily.Coded]` | `server/code.go` |
| Typed internal constants `codeETagWriteFailed` … `codeHashWriteFailed` mirroring the exported untyped strings | `server/errors.go:35-44` |
| Sentinel refactor: `ErrInvalidConfig error = codeInvalidConfig.Rejection(...)` — still interface-typed (erraudit sentinel guard), still matched by code+family | `server/errors.go:66` |
| All **6** construction sites refactored to `Code` constructors (2 in `wrapper.go`, 4 in `etag.go`; earlier summary said 5 — miscount) | `wrapper.go:56-70`, `etag.go:119-121,225,238,257` |
| Declarative `errorTemplates` map (verbatim wording preserved — httputil mirrors these templates) + loop registration; imperative helpers deleted | `server/errors.go:76-124` |
| New `server/code_test.go`: 6-family constructor table, 6 Wrap methods (cause preservation via `errors.Is`), `Domain` table, `DomainOf`/`InDomain` happy+unhappy paths | `server/code_test.go` |
| Bidirectional template-completeness test (every code const ↔ map entry, both directions) | `errors_test.go:45-101` |
| Docs: AGENTS.md architecture row for `code.go` + internal-construction/design-decision notes; FEATURES.md error row; CHANGELOG `[Unreleased]` entry | all three files |
| Research findings locked in: client `Options` **clamp by design** (not a validation gap); shim test is assertion-based (new symbols safe); `nolintlint` defaults active (empirically verified the map's nolint is used) | AGENTS.md:158-160 |
| Full gate green: `go build`, `go vet`, `go test -race -count=1 ./...` (3× ok), `golangci-lint fmt`, `golangci-lint run` → **0 issues**, erraudit → **0 actionable findings** | session terminal output |

## b) PARTIALLY DONE

| Item | What's missing |
|---|---|
| Doc hygiene | `TODO_LIST.md` not harvested from section (f); dprint markdown formatting not run on the edited `.md` files |
| LSP hygiene | Stale gopls/golangci-ls diagnostics (`indexByte`, err113, golines) kept firing all session; verified stale via real CLI runs each time (AGENTS rule followed), but never `lsp_restart`ed to silence the noise |
| Performance claim | "No perf impact (compile-time constants, error paths only)" is reasoned but not benchmark-backed; no smoke `-bench=.` run, no `reports/bench/` baseline |

## c) NOT STARTED

- HARVEST of the 50 next-items into `TODO_LIST.md` / `ROADMAP.md` (docs-health)
- GoDoc examples for the new exported surface (`ExampleCode_*`, `ExampleDomainOf` — `testableexamples` requires `// Output:`; httputil has none either, so no parity pressure)
- Benchmark baselines per AGENTS discipline (`-benchmem -count=6` before/after — arguably N/A, unproven)
- CI verification on the daemon's commits (local gates only; tag-time rule "CI green on exact commit" untouched)
- Release decision (nothing tagged; `CHANGELOG [Unreleased]` carries the entry)

## d) TOTALLY FUCKED UP

**Nothing.** Four near-misses, all caught and fixed within the session:

1. First draft of `code.go` used a hallucinated `indexByte` helper instead of `strings.IndexByte` — caught by diagnostics, fixed, verified with real `go build`.
2. Test asserted `Error() == "msg"`; actual rendering includes the `[family:code]` prefix — caught by running tests, switched to `strings.Contains`.
3. Attempted doc edits without View-tool reads first (harness contract violation) — one wasted round trip, then done correctly.
4. Final summary miscounted call sites ("5", actually 6) — corrected in this report.

No production code, tests, or docs carry any of these forward. The persistent stale LSP warnings are noise, not truth (twice contradicted by clean `golangci-lint run` + green build).

## e) WHAT WE SHOULD IMPROVE

1. **Verify-before-claiming on perf**: run even a smoke benchmark when touching shared code, rather than asserting N/A from reasoning.
2. **Single-source the code list**: the completeness test re-lists all codes as a literal slice; a `allETagErrorCodes` var (httputil's `allHTTputilErrorCodes` pattern) would remove the duplicate-literal drift risk.
3. **Restart the LSP when diagnostics contradict the CLI** instead of tolerating repeated stale noise.
4. **Read-before-edit discipline for harness contracts**: the View-first rule applies even when content is "known" from context.
5. **Decide YAGNI vs parity explicitly in-repo**: all-6-family constructors exist for httputil parity while only 4 families are used — the rationale lives in AGENTS.md now, but a roadmap note (or trim) should settle it permanently.
6. **Commit cadence**: the auto-daemon captured source mid-session with heuristic messages; doc fixes sat dirty for a while — fine here, but the consumer-repos recipe ("commit after the gate") exists for exactly this.

## f) 50 THINGS TO GET DONE NEXT (brainstorm, sorted roughly by impact; most are ROADMAP fuel, not commitments)

**Error system follow-ups**

1. HARVEST this report into `TODO_LIST.md` (docs-health HARVEST mode)
2. Add `allETagErrorCodes` single-source var for the completeness test
3. GoDoc examples for `Code` constructors / `DomainOf` / `InDomain` with `// Output:`
4. Assert `writer_type` context survives on hijack errors (Contextual smoke test)
5. README: document typed `Code`/`Domain` error routing for consumers
6. Decide: exported `Domain("http")` constant vs leave literal (YAGNI call)
7. Decide: keep all-6 families or trim to the 4 go-etag uses (parity vs YAGNI)
8. Verify httputil's test suite pins go-etag template VALUES verbatim; if not, upstream a mirror test (verify-before-filing first)
9. Run erraudit `--no-suppress` audit mode (surfaces the documented `drainAndClose` suppressions) and `erraudit nolint-audit .`
10. Go 1.26 `errors.AsType` adoption sweep across the repo (go-error-modernization)

**Client package**

11. Verify the `KeyFunc` error contract end-to-end (never read its full signature/flow this session)
12. Document clamp-don't-reject explicitly in `client/doc.go` (today only on the fields)
13. Design note: optional `Options.Validate()`-style loud-validation hook vs silent clamping
14. Fuzz `mergeHeader` (mirrors existing `FuzzHasNoStoreDirective` pattern)
15. Confirm CI race job covers the new `code_test.go` paths (`-race ./...` parity)
16. Document why `Stats()` has no error surface (nothing to classify — make it explicit)

**Server / quality**

17. Smoke benchmark `-bench=.` to back the no-perf-impact claim
18. Capture `-benchmem -count=6` baselines under `reports/bench/` per discipline
19. godoclint/pkg.go.dev rendering check of new doc comments post-publish
20. Record in `deprecated_test.go` docs that the typed surface is deliberately NOT shimmed
21. gosec sanity note: `Code`/`Domain` accept arbitrary strings; confirm no template-injection surface in errorfamily rendering
22. Check CI pins the same golangci-lint version used locally (version drift)
23. Run dprint on the edited Markdown files
24. docs-health VERIFY pass on the updated FEATURES.md claims
25. ANNOTATE older `docs/status/` reports that describe the error system as untyped-only

**Docs / memory**

26. Add `Code`/`Domain`/`Family` terms to `docs/DOMAIN_LANGUAGE.md` if error taxonomy is in scope there
27. ROADMAP.md: per-component domain-constants idea across larsartmann libraries
28. ROADMAP.md: registry introspection idea for go-error-family (`ListTemplates`) so completeness tests need no local mirror map
29. Cross-link AGENTS.md error sections of go-etag ↔ httputil (parity is now bidirectional)
30. Check `TODO_LIST.md` for pre-existing error items this work silently resolved (reverse harvest)
31. CHANGELOG: state "additive API, no migration required" explicitly for consumers
32. Post-release: verify pkg.go.dev renders the new API (eventual, not a gate)

**Cross-repo / ecosystem**

33. Consolidation review: could httputil reuse go-etag's `Code` type (it already imports `go-etag/server`) instead of owning a duplicate?
34. Ecosystem sweep: same untyped-code gap in other larsartmann libs (go-output, samber-do-auditlog, …)
35. DiscordSync drift-guard: check whether it pins go-etag's error surface before the next release (consumer-migration recipe)
36. Run `nix build` once with the new files (go.sum unchanged → should pass; verify)
37. Verify workspace + hermetic (`GOWORK=off`) modes both build
38. go-error-family: feature request candidate — template placeholder validation helpers

**Release / process**

39. Release decision: tag v0.3.2 with this additive change after CI green (user call)
40. Post-tag: clean-room `go get` + proxy/sum verification per release conventions
41. Post-tag: GitHub Release as Latest non-prerelease (house precedent)
42. Release commit style: `chore(release): cut CHANGELOG v0.3.2`, changelog cut before tag
43. Confirm the auto-daemon's heuristic commits don't violate the repo's linear-history expectations before tagging
44. Consider teaching the daemon the release-commit style (or accept heuristic noise in history)

**Smaller polish**

45. Add `GoString`/`String` consideration for `Code`/`Domain` (currently bare string aliases behave fine; decide if anything is needed)
46. Table-driven test for `Domain()` on multi-dot codes (`a.b.c` → `a`) — trivial addition
47. Assert `newInvalidConfig()` freshness (two calls return non-identical instances) in errors_test
48. Consider `exhaustruct_v5`-proofing note: `MessageTemplate` literals must set all 4 fields (currently lint-enforced; nothing to do, just awareness)
49. Sweep for other "5 call sites"-style miscounts in recent docs (self-audit of this report's claims)
50. Retrospective input: add "perf claims need a benchmark run" to global engineering memory if validated as a recurring miss

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Shim scope:** should the deprecated root shim receive the new typed surface (`Code`/`Domain`/`DomainOf`/`InDomain`) at v0.3.2, or stay frozen at the pinned v0.1.x surface until its v1.0.0 deletion? (I froze it; that's a product/compat call, not a technical one.)
2. **Release:** cut v0.3.2 now with this additive API, or batch with other pending work into a later release?
3. **YAGNI vs house parity:** keep all six family constructors (exact httputil mirror, two families unused) or trim to the four go-etag actually uses?

---

**WAITING FOR INSTRUCTIONS.**

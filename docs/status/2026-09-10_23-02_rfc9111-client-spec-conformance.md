# Status Report — RFC 9111 Client Spec Conformance (Session 2026-09-10)

**Scope:** Self-contained report on this session's run: Alex's field report (DEV.to stale-200/`Age: 137882`), the RFC review, the client-transport spec fixes, the spec test suite, and the honest gaps that remain. No research beyond what this session touched.

**Trigger:** Email from Alex (spinov001@gmail.com, 2026-09-09): "the 200 that should have been a 304" — a CDN served a 200 with a two-day-stale body whose ETag described the stale entity faithfully. Ask: review, check the specs, add tests based on the real spec.

**Verification state at time of report:** `go vet ./...` clean · `go test -race -count=1 ./...` all pass · `golangci-lint run` 0 issues · `golangci-lint fmt` clean · client coverage 89.6% · benchmarks smoke-run (Fresh200 ~8.4µs, 304Rebuild ~3.2µs, 10 iterations only). Working tree: 8 modified + 1 new file, uncommitted (auto-git daemon will sweep).

---

## a) FULLY DONE

1. **Primary-source spec verification** — downloaded RFC 9111 + RFC 7232 from rfc-editor.org and read the actual text of §3, §3.1, §3.2, §4.3.4, §4.3.5, §4.4, §5.1. No spec claims were made from memory. Key quote driving the work: RFC 9111 §5.1 — _"The presence of an Age header field implies that the response was not generated or validated by the origin server for this request."_ That is Alex's field case in one sentence.
2. **RFC 9111 §4.3.4 + §3.2 freshening as the default** — a 304 now replaces every stored header field it provides (previously only `Date` merged). Exceptions per spec: hop-by-hop (§3.1), `Content-Length`/`Content-Range` (§3.2), and `Content-Encoding` when net/http transparently decoded the body (§3.2 integrity allowance, recorded via `resp.Uncompressed`). `client/transport.go` `rebuiltHeader`/`freshenedHeader`/`skippedByFreshening`.
3. **Freshening persists to the stored entry** (§4.3.4 "the cache MUST update its header fields" of the _stored_ response) — `persistFreshened` + `responseCache.freshen` (concurrent-replace guard: only freshens when the key still maps to the validated response). Later revalidations carry the newest validator; `Stats.Stored` semantics unchanged (freshening is not storing).
4. **`Cache-Control: no-store` responses are never stored** (§3 MUST NOT) — quote-aware, case-insensitive directive parsing (`cacheControlDirectives` + `hasNoStoreDirective`); a quoted `"no-store"` argument inside another directive does not match. Bodies still stream through intact.
5. **Caller-supplied `If-None-Match` is never clobbered** (RFC 9110 §13.1.2 ownership) — the transport injects its stored validator only when the caller's conditional slot is empty; a 304 answering the caller's own conditional passes through unrebuilt.
6. **Hop-by-hop fields stripped at store time** (§3.1) — `Connection`-listed fields, `Keep-Alive`, `Proxy-*` never enter the store, so rebuilds cannot resurrect connection-scope state.
7. **Spec test suite** — `client/spec_test.go`, 10 tests / 13 assertions groups, every test citing its RFC section: Age surfaced verbatim on 200s (the DEV.to case with `Age: 137882`), Age-not-running-backwards across revalidation, full freshening (Age/Cache-Control/Expires/Date/X-Cache), the §3.1/§3.2 exceptions (incl. decoded-body Content-Encoding both directions), stored-validator persistence (weak→strong CDN normalization), no-store ban (+ quoted-arg + case-insensitivity subtests), no-cache-serves-via-revalidation (§5.2.2.4), HEAD bypass (§4.3.5 opt-out), caller-owned conditionals, stale-entry replacement (field-case happy ending), restricted `PreserveOn304` list semantics, hop-by-hop shedding.
8. **Docs updated** — `client/doc.go` (new "Stale edge caches and Age" section answering Alex's question publicly), README (bullets, options table, example comment), CHANGELOG `[Unreleased]` Added/Changed/Fixed, AGENTS.md (architecture rows, API design, client gotchas +5, testing conventions). Historical CHANGELOG 0.2.0 entry and `docs/planning/` intentionally left untouched (point-in-time records).
9. **Answer to Alex's question delivered in-chat**: Age was surfaced on 200s but stale-by-construction on rebuilds before this session; now 304s freshen it. The cache _decision_ remains internal (the transport always revalidates before serving a stored body — it is a conditional-GET accelerator, not a freshness cache).

## b) PARTIALLY DONE

1. ~~**RFC 9111 conformance** — §3, §3.1, §3.2, §4.3.4, §5.1 handled; §4.4 (unsafe-method invalidation, a MUST) and §4.3.5 (HEAD freshening, a SHOULD) knowingly not implemented — and worse, **§4.4 was in the extracted RFC text I read and I skipped it silently**.~~ §4.4 landed at `9204885`; §4.3.5 remains a documented opt-out (TODO_LIST #5). The AGENTS.md known-gap note is no longer needed for §4.4.
2. ~~**Repo hygiene** — `dprint.json` exists and I edited three Markdown files **without running dprint**. `CONTRIBUTING.md` exists and I never opened it (Project Discovery checklist violation).~~ done (block two: CONTRIBUTING read — nothing to reconcile; dprint investigated — binary absent locally and in CI, CHANGELOG excluded by config, skipped deliberately)
3. ~~**IDE diagnostics** — the LSP still shows 11 stale warnings (bodyclose at pre-refactor lines, `freshen` "unused") that fresh CLI runs contradict. Verified stale, never cleared via `lsp_restart`. The next session will see phantom errors.~~ resolved (moot — `golangci-lint run` re-verified 0 issues on 2026-09-10)
4. ~~**Validator-update spec tension (found during this self-review, not during the work)** — RFC 9111 §4.3.4's filtering says a 304 carrying a _strong_ validator only updates stored responses with that _same strong_ validator; my weak→strong validator persistence test encodes behavior that is practical (it is what the single-entry model wants) but arguably beyond what §4.3.4's update filter permits. Needs a deliberate decision: restrict validator replacement to matching tags, or document the deviation.~~ done at `9204885` — the strict reading won: `restoreMismatchedValidator` filters by weak match (RFC 9110 §8.8.3.2), and weak-form adoption (`W/"v1` answering stored `"v1`) is pinned by `TestSpecWeakFormOfStoredValidatorIsAdopted`
5. **Alex's email** — answered in-chat to the user; **no draft reply email exists** and his offer of raw header-capture fixtures was never taken up.

## c) NOT STARTED

_Open items route to TODO_LIST.md / ROADMAP.md — 2026-09-10 docs-health harvest._

- ~~§4.4 unsafe-method invalidation (see above).~~ done at `9204885`
- §4.3.5 HEAD-based freshening/invalidation.
- ~~`Vary` awareness in cache selection or even a documented sharp-edge note (today only the credential warning exists; `Vary: Accept-Encoding` across a shared key is the classic trap).~~ done at `9204885` — §Vary sharp-edge section in `client/doc.go` + AGENTS.md gotcha (KeyFunc is the mitigation)
- ~~Integration test against a real `httptest.Server` + real `http.Client` (all current tests are stub-based; canonical-form and hop-by-hop claims rest on knowledge, not an executed round trip).~~ done at `9204885` — `client/integration_test.go`
- Version/release decision (v0.3.0 vs holding) — the `PreserveOn304` default change is a behavior change sitting in `[Unreleased]`.
- ~~`TODO_LIST.md` / `ROADMAP.md` — the repo has neither; harvesting this report's section (f) is pending instruction.~~ done (2026-09-10 docs-health pass — both files created, this report harvested)
- Reply email to Alex; fixture ingestion (`testdata/`).

## d) TOTALLY FUCKED UP (all caught in-session, none shipped)

1. **Three edit-tool failures from sloppiness in one session**: the same struct-literal typo (`etag:` written as `ag:`) submitted twice in one batch, then repeated in the retry's `old_string`; one `old_string` that didn't match the file. Rule says never retry with guessed text — I repeated a typo instead of re-reading. Slow, avoidable.
2. **A junk line shipped into the first draft of `spec_test.go`** (`var _ io.Reader = (*strings.Reader)(nil)`) to silence an unused-import I should have just removed.
3. **The HEAD test initially asserted `method == HEAD` on a stub that also serves the priming GET** — first run failed on exactly that. Wrote the assertion without tracing the request flow.
4. **A varnamelen-driven "fix" that was itself a lint violation** (a `mu := func() ...` closure) — had to be fixed twice.
5. **Honesty correction**: my closing summary said "fixed four spec violations". §4.4 is also violated and I missed it — the true count of _known_ violations was five before this session's fixes, and one (§4.4) remains. Also "pinned everything" overstated the validator-persistence test, whose spec basis is arguable (see b.4).

## e) WHAT WE SHOULD IMPROVE

- **Read the whole relevant section chain before cutting scope.** I extracted §4.4 text and still skipped it. A one-line "known gaps vs RFC 9111" table in AGENTS.md would have forced the admission into the open.
- **Share the entity-tag domain type.** The module has a proper `ETag` type (parser, strength, weak/strong comparison) in `server/` while `client/` treats validators as raw strings. The §4.3.4 matching nuance (b.4) is _exactly_ the bug class a typed validator prevents. This is a small pre-existing split brain that this session's string-validator logic widened.
- **`PreserveOn304 []string` nil-vs-empty overload is subtle API design** — nil means "spec default (everything)", empty means "nothing", non-empty means "only these". A typed mode (`FreshenPerRFC` / `FreshenFields(...)` / `FreshenNone`) would make misuse unrepresentable.
- **Rebuilt responses don't set `Uncompressed`** even when the stored body is the transparently decoded form — one honest field we skip.
- **Marked Age survives a rebuild when the 304 omits Age** (we never delete metadata we cannot recompute). Defensible, but undocumented as a limitation.
- **Verification depth**: coverage percentage was checked, uncovered _branches_ were not (the `\\` escape branch of `cacheControlDirectives`, `freshen`'s skip-guard, `mergeHeader`'s canonical fallback are likely uncovered). Benchmarks were smoke-run at 10 iterations with no baseline comparison.
- **Commit discipline**: session work sits uncommitted awaiting the auto-daemon; fine per house rules, but the report-only commit below is the only deliberate one.

## f) UP TO 50 THINGS NEXT

_Spec conformance (high impact):_

1. ~~Implement RFC 9111 §4.4: invalidate the stored entry when a non-error response answers an unsafe method (PUT/POST/DELETE/PATCH) for the same URI.~~ done at `9204885`
2. ~~Test §4.4: PUT 200 → next GET must not rebuild from the pre-PUT entry.~~ done at `9204885` (`TestSpecUnsafeMethodInvalidatesEntry` + the URI-scoping test)
3. Implement/test §4.3.5 HEAD freshening: HEAD 200 with matching ETag (and Content-Length) freshens stored metadata; mismatch marks stale (drop entry).
4. ~~Resolve the §4.3.4 strong-validator filter question (b.4): restrict stored-validator replacement to matching tags, or document the deviation as deliberate.~~ done at `9204885` — `restoreMismatchedValidator` filters by weak match; weak-form adoption pinned by test
5. ~~Set `Uncompressed: entry.uncompressed` on synthesized responses.~~ done at `9204885`
6. ~~Document (or implement) `Vary` handling; minimum: a sharp-edge paragraph beside the credential warning.~~ done at `9204885` — §Vary section in `client/doc.go` + AGENTS.md gotcha
7. Document the "stored Age survives when the 304 omits Age" limitation; decide whether to drop it instead.
8. Add a MUST-by-MUST RFC 9111 conformance table to the docs (done/deviation/not-applicable).
9. Pin "only 200s are ever stored" with a test (206/304/500 never stored).
10. Decide `no-cache="field"` argument semantics (currently treated as plain no-cache for storage purposes — legal, but say so).

_Testing (high impact):_
11. ~~Integration test: `httptest.NewServer` + real `http.Client{Transport: NewTransport(...)}` — canonical `Etag` form, real 304 (no body), real `Age` flow, gzip round trip.~~ done at `9204885` — `client/integration_test.go`
12. ~~Cover the `cacheControlDirectives` escaped-quote branch (`private="a\"b,no-store"` must not match).~~ done at `9204885`
13. Cover `responseCache.freshen`'s skip-guard (freshen vs concurrent replace).
14. Race test: parallel rebuilds + stores on one key; assert `hits+stored` invariant and single `Entries`.
15. Fuzz `hasNoStoreDirective`/`cacheControlDirectives` (it is a parser; the server side already fuzzes parsers).
16. Fuzz `freshenedHeader` for map-aliasing surprises.
17. Uncovered-branch audit; push client coverage 89.6% → 95%+.
18. Test additive freshening: 304 provides fields the stored response never had (§3.2 "add each").
19. Test FromCacheHeader when the 304 itself echoes the marker name.
20. Property test: Age monotonicity across N revalidations under a growing CDN Age.
21. Test that a 304 with `Cache-Control: no-store` still rebuilds (it validates an already-stored response; §5.2.2.5 speaks to storing, not reuse-after-validation) — confirm and pin whichever reading we adopt.

_Architecture / types:_
22. Extract or share the `ETag` domain type into the client (typed validators, §4.3.4-aware matching).
23. Replace `PreserveOn304 []string` with a closed mode type.
24. Evolve `cacheEntry` toward a `storedResponse` type with validator-matching methods.
25. Consider request coalescing (singleflight) for concurrent identical GETs.
26. Consider observability hooks (OnHit/OnStore/OnFreshen) mirroring the server's.
27. Revisit `MaxBodyBytes` counting decoded bytes on transparent gzip (document or fix).

_Docs / process:_
28. ~~Run `dprint` over the Markdown I touched (README, CHANGELOG, AGENTS.md).~~ done (block two: binary absent locally and in CI, CHANGELOG excluded by `dprint.json` — deliberately skipped)
29. ~~Read `CONTRIBUTING.md`; reconcile anything I violated.~~ done (block two: nothing beyond the commands already followed)
30. Draft the reply email to Alex; decide on his fixture offer and where fixtures live (`client/testdata/`).
31. ~~Create `TODO_LIST.md` + `ROADMAP.md`; harvest section (f).~~ done (2026-09-10 docs-health pass)
32. ~~Update README's "98.9% test coverage" claim (now inaccurate as a number and unversioned).~~ done (prose fixed at `9204885`; hardcoded coverage badge removed 2026-09-10 docs-health pass)
33. ~~AGENTS.md: add "§4.4 not implemented" as an explicit KNOWN GAP.~~ superseded — §4.4 implemented at `9204885`; the §4.3.5 opt-out is documented instead
34. ~~`lsp_restart` to clear the 11 stale diagnostics.~~ resolved (moot — fresh session diagnostics clean; lint 0 issues re-verified 2026-09-10)
35. Upgrade `exhaustruct` → `exhaustruct_v5` (deprecation warning appeared during lint).
36. GoDoc example: an Age-aware client that rejects stale-at-the-edge responses.

_Release:_
37. Decide v0.3.0 vs holding for §4.4 + HEAD freshening; then run the release skill (CHANGELOG cut, tag, proxy verification).
38. Re-verify CHANGELOG wording against Keep-a-Changelog conventions.

_Roadmap-scale ideas:_
39. True freshness-based serving (§4.2: max-age/Expires parsing; serve stored without revalidating within freshness lifetime).
40. `stale-while-revalidate` / `stale-if-error`.
41. `Last-Modified` / `If-Modified-Since` second validator.
42. Opt-in disk persistence of the cache.
43. Per-host key partitioning by default (cross-origin KeyFunc safety).
44. OTEL/Prometheus example wiring.
45. Project website launch (sibling-repo pattern).
46. ~~Clarify the role of `reports/` vs `docs/status/` and unify.~~ **Won't implement — no `reports/` directory exists; `docs/status/` is the single snapshot location (ROADMAP non-goal).**

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **Release timing:** cut **v0.3.0 now** with the `PreserveOn304` behavior change in `[Unreleased]`, or hold until §4.4 invalidation (+ HEAD freshening) land so "RFC 9111 conformance" ships as one coherent minor?
   _**Status (2026-09-10):** §4.4 landed; §4.3.5 is a documented opt-out. Release still awaits owner GO — TODO_LIST.md #1._
2. **Domain-type direction:** may the client import `server`'s `ETag` type (new intra-module coupling), or should the type move to a shared subpackage first? This shapes items 22–24 and is a public-surface decision.
   _**Status (2026-09-10):** moved to ROADMAP.md Open Questions._
3. **Alex:** do you want a draft reply email (I'd fold in the Age answer + the no-store/freshening changes), and should we accept his raw header captures as fixtures — if yes, `client/testdata/` is my proposed home?
   _**Status (2026-09-10):** moved to ROADMAP.md Open Questions._

---

_Point-in-time snapshot. Written 2026-09-10 23:02. Format: Markdown per explicit user instruction (overriding the status-report skill's HTML default); the brutal-self-review questions are folded into sections a/e rather than emitted as a separate `docs/reviews/` HTML report, per the same single-file instruction._

# Roadmap

Long-term direction and raw ideas — not actionable tasks (those live in
`TODO_LIST.md`). Status indicators are deliberately absent: this is vision,
not inventory.

## Themes

### 1. Complete the RFC 9111 client story

The client transport is a conditional-GET accelerator: it always revalidates
before serving a stored body. The remaining spec surface, in rough order of
value:

- True freshness-based serving (§4.2): parse `max-age`/`Expires`, serve stored
  responses without revalidating inside the freshness lifetime. This changes
  what the library _is_ (from accelerator to cache) — needs a deliberate
  opt-in design so the default stays honest.
- `stale-while-revalidate` / `stale-if-error` extensions on top of §4.2.
- `Last-Modified` / `If-Modified-Since` as a second validator type alongside
  ETags (servers that send no ETag today get zero benefit from the client).

### 2. Typed client cache — COMPLETE (2026-09-18)

Delivered: the shared [`entitytag/`](../entitytag/README.md) package (moved
out of `server/`), typed validator comparison in the client, and the
`cacheEntry` → `storedResponse` evolution (`storedValidator`: the wire string
replayed verbatim plus the parsed `entitytag.ETag`, parsed once at store time
— commit `07fe65c`). The §4.3.4 validator-filtering bug class a typed
validator prevents is closed; no further work planned here.

### 3. Scale, persistence, and observability

- Server-side observability is complete — the `metrics/` package (moved in
  from `httputil/etagmetrics`, 2026-09-22) ships ready-made counters for the
  server hooks; no telemetry SDK dependency enters the module unless a
  concrete consumer asks for an exporter sub-module.
- Client observability hooks (`OnHit` / `OnStore` / `OnFreshen` /
  `OnInvalidate`) mirroring the server's — `Stats()` suffices until a consumer
  asks.
- Per-host key partitioning by default (cross-origin KeyFunc safety).
- Opt-in disk persistence of the cache.
- Request coalescing (singleflight) for concurrent identical GETs.

### 4. Public presence

- Project website launch (sibling-repo Astro/Starlight pattern).
- Comparison table vs other Go ETag/caching libraries; awesome-go listing.
- OTEL/Prometheus example wiring recipe on top of the existing hooks.

## Parked (demand-gated)

- **`go-etag/otel` sub-module** — automatic OpenTelemetry wiring. The core
  hooks (`OnETagGenerated`, `On304`, `OnBufferOverflow`) shipped in v0.2.0;
  manual wiring is ~10 lines, so the sub-module is convenience, not
  capability. Unpark when hooks are battle-tested AND a real consumer asks.
  The 2026-09-23 module split removed the last architectural blocker — a
  sub-module no longer pollutes core consumers — so the gate is purely
  demand. Full shape and prereqs: `docs/review-and-roadmap.md` §Parked.
- **Root shim deletion** — scheduled for the v1.0.0 tag (not before).
- **Per-component domain constants across larsartmann libraries** — every
  library that classifies errors declares its own `Domain("…")` strings;
  a shared convention (or package) would keep component domains consistent
  across go-etag, httputil, go-output, and friends. Only pays off once a
  second library actually routes by domain. Origin: report
  `2026-09-11_09-28_error-system-parity-typed-code.md` f#27.
- **go-error-family registry introspection (`ListTemplates`)** — a query
  surface on the classification registry would let completeness tests
  verify codes against the registry instead of mirroring a local literal
  slice. Upstream feature request candidate; blocks on go-error-family
  wanting the API. Origin: report
  `2026-09-11_09-28_error-system-parity-typed-code.md` f#28, f#38.
- **Data-model accepted-as-is items** — the 2026-09-23 data-model review
  left three shapes deliberately unchanged, revisit only at the v1.0.0
  design pass: `Stats`' int64/int mix, `Counters`' exported atomic fields,
  and the responseWrapper 2-bool+int state encoding. Full verdicts:
  `docs/reviews/2026-09-23_15-22_brutal-self-review.html`.

## Non-goals

- **Server-side If-Match middleware interception / automatic 412** — ETag
  generation ≠ conditional-request evaluation for unsafe methods;
  applications own preconditions via `MatchesIfMatch`. Deliberate scope
  boundary since v0.1.0.
- **Full RFC 7232 §6 precedence chain** on the server — same reasoning.
- **If-Range / Range support (RFC 7233)** — different problem domain.
- **Vary-aware secondary cache keys** — deliberate scope boundary (recorded
  2026-09-23): `client/doc.go` §Vary documents the sharp edge and `KeyFunc`
  is the mitigation; revisit only on a real consumer demand signal.
- **A telemetry dependency in core** — hooks only; backends wire themselves.
- **`reports/` directory** — status snapshots live in `docs/status/` only;
  no second location.

## Open questions

1. ~~**Domain-type direction (blocks Theme 2):** may the client import
   `server`'s `ETag` type (new intra-module coupling), or should the type
   move to a shared subpackage first?~~ Resolved 2026-09-18: the type moved
   to the shared `entitytag/` subpackage; `server/` re-exports the full
   surface via aliases + wrappers, and the client compares validators through
   `entitytag.ParseETag`/`WeakEqual` (no `client → server` edge). Theme 2's
   remaining piece is the `cacheEntry` → `storedResponse` evolution. The
   story continued: since the 2026-09-23 module split, `entitytag/` is its
   own module (zero deps) — both server and client now depend on it as an
   external, version-pinned module (v0.6.0).
2. ~~**Alex's field-report offer:** draft a reply email (Age answer +
   no-store/freshening changes are in), and accept his raw header captures
   as permanent fixtures in `client/testdata/`?~~ Resolved 2026-09-23: the
   owner sent the reply; fixture acceptance waits until the captures actually
   arrive (reopen as a TODO_LIST item then).
3. ~~**Release workflow for go-etag:** decide whether to add one or keep
   manual releases.~~ Resolved 2026-09-23 (owner: automate): `.github/workflows/release.yml`
   triggers on the ROOT `v*` tag, extracts the matching CHANGELOG section as
   the notes (fails loudly if the section was not cut), and creates the
   Release as Latest; idempotent on re-runs. Nested module tags deliberately
   do not get Release pages — the proxy is their consumer interface.
4. ~~**FNV-64a strength honesty (standing decision, revisitable).**~~
   Closed 2026-09-23 (owner defers to the documented position): the default
   `Strong` stays; the README and `entitytag` docs keep the honesty note
   (birthday bound ~4.3B bodies); changing the default would be breaking.
5. ~~**Residual open-low backlog:** promote the ~25 annotated-open "low"
   items, or let them die with the archives?~~ Resolved 2026-09-23 (owner
   delegated): they die with the archives. Rationale: open since August with
   zero demand; the RFC conformance table and coverage are strong; promoting
   25 low items would bury the high-signal backlog. Revivable — any item can
   re-enter via TODO_LIST on demand.
6. ~~**Deprecated shim scope:** typed surface, or frozen at the pinned v0.1.x
   surface until deletion?~~ Resolved 2026-09-23 (owner delegated; informed
   call): FROZEN. The shim is deleted at v1.0.0; typed-surface parity in a
   tombstone adds a re-export + parity-guard tax on every new export for
   consumers who should migrate to `…/server` anyway. The deprecation markers
   provide the migration path.
7. ~~**YAGNI vs house parity:** keep all six error-family constructors or
   trim to the four actually used?~~ Resolved 2026-09-23 (owner: consider
   go-error-family; otherwise 1 possible failure = 1 distinct error): KEEP
   all six — they are the shared vocabulary of the httputil mirror contract
   (errorTemplates pinned verbatim across repos), not per-error types; the
   1-failure-1-distinct-error principle already holds via the five `http.etag_*`/
   `http.hijack_*` codes. The exported `Domain("http")` constant stays
   literal (same parity reasoning).
8. ~~**art-dupl enforcement policy:** machine-enforce the accepted clone
   baseline, or prose-only?~~ Resolved 2026-09-23 (owner: enforce): the
   release gate now runs `art-dupl -t 1 --type-aware` and FAILS unless
   exactly the 4 accepted groups are shown (`scripts/pre-release-check.sh`,
   baseline `reports/dupl/2026-09-23_art-dupl-baseline.txt`; AGENTS documents
   each group's accept rationale). Extraction or deliberate re-baseline
   required on drift.
9. ~~**Validate() vs New() semantics** (from the 15:35 self-review's split
   brain #1): Validate rejects `MaxBufferSize <= 0` and out-of-range
   `Strength`; New clamps/accepts.~~ Resolved 2026-09-23 (owner: "most
   correct" delegated): keep the divergence — validation and defensive
   defaults are different jobs (reject at startup when the config comes from
   humans; never crash a server for a zero-value config). The divergence and
   when to call Validate are documented on `ETagConfig.Validate` (shipped
   2026-09-23); the server↔entitytag surface is parity-guarded.
10. ~~**Shim parity scope** (self-review split brain #2 follow-up): who
    guards the re-export chain?~~ Closed 2026-09-23: the chain is fully
    guarded — root↔server by `deprecated_test.go`, server↔entitytag by
    `server/entity_parity_test.go` (go/ast, stdlib-only). No further scope.
11. **Release cadence for proofing `release.yml`** (from report 22:07 g3,
    open): cut a trivial v0.6.1 soon to exercise the release workflow's
    first live run, or wait for real content (M16/M17/M19) and leave the
    workflow unproven until then? The workflow is locally verified only.
12. **go-github-kit's nix lane pattern** (from report 22:07 g2, open): their
    flake is plain `pkgs.buildGoModule` and the floor bump broke the hermetic
    build (FOD wants go >= 1.27.1). House pattern preference: pkgs-level go
    override, or migrating kit onto the go-standard flake module like
    library-policy (`goPkgAttr`)?
13. **cqrs-htmx's red lint** (from report 22:07 g1, open): their master is
    red on pre-existing dashboardui lint debt (golines/mnd/prealloc) across
    six-plus commits. Owner's lane or ours to drive green in their repo?

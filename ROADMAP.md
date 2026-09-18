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

### 2. Typed client cache

The server package owns a proper `ETag` domain type (parser, strength,
comparison); the client treats validators as raw strings. The §4.3.4
validator-filtering nuance is exactly the bug class a typed validator
prevents. Directions (decision open — see Open Questions):

- Client imports `server`'s `ETag` type, or the type moves to a shared
  subpackage.
- `cacheEntry` evolves toward a `storedResponse` type with §4.3.4-aware
  validator-matching methods.

### 3. Scale, persistence, and observability

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
  Full shape and prereqs: `docs/review-and-roadmap.md` §Parked.
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

## Non-goals

- **Server-side If-Match middleware interception / automatic 412** — ETag
  generation ≠ conditional-request evaluation for unsafe methods;
  applications own preconditions via `MatchesIfMatch`. Deliberate scope
  boundary since v0.1.0.
- **Full RFC 7232 §6 precedence chain** on the server — same reasoning.
- **If-Range / Range support (RFC 7233)** — different problem domain.
- **A telemetry dependency in core** — hooks only; backends wire themselves.
- **`reports/` directory** — status snapshots live in `docs/status/` only;
  no second location.

## Open questions

1. **Domain-type direction (blocks Theme 2):** may the client import
   `server`'s `ETag` type (new intra-module coupling), or should the type
   move to a shared subpackage first? Public-surface decision.
2. **Alex's field-report offer:** draft a reply email (Age answer +
   no-store/freshening changes are in), and accept his raw header captures
   as permanent fixtures in `client/testdata/`?
3. **Release workflow for go-etag:** the repo has no tag-triggered Release
   workflow; all five tags DO have manual Release pages (verified 2026-09-18
   via `gh release list`, v0.3.1 is Latest), so the gap is automation, not
   pages. Decide
   whether to add one or keep manual releases.
4. **FNV-64a strength honesty (standing decision, revisitable):** the
   default `Strong` is pragmatic (birthday bound ~4.3B bodies) but not
   spec-purist. Documented; changing the default would be breaking.
5. **Residual open-low backlog:** the ~25 annotated-open "low" items in the
   August reports (isolated parser benchmarks, single-pass `ParseETagList`,
   `Strength.String()`, obs-text validation, assorted boundary tests) are
   deliberately noise-gated — open in historical reports, tracked in no
   living doc. Promote them into TODO_LIST/ROADMAP cold storage, or let
   them die with the archives? (Asked 2026-09-10 and 2026-09-11; owner
   decision pending.)
6. **Deprecated shim scope:** should the root shim receive the new typed
   surface (`Code`/`Domain`/`DomainOf`/`InDomain`) in a v0.3.x, or stay
   frozen at the pinned v0.1.x surface until its v1.0.0 deletion? (Frozen
   for now; that's a product/compat call, not a technical one. Origin:
   report `2026-09-11_09-28_error-system-parity-typed-code.md` g#1.)
7. **YAGNI vs house parity:** keep all six error-family constructors
   (exact httputil mirror, two families unused by go-etag) or trim to the
   four actually used? Same call pending for an exported `Domain("http")`
   constant vs leaving it literal. (Origin: report
   `2026-09-11_09-28_error-system-parity-typed-code.md` f#6, f#7, g#3.)

# go-etag — AGENTS.md

## Hard Constraints (Will Break Your Code)

### Allowed Dependencies

The standard library, sibling modules in this repo, and `github.com/larsartmann/go-error-family` (same author, zero transitive deps; used by the server module only). No other third-party libraries — the go.mod files are the allowlist, and adding one anywhere is a policy change, not an edit.

### `exhaustruct_v5` — Every Struct Field Must Be Set

When creating any struct literal, you must populate **every field**. Use `//nolint:exhaustruct_v5` only for intentional zero-value returns (e.g. `ETag{}` for "no tag"). In test files this is relaxed. (Migrated from deprecated `exhaustruct`; v5 settings use `ignore-patterns` regexes, and nolint directives must name `exhaustruct_v5`.)

### `err113` — No Inline `errors.New()`

Package-level sentinel errors only. Do not call `errors.New()` or `fmt.Errorf()` inside functions to create error values that could be package-level sentinels.

### `wsl_v5` — Strict Whitespace Rules

Enforces blank lines before `return`, after declarations, and around control flow. Run `golangci-lint fmt` after editing.

### `nonamedreturns` — No Named Return Values

Do not use named returns in function signatures.

### `noctx` — Always Use Context

`http.NewRequest` is banned. Use `http.NewRequestWithContext`.

### `godot` — Comments End With Periods

All comments must end with a period.

### `mnd` — No Magic Numbers

Extract numeric literals into named constants.

### `gosec` — G705 Excluded Globally

This library's purpose is writing HTTP response bodies, so every `ResponseWriter.Write` is intentional output.

### `paralleltest` — Every Test Must Call `t.Parallel()`

If you write a test function, it must call `t.Parallel()` as its first line.

### `noinlineerr` — No Inline Error Checks

Forbidden: `if err := foo(); err != nil`. Use a separate assignment followed by the check.

### `canonicalheader` — Canonical Header Keys

Header keys must match Go's canonical MIME header form.

### `testableexamples` — Examples Need Output

Every `Example*` function must include a `// Output:` comment directive.

### `thelper` — Test Helpers Must Call `t.Helper()`

Any function taking `*testing.T` that calls `t.Fatal`/`t.Error` must start with `t.Helper()`.

## Commands

Local go is 1.26.7 with `GOTOOLCHAIN=local` persisted, so prefix every go command — and every golangci-lint invocation (it shells out to go) — with `GOTOOLCHAIN=auto` (downloads the go.mod toolchain once into the module cache). Do not change the persisted go env. Go directive history: v0.4.0 tagged `go 1.27.1`; daemon commit `969d077` (2026-09-22, unreviewed) relaxed it to `go 1.27`; restored to `1.27.1` on 2026-09-23 (commit `45e68c8`); the daemon relaxed it AGAIN at `6f15f81`; the 2026-09-23 module split re-pinned `go 1.27.1` in all five go.mod files and made the pre-release-check parity check exact-match (anchored) so the relaxation class fails loudly next time. The LSP is healthy: the deployed `~/.config/crush/crushrc` launches gopls and golangci_lint_ls with `--env GOTOOLCHAIN auto` (fix in the LarsArtmann/crush-config repo, deployed 2026-09-22), so in-editor diagnostics load; the `GOTOOLCHAIN=auto` CLI prefix is still required — only the LSP env was fixed, not the persisted global go env.

**Multi-module repo (since the 2026-09-23 split):** five modules — root tombstone, `entitytag/`, `server/`, `client/`, `metrics/` — joined by the committed `go.work`. From the repo root, `./...` spans ONLY the root module; carry the full pattern set, or cd into a module and run plain `./...`:

```bash
PACKAGES="./... ./client/... ./entitytag/... ./metrics/... ./server/..."
GOTOOLCHAIN=auto go test $PACKAGES              # Run tests (all five modules)
GOTOOLCHAIN=auto go test -race $PACKAGES        # Race detection (REQUIRED for tests with t.Parallel() or shared state)
GOTOOLCHAIN=auto go vet $PACKAGES               # Vet
GOTOOLCHAIN=auto go test -bench=. $PACKAGES     # Benchmarks
GOTOOLCHAIN=auto golangci-lint run              # Lint (root module only!)
for m in client entitytag metrics server; do (cd $m && GOTOOLCHAIN=auto golangci-lint run --config ../.golangci.yml); done  # Lint nested modules (golangci-lint has no workspace mode)
GOTOOLCHAIN=auto golangci-lint fmt              # Format (gofumpt + golines@120 + gci; root module — run per module too)
# Error audit is module-scoped (erraudit refuses to cross nested go.mod boundaries; run per module):
for m in . client entitytag metrics server; do (cd $m && GOTOOLCHAIN=auto GOEXPERIMENT=jsonv2 erraudit ./... --type-aware --enforce-go-error-family --enforce-samber-oops --enforce-generic-return); done
# nolint audit: per module, from the module root (a repo-root scan misjudges nested-module directives as stale; the root module has no nolint directives):
for m in client entitytag metrics server; do (cd $m && GOTOOLCHAIN=auto GOEXPERIMENT=jsonv2 erraudit nolint-audit .); done
```

**Verification gate (first pass, not last):** `golangci-lint fmt` (root + per module) → `golangci-lint run` root + per nested module (0 issues) → `go vet` over `$PACKAGES` → `go test -race -count=1` over `$PACKAGES` — even for comment-only edits (godot/wsl bite comments too; all commands take the `GOTOOLCHAIN=auto` prefix). `scripts/pre-release-check.sh` encodes the whole loop. Supplement when touching clones or error paths: `art-dupl -t 1 --type-aware` (expect exactly the 1 accepted group).

## Architecture

Five Go modules in one repo, released on a shared version train (all modules tag the same vX.Y.Z per release; a module's tag points at the commit where its go.mod/go.sum finalized — see Release Conventions). The shared domain type lives in `entitytag/` (module `…/entitytag`, package `entitytag`, zero deps), the real code in `server/` (module `…/server`, package name `etag`, deps: entitytag + go-error-family) and `client/` (module `…/client`, package `etagclient`, dep: entitytag only), the observability companion in `metrics/` (module `…/metrics`, package `metrics`, dep: server; moved in from `httputil/etagmetrics` 2026-09-22); the root module (`github.com/larsartmann/go-etag`) is the deprecated alias shim over the server package (deleted at v1.0.0). Dependency direction: `server → entitytag ← client` — the client must never import `server`; `metrics → server` is a leaf — the server must never import `metrics`. One external dependency in the whole repo: `github.com/larsartmann/go-error-family` (server module only). Go 1.27.1 in every go.mod.

**Workspace mechanics (the non-obvious part):** the committed `go.work` `use`s all five modules AND carries version-qualified `replace` directives for each (`…/entitytag v0.6.0 => ./entitytag`, …). The replaces are REQUIRED, not optional: with sibling requires pointing at an unreleased train version (v0.6.0 does not exist on the proxy until the staircase tags it), `use` alone cannot divert the merged module graph once any external dependency (go-error-family) forces graph resolution — builds fail with `unknown revision entitytag/v0.6.0`. Unqualified replaces conflict with `use` ("replaced at all versions"), hence version-qualified. The replaces live only in go.work, which consumers and the proxy never see, so every go.mod stays replace-free. When the train moves to v0.7.0: bump the sibling requires in go.mod files AND the replace versions in go.work together — `scripts/pre-release-check.sh` (version-train sync check) fails on any drift. `go mod tidy` is a release-time step only: mid-cycle it would try to resolve the unreleased train version and fail; workspace builds/tests/lint never need it.

| File                      | Exports                                                                                                                                                                             | Purpose                                                                                                                                                              |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `entitytag/entity_tag.go` | `ETag`, `Strength`, `Strong`, `Weak`, `NewETag`, `ParseETag`, `ParseETagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`, `Strength.IsValid`                                           | RFC 7232 §2.3 entity-tag domain type: opaque value + strength, strong/weak comparison, ABNF parser (moved here from `server/` 2026-09-18)                            |
| `server/entity_tag.go`    | same surface as `entitytag` (type aliases + wrapper functions)                                                                                                                      | Re-export shim keeping the historical `etag.ETag` surface compiling unchanged; add new domain logic to `entitytag/`, never here                                      |
| `server/etag.go`          | `ETagConfig`, `DefaultETagConfig()`, `Validate()`, `New()`                                                                                                                          | ETag middleware: FNV-64a generation, If-None-Match 304, buffer overflow, Hijack/Flush streaming                                                                      |
| `server/wrapper.go`       | (unexported `responseWrapper`)                                                                                                                                                      | Shared ResponseWriter wrapper: buffers WriteHeader, delegates Hijack/Flush                                                                                           |
| `server/middleware.go`    | `Middleware`                                                                                                                                                                        | Type alias for `func(http.Handler) http.Handler`                                                                                                                     |
| `server/errors.go`        | `ErrCodeETagWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeInvalidConfig`, `ErrCodeHashWriteFailed`, `ErrInvalidConfig`, `RegisterErrorClassifications()` | Error codes + stdlib sentinel registration + message templates                                                                                                       |
| `server/code.go`          | `Code`, `Domain`, `DomainOf`, `InDomain`                                                                                                                                            | Typed error-code surface: 6 family constructors + 6 Wrap methods per `Code`, component-domain routing (mirrors httputil's `Code`)                                    |
| `server/hex.go`           | (unexported `hexEncodeUint64`, `hexDigitsLower`)                                                                                                                                    | Zero-alloc stack-allocated hex encoding for uint64 FNV hashes                                                                                                        |
| `server/doc.go`           | (package doc only)                                                                                                                                                                  | Package-level GoDoc documentation                                                                                                                                    |
| `client/options.go`       | `Options` (KeyFunc / MaxEntries / MaxBodyBytes / FreshenOn304 / FromCacheHeader), `FreshenPolicy` + `FreshenPerRFC` / `FreshenFields` / `FreshenNone`                               | Client transport options with defaults (256 entries / 1 MiB / zero policy = RFC 9111 §4.3.4 freshening)                                                              |
| `client/cache.go`         | `Stats` (Hits/Stored/Entries); unexported `storedResponse` + `storedValidator` (wire string + parsed `entitytag.ETag`), `responseCache` (map + order-slice FIFO under a mutex)      | Bounded concurrency-safe cache with counters; validator parsed once at store time                                                                                    |
| `client/transport.go`     | `NewTransport`, `Transport` (`RoundTrip`, `Stats`)                                                                                                                                  | Conditional GET transport: If-None-Match replay, 304→200 rebuild with §4.3.4 freshening, §4.4 invalidation, no-store guard, body store with size skip                |
| `client/doc.go`           | (package doc only)                                                                                                                                                                  | Package docs incl. the KeyFunc credential warning                                                                                                                    |
| `metrics/metrics.go`      | `Counters` (`Generated`/`NotModified`/`BufferOverflows`), `Snapshot`, `Attach`, `HitRatio`                                                                                          | Atomic counters for the server hooks; preserves consumer hooks (they run after counting); moved from `httputil/etagmetrics` (was its own Go module, httputil v1.3.0) |
| `metrics/doc.go`          | (package doc only)                                                                                                                                                                  | Metric-agnostic stance: exposition format stays consumer-owned                                                                                                       |
| `deprecated.go`           | Full alias surface of `server/` (4 types, 7 consts, 1 var, 8 func wrappers), all `// Deprecated:`                                                                                   | Compatibility shim keeping v0.1.x imports compiling; removed at v1.0.0                                                                                               |
| `doc.go`                  | (package doc only)                                                                                                                                                                  | Tombstone package doc with migration snippet                                                                                                                         |

**Middleware pattern:** `New()` returns `func(http.Handler) http.Handler` (aliased as `Middleware`).
**Client pattern:** `NewTransport(next, opts) *Transport`; nil `next` falls back to `http.DefaultTransport`; the cache is embedded (no nil-cache state).

## API Design

### ETag Type (RFC 7232 §2.3)

The `ETag` struct holds an opaque string and a `Strength` (Strong/Weak). It provides:

- `String()` — wire format (`"opaque"` or `W/"opaque"`)
- `StrongEqual(o ETag) bool` — §2.3.2 strong comparison (both must be strong)
- `WeakEqual(o ETag) bool` — §2.3.2 weak comparison (ignores strength)
- `ParseETag(s string) (ETag, bool)` — parse one entity-tag from wire format
- `ParseETagList(header string) []ETag` — parse comma-separated list
- `MatchesIfNoneMatch(tag, header) bool` — weak comparison helper (used internally)
- `MatchesIfMatch(tag, header) bool` — strong comparison helper (for application code)

### Config

`ETagConfig` has fields: `Strength`, `MaxBufferSize`, `HashFunc` (returns string), `SkipIfPresent`, `Skip`, `OnError`, `OnETagGenerated`, `On304`, `OnBufferOverflow`.

- Zero-value `ETagConfig{}` is safe: MaxBufferSize clamped to default (1 MB).
- `HashFunc func([]byte) string` returns opaque-tag content (not a uint64).
- Observability hooks (all nil by default, zero cost when nil): `OnETagGenerated func(ETag)`, `On304 func(ETag)`, `OnBufferOverflow func(int)`. They make the library OTEL/Prometheus-ready without a telemetry dependency.

### Client (package `etagclient`)

`Options` has fields: `KeyFunc` (nil = request URL string; MUST scope by credential when responses vary by caller), `MaxEntries` (default 256, FIFO eviction), `MaxBodyBytes` (default 1 MiB; oversized bodies pass through uncached with bodies intact; counts decoded bytes under transparent gzip), `FreshenOn304` (`FreshenPolicy` — zero value = RFC 9111 §4.3.4 freshening, every 304-provided field replaces the stored value minus the §3.1/§3.2 exceptions; `FreshenFields(...)` restricts freshening to those fields; `FreshenNone()` disables it; the pre-v0.3.0 `PreserveOn304 []string` nil/empty overload is gone), `FromCacheHeader` (default empty = disabled). `Transport.Stats()` returns `Stats{Hits, Stored, Entries}`.

**Client gotchas:**

- **RoundTrip never mutates the caller's request** — If-None-Match rides on a clone (net/http RoundTripper contract).
- **Validator comparison is typed and parses once at store time** — a stored entry is a `storedResponse` whose `storedValidator` identity holds the wire string (replayed verbatim as `If-None-Match`) plus the `entitytag.ETag` parsed by `newStoredValidator` at store time; `storedValidator.weaklyMatches` parses only the incoming candidate, so §4.3.5 HEAD-confirmation and §4.3.4 304 mismatch-restore never re-parse stored state. A field value that does not parse as an RFC 7232 §2.3 entity-tag (unclosed quote, bare `*`, lowercase `w/`) can never weak-match anything, so HEAD-confirmation and 304 mismatch-restore treat it as a non-match. Parse is allocation-free (`reports/bench/2026-09-18_{before,after}-typed-validator.txt`; `reports/bench/2026-09-18_baseline-typed-code-stored-validator.txt`).
- **Caller-supplied `If-None-Match` is never clobbered** — a caller who set their own conditional keeps it; a 304 answering it passes through unrebuilt (RFC 9110 §13.1.2 ownership).
- **304 freshening is the RFC 9111 §4.3.4 default** — every field the 304 provides replaces the stored value, except hop-by-hop (§3.1), Content-Length/Content-Range (§3.2), and Content-Encoding when net/http transparently decoded the body (§3.2 integrity allowance; recorded via `resp.Uncompressed`).
- **Freshening persists to the stored entry** — the 304's validator replaces the stored one for later revalidations; `responseCache.freshen` skips when a concurrent 200 already replaced the entry.
- **Mismatched 304 validators are not adopted** — a 304 declaring an ETag that does not weak-match the one it validated (broken server naming a different representation) leaves both the stored validator and the rebuilt ETag alone (§4.3.4 filtering, `restoreMismatchedValidator`).
- **§4.4 invalidation** — a non-error (2xx/3xx) response to an unsafe method (not GET/HEAD/OPTIONS/TRACE) drops the stored entry for that URI; error responses and safe methods keep it. The key is derived lazily: only after a response actually triggers invalidation (passthrough round trips never call `KeyFunc`).
- **§4.3.5 HEAD freshening is implemented** — a HEAD 200 with a comparable validator (ETag weakly, Last-Modified exactly) and matching Content-Length updates stored metadata via §3.2 rules; any mismatch — or no comparable validator — invalidates; no-store HEAD does neither. `FreshenOn304` does NOT apply to HEAD updates. HEAD is never made conditional and never rebuilt.
- **Request `no-store` bypasses the cache** (§5.2.2.5 request directive) — a no-store GET never injects validators, never rebuilds, never stores; a no-store HEAD skips freshening; unsafe methods still invalidate (deletion is not storage).
- **Rebuilt responses carry `Uncompressed`** when the stored body is the transparently decoded form.
- **Age is surfaced, and cannot run backwards** — a 200's Age passes through verbatim (§5.1 staleness signal); a 304's Age replaces the stored Age on rebuilds. An edge cache serving a stale copy is detectable by Age even though the ETag faithfully matches.
- **`Cache-Control: no-store` responses are never stored** — RFC 9111 §3 MUST NOT; quote-aware, case-insensitive directive parsing (a quoted `"no-store"` argument does not match).
- **Hop-by-hop fields are stripped at store time** — Connection-listed fields, Keep-Alive, Proxy-* (§3.1), so rebuilds cannot resurrect them.
- **Vary is not negotiated** — the cache key is the URL (or KeyFunc); RFC 9111 §4.1 secondary keys are out of scope, so responses varying on request headers must be disambiguated via `KeyFunc`.
- **Oversized bodies keep streaming** — the buffered prefix is chained to the unread remainder; Close still reaches the original body.
- **Go 1.26 canonical header form of `ETag` is `Etag`** — map literals with `"ETag"` keys are invisible to `Header.Get/Set` (Get canonicalizes on read). Build stub headers via `Header.Set` in tests; real transports canonicalize on both sides.
- **Runtime canonicalization vs the `canonicalheader` linter are different authorities** — the linter demands the literal `ETag` spelling in source (our `headerETag = "ETag"` constant); the runtime canonicalizes whatever you pass into `Etag` on the wire. Do not "fix" one to match the other.
- **`drainAndClose` deliberately ignores drain/Close errors** — the 304 is discarded either way, and net/http closes an undrained connection, so the worst case is lost connection reuse. Suppressed with documented `//nolint:erraudit` directives (validate with `erraudit nolint-audit .` from INSIDE `client/` — it takes a path, not `./...`, and a repo-root scan misjudges nested-module directives as stale); `--no-suppress` audit mode surfaces them by design.

## Error Classification

Errors from `etagWriter` are classified using `go-error-family`:

| Source       | Error Code                    | Family         | Retryable | When                                              |
| ------------ | ----------------------------- | -------------- | --------- | ------------------------------------------------- |
| `Write`      | `http.etag_write_failed`      | Transient      | Yes       | Streaming, overflow, or flush write failure       |
| `Hijack`     | `http.hijack_unsupported`     | Infrastructure | No        | Underlying writer doesn't implement Hijacker      |
| `Hijack`     | `http.hijack_failed`          | Transient      | Yes       | Underlying Hijack call fails                      |
| `Validate`   | `http.etag_config_invalid`    | Rejection      | No        | ETagConfig field value is invalid (e.g. <= 0)     |
| `hash.Write` | `http.etag_hash_write_failed` | Orchestration  | No        | Hash.Write returned an error (contract violation) |

All errors are `*errorfamily.Error` — classified, contextual, retryable-aware.
`ErrInvalidConfig` is the package-level sentinel for `Validate`, declared as the `error` interface (erraudit's sentinel guard rejects concrete-typed sentinels); `errors.Is` matches by code+family, not identity. `Validate` derives context-bearing errors via `newInvalidConfig().WithContextf(...)` — fresh instances, so the package-level sentinel stays immutable — and the deprecated root shim re-exports the sentinel with the same interface type.
Hijack errors include `writer_type` context via `WithContextf`.
Flush-path write errors are forwarded to `ETagConfig.OnError` (a `func(*errorfamily.Error)`) for observability, since they cannot be surfaced to the client or returned from `Write`.
Internal construction goes through the typed `Code` constants in `server/errors.go` (e.g. `codeETagWriteFailed.WrapTransient(...)`, mirroring httputil's pattern); the exported `ErrCode*` strings stay untyped for backward compatibility. Message templates live in the declarative `errorTemplates` map (registered by `RegisterErrorClassifications` looping over it), pinned bidirectionally against `allETagErrorCodes` — the canonical code list in `server/errors.go` — by `errors_test.go`; add a new code to `allETagErrorCodes` and `errorTemplates` together. httputil's `RegisterErrorClassifications` registers these templates verbatim on go-etag's behalf, so template wording must not drift; keep the client clamp-don't-reject design (no classified errors there) unless making a deliberate breaking change.

## Non-Obvious Behaviors

- **The middleware constructor is `New(cfg ETagConfig)`** — not `ETag()`. `ETag` is the domain type.
- **`HashFunc` returns a string** — the opaque-tag content, not a uint64. The middleware wraps it with quotes and optional `W/`.
- **Zero-value config is safe** — MaxBufferSize <= 0 is clamped to 1 MB in `newETagWriter`.
- **HEAD responses have no body** — RFC 7230 §3.3. Content-Length is set from the buffered body size.
- **304 strips Content-Length** — RFC 7232 §4.1.
- **`SkipIfPresent` defaults to false** — set to true to respect handler-set ETags.
- **`Skip` predicate** — exclude specific routes from ETag processing.
- **`MatchesIfMatch` is an exported helper** — applications call it in their handlers for unsafe-method lost-update prevention.
- **`ETag` `If-None-Match` uses RFC 7232 §2.3.2 weak comparison** — `W/"abc"` and `"abc"` are treated as equivalent.
- **Buffer overflow disables ETag** — responses exceeding `MaxBufferSize` are streamed without ETag.
- **`OnETagGenerated` only fires for computed tags** — handler-provided tags adopted via `SkipIfPresent` do NOT fire it (name honesty: "generated" ≠ "adopted").
- **`On304` fires regardless of tag source** — computed or handler-provided — and fires in addition to `OnETagGenerated` (ordering: generated first, then 304).
- **`OnBufferOverflow` fires exactly once per response** — the flushed-flag transition guarantees single fire; handler-initiated `Flush()` does NOT fire it. Payload is the exceeded `MaxBufferSize`.
- **Hooks are synchronous and unprotected** — they run in the request goroutine with no recover; net/http isolates panics like handler panics.
- **`metrics.HitRatio()` is `NotModified / Generated` and the denominator is exact** — `On304` fires in addition to `OnETagGenerated` for computed tags, so `Generated` alone counts every tag-computing response; do not "complete" the denominator by adding `NotModified` (that double-counts every 304 — the bug fixed on move-in from httputil v1.3.0). 304s on handler-adopted tags (`SkipIfPresent`) fire `On304` without `OnETagGenerated` and can push the ratio above 1; the doc comment says so.
- **Hijack/Flush switches to streaming mode** — after either call, the middleware writes through without buffering.
- **Hash.Write errors panic with a classified Orchestration error** — the `hash.Hash` contract guarantees Write never fails; if it does, the hash implementation is broken and we panic with `http.etag_hash_write_failed`.
- **The one `art-dupl` clone group is accepted — do not extract it** — `art-dupl -t 1 --type-aware` reports a single group: `etagWriter.Write`'s body-buffer append vs `nonHijackableRecorder.Write` in `server/testutil_test.go`. The duplication is deliberate: the test double must NOT implement `http.Hijacker` (that is its entire testing purpose), so it cannot embed `httptest.ResponseRecorder` or share any production buffer type; production code cannot import test helpers; and the shared logic is a single `append` plus `return len(b), nil` — any abstraction would take more parameters than the duplicated code has lines. Full rationale on the double's doc comment; accept this group in every future dedup pass.

## Testing Conventions

- **Same package** (`package etag` in `server/`, `package etagclient` in `client/`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **BDD-style specs** in `server/etag_bdd_test.go` — Describe/Context/It pattern using `t.Run`
- **RFC-citing spec suite** in `client/spec_test.go` — pins the transport to RFC 9111 requirements (Age surfacing §5.1, freshening §4.3.4/§3.2, HEAD freshening §4.3.5, request/response no-store §3/§5.2.2.5, invalidation §4.4, HEAD-never-conditional, caller-owned If-None-Match); every test cites its section. The full requirement-by-requirement account lives in `docs/rfc9111-conformance.md`
- **Real-server integration test** in `client/integration_test.go` — httptest.Server + real http.Client round trip for canonical header forms, bodiless 304s, and §4.4 invalidation through a real PUT (the next GET must refetch unconditionally)
- **Real-wire server integration** in `server/integration_test.go` — pins Content-Length framing on 200s, HEAD body suppression with advertised length, and a bodiless 304 without Content-Length over TCP (the rest of the server suite is recorder-only)
- **Shim export-parity suite** in `deprecated_test.go` (root) — compile-time type-identity assertions plus wrapper smoke tests pin the deprecated alias surface until its v1.0.0 removal
- **Client fuzz target** in `client/fuzz_test.go` — `FuzzHasNoStoreDirective` with a soundness property (a literal top-level no-store directive is always detected), wired into the CI fuzz job alongside the server targets
- **`t.Errorf`** for non-fatal, **`t.Fatalf`** for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for server doubles; `roundTripperFunc` stubs for client doubles
- **Shared test helpers** in `server/testutil_test.go`; client helpers live in `client/transport_test.go`
- **Benchmark discipline** — capture `-benchmem -count=6` baselines under `reports/bench/<date>_<name>.txt` before and after perf-relevant changes (e.g. `2026-09-11_before-lazy-key.txt` / `_after-lazy-key.txt`, `_baseline-post-435.txt`); smoke-run `-count=1` only for quick sanity checks

## Repo Workflow Notes

- **An auto-git daemon commits and re-formats continuously** — status reports and tables get re-touched within minutes of your edit. Always re-read a shared file immediately before re-editing it; never assume remembered file geometry.
- **`docs/status/` and `docs/planning/` are point-in-time snapshots** — never rewrite their narrative; annotate resolved items inline (`~~item~~ done at`hash``) per the docs-health skill. The living backlog is `TODO_LIST.md`, never the reports.
- **Consumer-migration recipe (per go-ecosystem-upgrade sweeps):** before bumping go-etag in a consumer repo, check its version surface first (`flake.nix` inputs, `vendorHash`, CI pins — DiscordSync's drift-guard test exists for exactly this); commit immediately after the gate so the auto-daemon doesn't capture the diff; run `nix build` whenever that repo's `go.sum` changed; verify workspace repos in BOTH modes (hermetic `GOWORK=off` + workspace MVS).
- **Version-train discipline (multi-module):** all internal requires and the go.work replaces move to the new vX.Y.Z TOGETHER, in the same commit. `scripts/pre-release-check.sh` verifies the train is in sync (single internal version everywhere; every require has its matching go.work replace). A consumer importing `…/client` after v0.6.0 pulls client+entitytag only — the composability win the split bought; the root module no longer bundles the other packages, so consumers importing nested paths via `go get github.com/larsartmann/go-etag@vX` need one extra `go mod tidy`/`go get` of the nested module (import paths themselves never changed).

## Release Conventions (validated at v0.4.0 and v0.5.0)

- **Release commit style:** `chore(release): cut CHANGELOG vX.Y.Z` (matches v0.2.0/v0.3.0), CHANGELOG cut BEFORE tagging; empty `[Unreleased]` placeholders + compare links stay behind.
- **Run `scripts/pre-release-check.sh` on the exact release commit before the CHANGELOG cut** — one command for every locally checkable gate (build, vet, race, lint, fmt-diff, erraudit + nolint-audit, `go mod verify`, replace/pseudo-version ban, and go-directive ↔ CI-pin parity that catches the 969d077 daemon-relaxation class of drift). v0.4.0 shipped with skipped steps (`go mod verify`, nolint-audit); the script exists so that class cannot recur.
- **The daemon can sweep the release-prep commit first** (v0.4.0: the CHANGELOG cut landed inside a `chore: auto-commit`). Harmless if the tree content is exactly the release content — verify the tag's `.info` hash on proxy.golang.org matches the intended commit, as done at v0.4.0 (16369cc).
- **v0.x GitHub Releases are published as Latest, non-prerelease** — house precedent (v0.1.0 → v0.5.0), a deliberate deviation from the go-release skill's `--prerelease` suggestion.
- **The consumer-facing gates are proxy.golang.org + sum.golang.org + a clean-room `go get`** — pkg.go.dev doc rendering lags by minutes and is NOT a release blocker; verify it eventually, not gate on it.
- **Tag only after CI is green on the exact commit** (workflow triggers on both master pushes and `v*` tags, so the tag gets its own frozen run).
- **The go-release skill lives at** `~/.config/crush/skills/go-release/` — load it for any release; it repays its reading cost in immutable-tag discipline.
- **v0.5.0 (2026-09-23, commit `af2b6d1`) validated the full loop with zero skipped steps**: pre-release-check.sh exit 0 on the exact commit, master pushed so CI ran green on it, annotated tag, frozen tag-run green, proxy `.info` hash == commit, sum.golang.org recorded, clean-room `go get …/metrics@v0.5.0` + Attach/HitRatio smoke, GitHub Release Latest, pkg.go.dev rendering verified within the hour. The floor decision (TODO_LIST #2) was resolved by restoring `go 1.27.1` (commit `45e68c8`) to match the README badge and all three CI pins.

## Release Conventions, multi-module era (from v0.6.0, after the 2026-09-23 split)

The single-commit discipline becomes a **bottom-up tag staircase** — the price of five independently consumable modules. Each module's tag points at the commit where its own go.mod/go.sum finalized; subtree content is identical across the staircase, which is all consumers ever see.

1. **Prepare:** land all code changes; run `scripts/pre-release-check.sh` (workspace mode — it never needs the sibling tags). Cut the CHANGELOG as usual (`chore(release): cut CHANGELOG vX.Y.Z`).
2. **Staircase, dependency-first — entitytag → server → client → metrics → root.** For each module in order, from its directory: `GOTOOLCHAIN=auto go mod tidy && GOTOOLCHAIN=auto go mod verify && GOWORK=off go build ./... && GOWORK=off go test -race -count=1 ./...` (tidy resolves the already-tagged dependencies below it; the first module, entitytag, has none), commit the tidy result if it changed go.mod/go.sum, wait for CI green on that commit, then create the ANNOTATED tag: `entitytag/vX.Y.Z`, `server/vX.Y.Z`, `client/vX.Y.Z`, `metrics/vX.Y.Z`, and finally the root `vX.Y.Z`. Root goes last — it is the "release commit" whose CHANGELOG cut the release.
3. **Bump the train for the next cycle:** after tagging, set the sibling requires in go.mod files and the replace versions in go.work to the NEXT dev version (or leave them at the just-released version and bump at the next release — either way the pre-release-check sync gate enforces they agree).
4. **Consumer gates per module tag:** proxy `.info` hash == the tagged commit, sum.golang.org entry, and a clean-room `go get github.com/larsartmann/go-etag/<module>@vX.Y.Z` for each of the four nested modules plus the root. Proxy fetches of a fresh tag can lag a minute or two; `GOPROXY` direct fallback covers it.
5. **Pre-tag bootstrap (v0.6.0 only):** the nested requires point at v0.6.0 before any such tag exists. That is expected: workspace mode (go.work) keeps every gate green; ONLY the staircase's tidy/GOWORK=off steps need the tags to exist, in order. Do not "fix" the unresolvable requires with go.mod replace directives — the release gate bans replace in every go.mod, and proxy consumers would break.

Everything else carries over: tag only after CI green on the exact commit, GitHub Release as Latest, load the go-release skill, pkg.go.dev is not a gate. The proposal doc `docs/modularization/2026-09-23_modularization-proposal.html` records why metrics got a module (tombstone purity, not composability) and why go.mod replace directives stay banned (proxy-published repo).

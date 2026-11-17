# go-etag — AGENTS.md

## Hard Constraints (Will Break Your Code)

### Allowed Dependencies

The standard library, sibling modules in this repo, and `github.com/larsartmann/go-error-family` (same author, zero transitive deps; used by the server module only). No other third-party libraries — the go.mod files are the allowlist, and adding one anywhere is a policy change, not an edit. This also rejects BuildFlow's `go-auto-upgrade` suggestions to use `lo.Map`/`lo.SliceToMap` (samber/lo is third-party).

### Linter Rules (golangci-lint, all enforced)

| Linter              | Rule                                                                                                                                          |
| ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `exhaustruct_v5`    | Every struct-literal field set; `//nolint:exhaustruct_v5` only for intentional zero-value returns. Relaxed in test files.                     |
| `err113`            | No inline `errors.New()`/`fmt.Errorf()` where a package-level sentinel fits.                                                                  |
| `wsl_v5`            | Blank lines before `return`, after declarations, around control flow. Run `golangci-lint fmt` after editing.                                  |
| `nonamedreturns`    | No named return values.                                                                                                                       |
| `noctx`             | `http.NewRequest` banned; use `http.NewRequestWithContext`.                                                                                   |
| `godot`             | Comments end with periods (bites comment-only edits too).                                                                                     |
| `mnd`               | No magic numbers; extract named constants.                                                                                                    |
| `gosec`             | G705 excluded globally — `ResponseWriter.Write` is this library's intentional output.                                                         |
| `paralleltest`      | Every test calls `t.Parallel()` first.                                                                                                        |
| `noinlineerr`       | No `if err := foo(); err != nil`; assign, then check.                                                                                         |
| `canonicalheader`   | Header keys in Go's canonical MIME form (see the runtime-vs-linter note under Client gotchas).                                                |
| `testableexamples`  | Every `Example*` has an `// Output:` directive.                                                                                               |
| `thelper`           | Helpers taking `*testing.T` that fail start with `t.Helper()`.                                                                                |

## Commands

Local go is 1.26.7 with `GOTOOLCHAIN=local` persisted, so prefix every go command — and every golangci-lint invocation (it shells out to go) — with `GOTOOLCHAIN=auto`. Do not change the persisted go env. All five go.mod files pin `go 1.27.1`; the auto-commit daemon has twice relaxed the directive (`969d077`, `6f15f81`), and the 2026-09-23 module split re-pinned it and made the pre-release-check parity check exact-match, so that drift class now fails loudly. The LSP already injects `GOTOOLCHAIN=auto`; the CLI prefix is still required.

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
# Doc snippet symbol check (ported from httputil 2026-10-15; wired into ci.yml):
GOTOOLCHAIN=auto go run ./scripts/doc-snippet-refs README.md docs/*.md
```

**Known tool noise (do not "fix"):** golangci-lint warns `unknown linters in //nolint directives: erraudit` — benign, because the directives serve the standalone erraudit CLI (module-scoped, run separately above), not golangci-lint; do not delete `//nolint:erraudit` directives over this warning (they are re-validated by the nolint audit). The standalone CLI's `--explain` flag is deliberately not used in these examples — the gate runs without it (recorded 2026-09-23, M19/F73).

**Verification gate (first pass, not last):** `golangci-lint fmt` (root + per module) → `golangci-lint run` root + per nested module (0 issues) → `go vet` over `$PACKAGES` → `go test -race -count=1` over `$PACKAGES` — even for comment-only edits. `scripts/pre-release-check.sh` encodes the whole loop. Supplement when touching clones or error paths: `art-dupl -t 1 --type-aware` (expect exactly the 4 accepted groups, see Non-Obvious Behaviors).

## BuildFlow Gate

`buildflow --fix --build-mode full` is the standing quality gate; it exits non-zero while error-severity findings remain.

- Go files stay ≤350 lines; AGENTS.md ≤220 lines and refreshed within 14 days (file-size-check / go-structure-linter). Split by cohesive topic when a file outgrows it — done 2026-11-17: `server/etag.go` → `server/config.go`, `client/transport.go` → `rebuild.go`/`header.go`/`body.go`, and the big test files split by topic.
- `example_test.go` files stay in-package — godoc attaches `Example*` funcs only from the package's own test files, so the go-structure-linter "use examples/ directory" suggestion is a deliberate non-fix. `internal/` does not apply either: this is a library whose exports ARE the API.
- govulncheck stdlib findings (e.g. GO-2026-6859) are warnings here: the remedy is a coordinated fleet Go-toolchain bump, not a library edit — route to TODO_LIST, do not bump the go directive unilaterally.

## Architecture

Five Go modules in one repo, released on a shared version train (all modules tag the same vX.Y.Z per release; a module's tag points at the commit where its go.mod/go.sum finalized — see `docs/release-playbook.md`). The shared domain type lives in `entitytag/` (module `…/entitytag`, package `entitytag`, zero deps), the real code in `server/` (module `…/server`, package name `etag`, deps: entitytag + go-error-family) and `client/` (module `…/client`, package `etagclient`, dep: entitytag only), the observability companion in `metrics/` (module `…/metrics`, package `metrics`, dep: server; moved in from `httputil/etagmetrics` 2026-09-22); the root module (`github.com/larsartmann/go-etag`) is the deprecated alias shim over the server package (deleted at v1.0.0). Dependency direction: `server → entitytag ← client` — the client must never import `server`; `metrics → server` is a leaf — the server must never import `metrics`. One external dependency in the whole repo: `github.com/larsartmann/go-error-family` (server module only). Go 1.27.1 in every go.mod.

**Workspace mechanics (the non-obvious part):** the committed `go.work` uses all five modules AND carries version-qualified `replace` directives for each (`…/entitytag v0.6.0 => ./entitytag`, …). The replaces are REQUIRED, not optional: with sibling requires pointing at an unreleased train version (v0.6.0 does not exist on the proxy until the staircase tags it), `use` alone cannot divert the merged module graph once any external dependency (go-error-family) forces graph resolution — builds fail with `unknown revision entitytag/v0.6.0`. Unqualified replaces conflict with `use` ("replaced at all versions"), hence version-qualified. The replaces live only in go.work, which consumers and the proxy never see, so every go.mod stays replace-free. When the train moves: bump the sibling requires in go.mod files AND the replace versions in go.work together — the pre-release-check sync gate fails on any drift. `go mod tidy` is a release-time step only: mid-cycle it would try to resolve the unreleased train version and fail; workspace builds/tests/lint never need it.

| File                      | Exports                                                                                                                                                                             | Purpose                                                                                                                                                              |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `entitytag/entity_tag.go` | `ETag`, `Strength`, `Strong`, `Weak`, `NewETag`, `ParseETag`, `ParseETagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`, `Strength.IsValid`                                           | RFC 7232 §2.3 entity-tag domain type: opaque value + strength, strong/weak comparison, ABNF parser (moved here from `server/` 2026-09-18)                            |
| `server/entity_tag.go`    | same surface as `entitytag` (type aliases + wrapper functions)                                                                                                                      | Re-export shim keeping the historical `etag.ETag` surface compiling unchanged; add new domain logic to `entitytag/`, never here                                      |
| `server/config.go`        | `ETagConfig`, `DefaultETagConfig()`, `Validate()`, `New()`                                                                                                                          | Config surface, defaults, FNV-64a default hash, middleware constructor (split out of `etag.go` 2026-11-17)                                                           |
| `server/etag.go`          | (unexported `etagWriter`)                                                                                                                                                           | Middleware writer runtime: If-None-Match 304, buffer overflow, Hijack/Flush streaming                                                                                |
| `server/wrapper.go`       | (unexported `responseWrapper`)                                                                                                                                                      | Shared ResponseWriter wrapper: buffers WriteHeader, delegates Hijack/Flush                                                                                           |
| `server/middleware.go`    | `Middleware`                                                                                                                                                                        | Type alias for `func(http.Handler) http.Handler`                                                                                                                     |
| `server/errors.go`        | `ErrCodeETagWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeInvalidConfig`, `ErrCodeHashWriteFailed`, `ErrInvalidConfig`, `RegisterErrorClassifications()` | Error codes + stdlib sentinel registration + message templates                                                                                                       |
| `server/code.go`          | `Code`, `Domain`, `DomainOf`, `InDomain`                                                                                                                                            | Typed error-code surface: 6 family constructors + 6 Wrap methods per `Code`, component-domain routing (mirrors httputil's `Code`)                                    |
| `server/hex.go`           | (unexported `hexEncodeUint64`, `hexDigitsLower`)                                                                                                                                    | Zero-alloc stack-allocated hex encoding for uint64 FNV hashes                                                                                                        |
| `client/options.go`       | `Options` (KeyFunc / MaxEntries / MaxBodyBytes / FreshenOn304 / FromCacheHeader), `FreshenPolicy` + `FreshenPerRFC` / `FreshenFields` / `FreshenNone`                               | Client transport options with defaults (256 entries / 1 MiB / zero policy = RFC 9111 §4.3.4 freshening)                                                              |
| `client/cache.go`         | `Stats` (Hits/Stored/Entries); unexported `storedResponse` + `storedValidator` (wire string + parsed `entitytag.ETag`), `responseCache` (map + order-slice FIFO under a mutex)      | Bounded concurrency-safe cache with counters; validator parsed once at store time                                                                                    |
| `client/transport.go`     | `NewTransport`, `Transport` (`RoundTrip`, `Stats`)                                                                                                                                  | Conditional GET transport core: If-None-Match replay, §4.4 invalidation, §4.3.5 HEAD confirmation, method/status classification                                     |
| `client/rebuild.go`       | (unexported)                                                                                                                                                                        | 304→200 rebuild, §4.3.4 freshening persistence, store path (split 2026-11-17)                                                                                        |
| `client/header.go`        | (unexported)                                                                                                                                                                        | Header-field semantics: clone/merge, freshening exceptions (§3.1/§3.2), hop-by-hop stripping, no-store directive parsing                                             |
| `client/body.go`          | (unexported `chainedBody`)                                                                                                                                                          | Chained buffered-prefix bodies for oversized/read-error streams; 304 drain-and-close                                                                                 |
| `metrics/metrics.go`      | `Counters` (`Generated`/`NotModified`/`BufferOverflows`), `Snapshot`, `Attach`, `HitRatio`                                                                                          | Atomic counters for the server hooks; preserves consumer hooks (they run after counting); moved from `httputil/etagmetrics` (was its own Go module, httputil v1.3.0) |
| `deprecated.go`           | Full alias surface of `server/` (4 types, 7 consts, 1 var, 8 func wrappers), all `// Deprecated:`                                                                                   | Compatibility shim keeping v0.1.x imports compiling; removed at v1.0.0                                                                                               |

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
- **The four `art-dupl` clone groups are accepted — do not extract them** — `art-dupl -t 1 --type-aware` shows 4 groups (baseline: `reports/dupl/2026-09-23_art-dupl-baseline.txt`): (1) `etagWriter.Write`'s flushed-write vs overflow-write blocks in `server/etag.go` — both delegate then classify with deliberately distinct error context ("streaming write failed" vs "overflow write failed"), a parameterized helper would hide the branch semantics the messages exist to convey; (2) the §4.3.5 HEAD-confirmation guard arms in `client/transport.go` (`return false` on ETag mismatch, `return false` on Last-Modified mismatch) — two distinct RFC rules, not duplication; (3) the read-error arm in `client/transport.go` and the oversized-body arm in `client/rebuild.go`, both chaining `resp.Body = newChainedBody(buffered, resp.Body)` — the shared logic is a single line, a helper would take more parameters than the duplication has lines; (4) the `default: return false` arms of the §3.2 field-filter and §3.1 hop-by-hop switches — idiom, not duplication. The historical fifth pair (the `Write` body vs `nonHijackableRecorder.Write` in `server/testutil_test.go`) is now filter-suppressed (test files) but the do-not-extract rule stands: the test double must NOT implement `http.Hijacker` (that is its entire testing purpose), so it cannot embed `httptest.ResponseRecorder` or share any production buffer type. Accept all of these in every future dedup pass — enforced, not just documented: the release gate fails unless exactly 4 groups are shown (OQ8, 2026-09-23).

## Testing Conventions

- **Same package** (`package etag` in `server/`, `package etagclient` in `client/`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **BDD-style specs** in `server/etag_bdd*_test.go` + `hooks_bdd_test.go` — Describe/Context/It pattern using `t.Run`
- **RFC-citing spec suite** across `client/spec*_test.go` — pins the transport to RFC 9111 requirements (Age surfacing §5.1, freshening §4.3.4/§3.2, HEAD freshening §4.3.5, request/response no-store §3/§5.2.2.5, invalidation §4.4, HEAD-never-conditional, caller-owned If-None-Match); every test cites its section; split by topic (freshen / freshen-exceptions / storage / conditional / invalidation / head / passthrough) 2026-11-17. The full requirement-by-requirement account lives in `docs/rfc9111-conformance.md`
- **Real-server integration test** in `client/integration_test.go` — httptest.Server + real http.Client round trip for canonical header forms, bodiless 304s, and §4.4 invalidation through a real PUT (the next GET must refetch unconditionally)
- **Real-wire server integration** in `server/integration_test.go` — pins Content-Length framing on 200s, HEAD body suppression with advertised length, and a bodiless 304 without Content-Length over TCP (the rest of the server suite is recorder-only)
- **Shim export-parity suite** in `deprecated_test.go` (root) — compile-time type-identity assertions plus wrapper smoke tests pin the deprecated alias surface until its v1.0.0 removal
- **Fuzz targets** — `client/fuzz_test.go` (`FuzzHasNoStoreDirective`, soundness property: a literal top-level no-store directive is always detected) and `server/fuzz_test.go` (`FuzzETag`), both wired into the CI fuzz job
- **No fuzz target for `metrics/` — deliberate, do not "fix" in an audit** — the package is pure atomic counters: no parsing, no input-dependent branching, no serialization, so a fuzz target has no oracle to violate and no crash surface to find (the counter invariants are pinned by `metrics/metrics_test.go` behavior tests instead (zero-before-events, adopted-tag 304 accounting)). Documented 2026-10-15 to preempt the next fuzz-coverage audit from re-flagging it.
- **`t.Errorf`** for non-fatal, **`t.Fatalf`** for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for server doubles; `roundTripperFunc` stubs for client doubles
- **Shared test helpers** in `server/testutil_test.go` and `client/testutil_test.go`
- **Benchmark discipline** — capture `-benchmem -count=6` baselines under `reports/bench/<date>_<name>.txt` before and after perf-relevant changes (e.g. `2026-09-11_before-lazy-key.txt` / `_after-lazy-key.txt`, `_baseline-post-435.txt`); smoke-run `-count=1` only for quick sanity checks

## Repo Workflow Notes

- **An auto-git daemon commits and re-formats continuously** — status reports and tables get re-touched within minutes of your edit. Always re-read a shared file immediately before re-editing it; never assume remembered file geometry. Observed daemon behaviors: it rewrote the `go 1.27.1` directive to `go 1.27` twice (`969d077`, `6f15f81` — the release-gate parity check catches this class), and it regenerates the buildflow-managed `.gitignore` block, re-adding `go.work`/`go.work.sum` ignores that were removed by hand (2026-09-23: this silently untracked `go.work` and cost a fully red CI run — the durable fix is `!go.work`/`!go.work.sum` negation lines OUTSIDE the marker block, where regeneration cannot touch them).
- **Shared-tree protocol (multiple agent sessions may run in this repo at once)** — check `git log` + `git status` (and the remote) before deep work and again before any edit batch; when a release train is in flight, go docs-only or nothing. Both 2026-09-23 lanes independently worked blind to each other for ~15 tool calls (reports 15:35 d1, 16:40 d1); a start-of-session git-state check would have exposed the parallel lane immediately.
- **`docs/status/` and `docs/planning/` are point-in-time snapshots** — never rewrite their narrative; annotate resolved items inline (strikethrough plus a done-at hash citation) per the docs-health skill. The living backlog is `TODO_LIST.md`, never the reports.
- **Consumer-migration recipe (per go-ecosystem-upgrade sweeps):** before bumping go-etag in a consumer repo, check its version surface first (`flake.nix` inputs, `vendorHash`, CI pins — DiscordSync's drift-guard test exists for exactly this); commit immediately after the gate so the auto-daemon doesn't capture the diff; run `nix build` whenever that repo's `go.sum` changed; verify workspace repos in BOTH modes (hermetic `GOWORK=off` + workspace MVS). Split-era specifics: bump the root and the nested module in ONE `go get` resolution (`go get github.com/larsartmann/go-etag@vX github.com/larsartmann/go-etag/server@vX`) — bumping only the nested module hits the ambiguous-import error because the old root version still contains the nested packages; and RE-CHECK the consumer's `go` directive after `go get` — it can silently rewrite `go 1.27.1` down to `go 1.27` (nsfw-classifier 2026-09-23), which then fails vendored deps declaring a 1.27.1 floor; restore with `go mod edit -go=1.27.1`.
- **Version-train discipline (multi-module):** all internal requires and the go.work replaces move to the new vX.Y.Z TOGETHER, in the same commit. `scripts/pre-release-check.sh` verifies the train is in sync (single internal version everywhere; every require has its matching go.work replace). A consumer importing `…/client` after v0.6.0 pulls client+entitytag only — the composability win the split bought; the root module no longer bundles the other packages, so consumers importing nested paths via `go get github.com/larsartmann/go-etag@vX` need one extra `go mod tidy`/`go get` of the nested module (import paths themselves never changed) — validated by the 2026-09-23 sweep: every consumer's tidy DROPPED the root require entirely.

## Release Conventions

Full multi-module playbook: `docs/release-playbook.md` (single-module history, the bottom-up tag staircase, consumer gates, the v1.0.0 root-deletion plan). Core rules:

- Cut the CHANGELOG before tagging (`chore(release): cut CHANGELOG vX.Y.Z`); run `scripts/pre-release-check.sh` on the exact release commit first.
- Staircase order: entitytag → server → client → metrics → root; each ANNOTATED tag lands only after CI is green on the commit where that module's go.mod finalized.
- v0.x GitHub Releases are Latest, non-prerelease; the consumer gates are proxy.golang.org + sum.golang.org + clean-room `go get` per module tag; pkg.go.dev rendering is not a gate.
- Load the go-release skill (`~/.config/crush/skills/go-release/`) for any release.

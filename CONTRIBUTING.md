# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run the full verification (below) — build, race tests, vet, lint, format
5. Submit a pull request

## Development Setup

Go 1.27+ (go.mod sets the exact floor) and [golangci-lint](https://golangci-lint.run)
v2.13+ are the only requirements. If your local toolchain is older than the
go.mod directive, prefix every command — golangci-lint included, it shells
out to go — with `GOTOOLCHAIN=auto`:

```bash
# Multi-module repo: "./..." spans only the root module, so carry the full
# pattern set (or cd into a module and run plain ./...). The committed go.work
# joins all five modules into one workspace build.
PACKAGES="./... ./client/... ./entitytag/... ./metrics/... ./server/..."

go build $PACKAGES                   # compile everything
go vet $PACKAGES                     # vet
go test -race -count=1 $PACKAGES     # tests with race detection (required: tests use t.Parallel)
golangci-lint run                   # lint the root module
for m in client entitytag metrics server; do (cd $m && golangci-lint run --config ../.golangci.yml); done
golangci-lint fmt                   # format (gofumpt + golines@120 + gci)
golangci-lint config verify    # sanity-check .golangci.yml
```

Releases additionally run `scripts/pre-release-check.sh`, which encodes the
full local gate in one command (everything above, per module, plus erraudit,
the nolint audit, `go mod verify`, replace/pseudo-version bans across every
go.mod, go-directive ↔ CI-pin parity, and a version-train sync check between
go.mod requires and go.work replacements).

Benchmarks (baseline snapshots live in `reports/bench/`; compare with
`-count=6` runs before and after perf-relevant changes):

```bash
go test -run '^$' -bench . -benchmem -count=6 ./...
```

## Project Layout

Five Go modules in one repo, released on a shared version train; a committed
`go.work` joins them for local development and CI.

| Path            | Package       | Module? | Purpose                                                                    |
| --------------- | ------------- | ------- | -------------------------------------------------------------------------- |
| `entitytag/`    | `entitytag`   | yes     | Shared RFC 7232 §2.3 entity-tag domain type (zero dependencies)             |
| `server/`       | `etag`        | yes     | RFC 7232 ETag response middleware (deps: entitytag, go-error-family)        |
| `client/`       | `etagclient`  | yes     | RFC 9111 conditional-GET cache transport (dep: entitytag)                   |
| `metrics/`      | `metrics`     | yes     | Atomic counters for the server's observability hooks (dep: server)          |
| `deprecated.go` | `etag` (root) | root    | Deprecated alias shim; removed at v1.0.0 (dep: server)                      |
| `docs/`         | —             | —       | `rfc9111-conformance.md` (MUST-by-MUST table), planning and status reports  |

## Constraints Worth Knowing Before Your First Edit

- **Dependencies are locked down.** The standard library, sibling modules in
  this repo, and `go-error-family` only. Do not add third-party imports.
- **Every struct field must be set** (`exhaustruct_v5`). Use
  `//nolint:exhaustruct_v5` with a justification for intentional zero values.
- **No dynamic `errors.New`/`fmt.Errorf` where a sentinel fits** (`err113`).
- **Strict whitespace** (`wsl_v5`) — run `golangci-lint fmt` after editing.
- **Canonical header keys** (`canonicalheader`) — write `"Etag"`, `"X-Ratelimit-Remaining"`,
  etc. in Go 1.26 canonical form; build test headers with `Header.Set`, never
  map literals with non-canonical keys.
- **No magic numbers** (`mnd`), **no named returns**, **`t.Parallel()` in
  every test**, comments end with a period (`godot`).

## Testing Conventions

- Tests live in the same package they test (`etag`, `etagclient`) and use
  plain `testing` — no assertion libraries.
- Spec tests (`client/spec_test.go`, `server/etag_bdd_test.go`) cite the RFC
  section they pin in the doc comment; one behavior per test.
- Integration tests (`client/integration_test.go`, `server/integration_test.go`)
  exercise real `httptest` servers and canonical wire behavior.
- Fuzz targets exist for every parser — add one when you add a parser.
- Coverage expectation: keep it at the current bar (~98%+ total); the suite
  includes statement-coverage auditing (`go tool cover -func`).

## CI

`.github/workflows/ci.yml` runs four jobs on every push and PR:

1. **Test** — build, vet, `go test -race`, coverage artifact (full pattern
   set across all five modules via the committed go.work).
2. **Lint** — golangci-lint at the root plus one run per nested module
   (pinned version; must match `.golangci.yml`'s needs).
3. **Fuzz** — 30s per fuzz target (parser targets run under `./entitytag/...`).
4. **Vulnerabilities** — govulncheck over the full pattern set.

The toolchain is pinned via `GOTOOLCHAIN` in the workflow env; keep it in
exact sync with the `go` directive in every go.mod (all five must agree —
`scripts/pre-release-check.sh` enforces this).

## Reporting Issues

Please use GitHub Issues to report bugs or request features. For conformance
questions, cite the specific RFC section and include a minimal reproduction
(a stub `http.RoundTripper` or `httptest` server is usually enough).

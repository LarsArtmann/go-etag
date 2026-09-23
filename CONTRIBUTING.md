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
go build ./...          # compile everything
go vet ./...            # vet
go test -race -count=1 ./...   # tests with race detection (required: tests use t.Parallel)
golangci-lint run       # lint (grep out 'level=warning' lines; must end at 0 issues)
golangci-lint fmt       # format (gofumpt + golines@120 + gci)
golangci-lint config verify    # sanity-check .golangci.yml
```

Releases additionally run `scripts/pre-release-check.sh`, which encodes the
full local gate in one command (everything above plus erraudit, the nolint
audit, `go mod verify`, replace/pseudo-version bans, and go-directive ↔
CI-pin parity).

Benchmarks (baseline snapshots live in `reports/bench/`; compare with
`-count=6` runs before and after perf-relevant changes):

```bash
go test -run '^$' -bench . -benchmem -count=6 ./...
```

## Project Layout

| Path            | Package       | Purpose                                                                    |
| --------------- | ------------- | -------------------------------------------------------------------------- |
| `server/`       | `etag`        | RFC 7232 ETag response middleware                                          |
| `client/`       | `etagclient`  | RFC 9111 conditional-GET cache transport                                   |
| `entitytag/`    | `entitytag`   | Shared RFC 7232 §2.3 entity-tag domain type                                |
| `metrics/`      | `metrics`     | Atomic counters for the server's observability hooks                       |
| `deprecated.go` | `etag` (root) | Deprecated alias shim; removed at v1.0.0                                   |
| `docs/`         | —             | `rfc9111-conformance.md` (MUST-by-MUST table), planning and status reports |

## Constraints Worth Knowing Before Your First Edit

- **Dependencies are locked down.** `depguard` allows the standard library,
  this module, and `go-error-family` only. Do not add third-party imports.
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

1. **Test** — build, vet, `go test -race`, coverage artifact.
2. **Lint** — golangci-lint (pinned version; must match `.golangci.yml`'s needs).
3. **Fuzz** — 30s per fuzz target.
4. **Vulnerabilities** — govulncheck.

The toolchain is pinned via `GOTOOLCHAIN` in the workflow env; keep it in
sync with the `go` directive in `go.mod`.

## Reporting Issues

Please use GitHub Issues to report bugs or request features. For conformance
questions, cite the specific RFC section and include a minimal reproduction
(a stub `http.RoundTripper` or `httptest` server is usually enough).

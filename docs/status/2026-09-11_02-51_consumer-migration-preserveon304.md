# Consumer Migration: `PreserveOn304` → `FreshenOn304` (2026-09-11 02:51)

Executed TODO_LIST item 1 via the go-ecosystem-upgrade flow: bumped all six
in-house direct consumers from `go-etag` v0.2.0 to v0.3.0, migrated code where
needed, verified build + test + `go mod verify` per repo, committed per repo.

## Evidence-based correction to the item's premise

A full-enumeration grep (`rg PreserveOn304 --no-ignore-vcs` across
`~/projects`, excluding vendor) found exactly ONE consumer using the removed
API: `go-github-kit/etag.go`. The other five import only `go-etag/server`,
whose API is unchanged in v0.3.0 — they were never compile-broken; they needed
version-bump + verification only. The report `2026-09-11_02-08` item 2 claim
("the six still compile against `PreserveOn304`") overstated the blast radius.

## Per-repo results

| Consumer                                 | Change                                                                                                      | Build/Test                                 | Commit                    |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------------------- | ------------------------------------------ | ------------------------- |
| `go-github-kit`                          | v0.2.0→v0.3.0 + `PreserveOn304: []string{...}` → `FreshenOn304: FreshenFields(...)` (1:1 whitelist mapping) | pass                                       | go-github-kit `f3ece58`   |
| `httputil`                               | v0.2.0→v0.3.0 (go.work renamed during `go get` per F2; vendor refreshed)                                    | pass (workspace mode)                      | httputil `ff724e0`        |
| `cqrs-htmx/examples/middleware-showcase` | v0.2.0→v0.3.0 (repo go.work renamed during operation; vendor refreshed)                                     | pass (hermetic GOWORK=off)                 | cqrs-htmx `533acf40`      |
| `DiscordSync`                            | v0.2.0→v0.3.0 + flake.nix/flake.lock go-etag input repinned to v0.3.0 tag commit `fe5dede`                  | pass except 1 pre-existing failure (below) | DiscordSync `42c9d06f`    |
| `nsfw-classifier`                        | v0.2.0→v0.3.0                                                                                               | pass                                       | nsfw-classifier `f910b03` |
| `library-policy`                         | v0.2.0→v0.3.0                                                                                               | pass                                       | library-policy `9c47f43`  |

The `PreserveOn304` mapping used (from the old field doc, commit `9204885`):
`nil` → RFC default (field omitted), non-empty whitelist → `FreshenFields(...)`,
empty non-nil slice → `FreshenNone()`. Only the whitelist case occurs in the
wild (go-github-kit).

## Notes and pre-existing failures

- **DiscordSync pre-existing failure (unrelated, unchanged by this work):**
  `TestEvaluateHealthChecks_LabelsResults` in `cmd/discordsync` fails on
  master before and after the bump (baseline comparison per F11).
- **DiscordSync drift guard did its job:** after the go.mod bump,
  `TestCheckFlakePins_NoSilentSkip` reported `DRIFT: go-etag` because
  flake.nix still pinned the v0.2.0 tag commit. Fixed by repinning to
  `fe5dede` + `nix flake lock --update-input go-etag`; guard passes.
  No other repo pins go-etag in a flake (checked all six).
- **cqrs-htmx root module untouched:** go-etag is `// indirect` there
  (via published httputil tag) with no source usage; its only direct go-etag
  require is the middleware-showcase example, which is migrated. The
  pre-commit train checker confirms 43 train-lag entries remain (see
  follow-up below) but none unpublished.
- **Zero residual references:** post-migration grep finds no
  `PreserveOn304` in any consumer source and no direct v0.2.0 requires.

## Follow-ups discovered (tracked in TODO_LIST)

1. ~~**httputil release needed:** cqrs-htmx's 43 train-lag entries (and any
   other httputil consumers) stay on go-etag v0.2.0 until httputil cuts a
   release carrying v0.3.0. Bumping httputil's master pin (done here) is not
   visible to consumers until tagged (F17). Needs the go-release flow.~~ done 2026-09-11 — httputil v1.0.1 released carrying go-etag v0.3.1 transitively (report 2026-09-11_05-09 §a.1)
2. **BuildFlow hook noise in cqrs-htmx:** the pre-commit run reported
   transient `/tmp` exhaustion (`No space left on device` tmpfs) and many
   pre-existing gomod-check findings; unrelated to this migration but worth
   a look. _Foreign repo; its three standalone failures were verified passing
   on 2026-09-18 (report 2026-09-18_19-49 §a)._

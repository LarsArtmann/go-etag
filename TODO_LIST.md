# TODO List

Short- and mid-term actionable work, ranked by impact. Open items only — completed
items live in `CHANGELOG.md`. Every item cites evidence (code) and origin (report).

| # | Priority | Task                                                                                                                                                                               | Effort | Evidence / Origin                                                    |
| - | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------- |
| 1 | Critical | **Migrate the six in-house consumers (`httputil`, `go-github-kit`, `cqrs-htmx`, `DiscordSync`, `nsfw-classifier`, `library-policy`) off the removed `PreserveOn304` onto `FreshenOn304`** — they are compile-broken against v0.3.0. Run the go-ecosystem-upgrade flow per repo. | M      | `CHANGELOG.md` `[0.3.0]` BREAKING; report `2026-09-11_02-08` item 2 |

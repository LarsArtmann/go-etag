# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Nothing yet.

### Changed

- Nothing yet.

### Fixed

- Nothing yet.

## [0.1.1] - 2026-08-14

### Changed

- `Middleware` is now a type alias (`type Middleware = func(http.Handler) http.Handler`) instead of a distinct named type, making it fully interchangeable with the standard middleware signature without an explicit conversion step. Backwards compatible — existing code compiles unchanged.

### Fixed

- Replaced the proprietary LICENSE with the MIT License.

## [0.1.0] - 2026-08-07

### Added

- RFC 7232 HTTP ETag middleware that buffers GET/HEAD response bodies, computes entity-tags, and handles `If-None-Match` conditional requests with `304 Not Modified` responses.
- Dedicated `ETag` value type with `Strength` enum (`Strong`/`Weak`) encoding RFC 7232 §2.1 validator semantics directly in the type system.
- `StrongEqual` and `WeakEqual` methods implementing both RFC 7232 §2.3.2 comparison functions.
- `ParseETag` and `ParseETagList` for parsing entity-tags from wire format (quote-aware, handles escaped quotes and commas inside quotes).
- `MatchesIfNoneMatch` (weak comparison) and `MatchesIfMatch` (strong comparison) exported helpers for manual conditional-request evaluation.
- `SkipIfPresent` config field to respect handler-set ETags instead of overwriting them.
- `Skip` config predicate to exclude specific routes from ETag processing (SSE, large downloads, streaming).
- `HashFunc` accepts `func([]byte) string` for arbitrary opaque-tag values (not just uint64 hashes); defaults to FNV-64a.
- RFC 7230 §3.3 compliance: HEAD responses set `Content-Length` without sending a body.
- RFC 7232 §4.1 compliance: 304 responses strip `Content-Length`.
- Classified errors via `go-error-family` for retry-aware observability with `OnError` callback hook.
- BDD-style spec suite (`etag_bdd_test.go`) mapping RFC 7232 sections to behavioral expectations.
- Fuzz tests for `ParseETag` and `ParseETagList` verifying no panics and round-trip integrity on arbitrary input.

[Unreleased]: https://github.com/larsartmann/go-etag/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/larsartmann/go-etag/releases/tag/v0.1.1
[0.1.0]: https://github.com/larsartmann/go-etag/releases/tag/v0.1.0

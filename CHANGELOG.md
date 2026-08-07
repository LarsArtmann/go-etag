# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Dedicated `ETag` value type with `Strength` enum (`Strong`/`Weak`) encoding RFC 7232 §2.1 validator semantics directly in the type system
- `StrongEqual` and `WeakEqual` methods implementing both RFC 7232 §2.3.2 comparison functions
- `ParseETag` and `ParseETagList` for parsing entity-tags from wire format
- `MatchesIfNoneMatch` (weak comparison) and `MatchesIfMatch` (strong comparison) exported helpers for conditional-request evaluation
- `SkipIfPresent` config field to respect handler-set ETags instead of overwriting them
- `Skip` config predicate to exclude specific routes from ETag processing
- `HashFunc` now accepts `func([]byte) string` for arbitrary opaque-tag values (not just uint64 hashes)
- RFC 7230 §3.3 compliance: HEAD responses now set Content-Length without sending a body
- RFC 7232 §4.1 compliance: 304 responses now strip Content-Length
- BDD-style spec suite (`etag_bdd_test.go`) mapping RFC 7232 sections to behavioral expectations

### Changed

- **Breaking:** middleware constructor renamed from `ETag()` to `New()`
- **Breaking:** `ETagConfig.Weak bool` replaced with `ETagConfig.Strength Strength`
- **Breaking:** `ETagConfig.HashFunc func([]byte) uint64` changed to `func([]byte) string`
- Zero-value `ETagConfig{}` now clamps `MaxBufferSize` to 1 MB (previously caused unbounded buffering)
- `wrapper.go` internal fields renamed: `wroteHeader` → `headerBuffered`, `headerWritten` → `headerCommitted`
- `errors.go` `RegisterErrorClassifications` trimmed to only sentinels this library returns (`ErrNotSupported`, `ErrAbortHandler`)

### Fixed

- Zero-value `ETagConfig{}` no longer causes unbounded memory buffering (B1)
- HEAD requests no longer send a response body (B2, RFC 7230 §3.3)
- 304 responses no longer leak `Content-Length` header (B3, RFC 7232 §4.1)

## [0.1.0] - 2026-01-01

### Added

- Initial release

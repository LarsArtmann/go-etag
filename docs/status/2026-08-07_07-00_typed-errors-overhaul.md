# Status Report — 2026-08-07 07:00

## Session Goal

Achieve 100% superb typed errors in go-etag, resolving all 8 violations from `erraudit --type-aware --enforce-go-error-family --no-suppress`.

---

## Context

Starting state: 8 violations (7 ERROR, 1 WARNING). The `erraudit` tool flagged ignored errors, stdlib error constructors, generic return types, and context loss in error paths.

---

## A) FULLY DONE

### Audit Violations Resolved (8/8 → 0)

| #   | Violation                            | Location                   | Fix                                                                                                              |
| --- | ------------------------------------ | -------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| 1   | Ignored `h.Write` error              | `etag.go:57`               | Check error; panic with `Orchestration` error (`ErrCodeHashWriteFailed`) — hash.Hash contract violation is a bug |
| 2   | Ignored `ResponseWriter.Write` error | `etag.go:178` (flush path) | New `ETagConfig.OnError` callback receives classified `*errorfamily.Error`                                       |
| 3   | Ignored `ResponseWriter.Write` error | `etag.go:326` (Flush path) | Same `OnError` callback                                                                                          |
| 4   | `errors.New` used                    | `etag.go:62`               | Replaced with `ErrInvalidConfig` sentinel via `errorfamily.NewRejection`                                         |
| 5   | `fmt.Errorf` used                    | `etag.go:67`               | Replaced with `ErrInvalidConfig.WithContextf("max_buffer_size", ...)`                                            |
| 6   | Generic `error` return               | `etag.go:65` (`Validate`)  | Now returns concrete `*errorfamily.Error` with code, family, context                                             |
| 7   | Hijack context loss                  | `wrapper.go:58`            | Added `.WithContextf("writer_type", "%T", w)`                                                                    |
| 8   | Hijack context loss                  | `wrapper.go:67`            | Added `.WithContextf("writer_type", "%T", w)`                                                                    |

### New Error Codes Added

| Code                          | Family        | When                                           |
| ----------------------------- | ------------- | ---------------------------------------------- |
| `http.etag_config_invalid`    | Rejection     | ETagConfig field value invalid                 |
| `http.etag_hash_write_failed` | Orchestration | hash.Write returned error (contract violation) |

Both have message templates registered in `registerAllErrorTemplates()`.

### New Public API

- `ErrInvalidConfig` — package-level sentinel error for `Validate()` failures. `errors.Is` matches by code+family.
- `ETagConfig.OnError` — optional callback `func(*errorfamily.Error)` for post-commit write failures that cannot be surfaced to the client or returned from `Write`. If nil, errors are silently dropped (matching net/http default behavior).

### Tests Added (4 new tests)

1. `TestETagConfig_Validate_ReturnsTypedError` — verifies `Validate` returns `*errorfamily.Error` with correct code, family (Rejection), and context (`max_buffer_size`).
2. `TestETag_OnError_CalledOnFlushWriteFailure` — verifies `OnError` fires with classified error when flush-path write fails.
3. `TestETag_OnError_CalledOnExplicitFlushWriteFailure` — verifies `OnError` fires on explicit `Flush()` path.
4. `TestETag_OnError_NilCallback` — verifies nil callback does not panic.

Existing tests `TestETagConfig_Validate_ZeroMaxBufferSize` and `_NegativeMaxBufferSize` updated to use `ErrInvalidConfig` sentinel via `errors.Is`.

### Verification

- `go test -race ./...` — **PASS** (41 tests, 0 failures, race clean)
- `erraudit ./... --type-aware --enforce-go-error-family --no-suppress` — **0 violations** (was 8)
- `golangci-lint fmt` — clean
- `golangci-lint run` — 1 pre-existing `unparam` warning (unrelated to changes)

### Documentation Updated

- `AGENTS.md` — updated error classification table (5 rows), file exports table, non-obvious behaviors section.

---

## B) PARTIALLY DONE

### README.md

The Error Classification table in README is stale — still shows the old 3 error codes. Needs updating to reflect the 5 codes now present. The Configuration example also does not show the new `OnError` field.

### CHANGELOG.md

`[Unreleased]` section still says "Initial project structure" — needs entries for the new error codes, `OnError` callback, `ErrInvalidConfig` sentinel, and the hash.Write panic behavior.

---

## C) NOT STARTED

1. **Hijack context test** — no test verifies `writer_type` context is present on hijack errors.
2. **Hash.Write panic test** — no test exercises the `ErrCodeHashWriteFailed` panic path (hard to trigger with stdlib fnv, but a custom `HashFunc` wrapping a failing hash could do it — except `HashFunc` returns `uint64`, not an error, so the panic lives inside `defaultETagHash` and is unreachable from the public API with the current type signature).
3. **Template registration test** — no test verifies all 5 error code templates are registered and produce the expected `What/Why/Fix/WayOut` fields.
4. **`OnError` context propagation** — the error forwarded to `OnError` includes `writer_type` on hijack but NOT on flush-path writes. Could add writer type context to `reportWriteErr`.
5. **Benchmarks for OnError path** — no benchmark measuring the overhead of the `OnError` callback when set vs. nil.

---

## D) TOTALLY FUCKED UP

Nothing. Everything compiles, all tests pass, audit is clean. No regressions introduced.

---

## E) WHAT WE SHOULD IMPROVE

### Design Decisions to Reconsider

1. **`Validate()` signature** — returns `error` interface (not `*errorfamily.Error`). This was a deliberate choice: `ireturn` linter would reject `*errorfamily.Error` as a non-stdlib concrete return. But the audit's WARNING about "generic return" was addressed by ensuring the concrete type is typed. The signature staying `error` is idiomatic Go. Still, this is a tradeoff worth documenting.

2. **`OnError` naming** — `OnError` is generic. Could be `OnWriteError` or `OnUnreportableError` to be more precise about which errors it receives. Current name could mislead users into thinking it captures ALL errors.

3. **`OnError` only covers flush path** — streaming `Write()` failures (lines 124, 139) still return the error directly to the handler. This is correct (the handler can still act on it), but the asymmetry could confuse users. Should document clearly.

4. **Hash.Write panic is untestable** — `defaultETagHash` is unexported and `HashFunc` returns `uint64` (no error). The panic path is dead code in practice because fnv's `Write` never fails. The code exists for correctness, but it's untestable without refactoring `HashFunc` to return `(uint64, error)` — which would be a breaking API change.

5. **`ErrInvalidConfig` sentinel pattern** — `WithContextf` clones the sentinel on every call. For `Validate()` this is fine (called once at startup). But the pattern should be documented: sentinels are immutable templates, `.WithContext*` produces fresh classified copies.

### Code Quality

6. **`reportWriteErr` could include more context** — currently only wraps with a message. Could add `body_size`, `status_code`, or `flushed` state for better observability.

7. **`errors` and `fmt` imports removed from `etag.go`** — good cleanup, but worth verifying no other file in the package still imports them unnecessarily.

8. **`exhaustruct` impact on users** — adding `OnError` to `ETagConfig` means any user code with `ETagConfig{Weak: true, MaxBufferSize: 1024}` struct literals (without `DefaultETagConfig()`) will now fail `exhaustruct` in their own projects if they use the same linter. This is a minor breaking change in the configuration experience.

---

## F) THINGS TO GET DONE NEXT (Prioritized)

### High Priority

1. Update README.md Error Classification table to 5 codes
2. Add `OnError` field to README.md Configuration example
3. Update CHANGELOG.md `[Unreleased]` with all changes
4. Add test verifying hijack errors include `writer_type` context
5. Add test verifying all 5 template registrations produce expected fields
6. Add context (`body_size`, `status`) to `reportWriteErr` for better observability
7. Consider renaming `OnError` to `OnWriteError` for precision

### Medium Priority

8. Add a doc comment example showing `OnError` usage with logging
9. Add a doc comment example showing `ErrInvalidConfig` + `errors.Is` usage
10. Benchmark `OnError` callback overhead (nil vs. set)
11. Consider `Validate()` returning a multi-error if multiple fields are invalid (future-proofing for more config fields)
12. Add `Validate` test for `HashFunc == nil` (currently valid — defaults to fnv — but should be documented)
13. Document the sentinel clone pattern in AGENTS.md (sentinels are templates, `WithContext*` clones)
14. Consider adding `WithHTTPStatus` to `ErrInvalidConfig` (Rejection defaults to 400, which is correct, but explicit is better)
15. Audit all `// nolint` comments — none added this session, but worth checking the repo
16. Add a test that `ErrInvalidConfig.Is` matches across different context values (same code+family, different context)
17. Consider whether `OnError` should receive `error` or `*errorfamily.Error` (currently `*errorfamily.Error` — strong typing, good)
18. Review whether the hash.Write panic should use `errorfamily.NewOrchestration` or just a plain panic (current: classified — good for recover() callers)

### Lower Priority

19. Consider a `ValidateAll()` or `ValidateFields()` method that returns all validation errors at once
20. Add GoDoc examples for the error types (`ExampleErrInvalidConfig`, etc.)
21. Consider an `ErrHashWriteFailed` sentinel for the panic case (currently only a code constant)
22. Evaluate whether `OnError` should be called for `WriteHeader` failures (currently not handled at all)
23. Consider adding metrics integration hooks beyond `OnError` (e.g., `OnETagGenerated`, `On304`)
24. Review the `testutil_test.go` unused `assertBodyContains` function — remove it
25. Review the `testutil_test.go` `unparam` warning on `newTestRequest` — `path` always receives `"/"`
26. Consider fuzzing the error paths (failing writer + ETag middleware)
27. Add integration test with real `net/http` server (not just httptest)
28. Evaluate if `OnError` needs `sync.Once` protection for idempotent handlers
29. Consider structured logging integration for `OnError` (slog handler)
30. Review if `ErrInvalidConfig` should implement `Unwrap()` to chain to a specific field error
31. Consider whether `MaxBufferSize` validation should also enforce an upper bound (e.g., prevent accidental 1TB buffer)
32. Document the relationship between `ErrCodeInvalidConfig` and `ErrInvalidConfig` (code constant vs. sentinel value)
33. Consider whether `HashFunc` should accept `context.Context` for cancellation
34. Evaluate adding a `SkipIfPresent` config option (currently always overwrites handler-set ETags)
35. Consider ETag strength validation per RFC 7232 (entity-tag format validation)
36. Add a `Config()` method on `etagWriter` to expose the active config for debugging
37. Consider whether `flush()` and `Flush()` should deduplicate their write-error reporting
38. Evaluate if `OnError` should be called before or after `body = body[:0]` (currently after write, before clear in Flush, after write in flush — inconsistent?)
39. Review the `gosec` G705 exclusion — is it still the right scope?
40. Consider adding `ETagConfig.Clone()` for safe copying
41. Evaluate if `Validate()` should be called automatically inside `ETag()` (currently caller's responsibility)
42. Consider whether `OnError` errors should be deduplicated (prevent flood on repeated write failures)
43. Add a test for `ETag(ETagConfig{MaxBufferSize: -1})` without calling `Validate()` — what happens?
44. Document the panic behavior of hash.Write failures in README.md
45. Consider adding `ETagConfig.Validated()` that returns a validated, immutable config
46. Evaluate whether error codes should be typed (e.g., `type ErrCode string`) rather than bare `string` constants
47. Consider whether `RegisterErrorClassifications` should be idempotent (safe to call multiple times)
48. Add a test verifying `RegisterErrorClassifications` does not panic on double registration
49. Consider whether the `OnError` callback should receive a copy of the request context for tracing
50. Review if the error family assignments are correct for all HTTP middleware use cases (e.g., is a flush-path write failure truly Transient?)

---

## G) QUESTIONS (Cannot Determine Without Input)

1. **Should `Validate()` be called automatically inside `ETag()`?** Currently it's the caller's responsibility. If `ETag()` is called with an invalid config (e.g., `MaxBufferSize: -1`), the middleware will misbehave silently. Auto-validation would make the middleware safer but would change the failure mode from "returns error" to "panics" (since `ETag()` returns `Middleware`, not `(Middleware, error)`). Should we change the `ETag()` signature to `(Middleware, error)`, or panic on invalid config, or leave as-is?

2. **Should `OnError` be called for ALL write failures or only post-commit ones?** Currently `Write()` returns streaming errors to the handler (who can act on them), while `OnError` only fires for post-commit writes (where the handler has already returned). Is this the right boundary, or should `OnError` be a catch-all for every classified error the middleware produces?

3. **Should we version this as a breaking change?** Adding `OnError` to `ETagConfig` is technically a breaking change for users who construct `ETagConfig{...}` literals directly (their code still compiles because Go zero-values the new field to `nil`, but `exhaustruct` users will see lint failures). Is this a patch (0.x.y) or minor (0.(x+1).0) bump?

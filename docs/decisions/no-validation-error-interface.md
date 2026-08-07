# ADR: No `ValidationError` Interface

**Status:** Accepted  
**Date:** 2026-08-07  
**Decider:** Lars Artmann

---

## Context

During the typed-errors overhaul, `Validate()` was migrated from ad-hoc sentinel
errors (`errNonPositiveMaxBufferSize` + `fmt.Errorf`) to the
`go-error-family` classification system. It now returns a clone of the
`ErrInvalidConfig` sentinel — a `*errorfamily.Error` classified as `Rejection`
with code `http.etag_config_invalid` and diagnostic context
(`max_buffer_size`).

A natural question arose: **should we introduce a dedicated `ValidationError`
interface** (e.g. with `Field() string`, `Value() any` methods) to give
callers a stronger contract than the bare `error` return type?

## Decision

**Do not add a `ValidationError` interface.** The `go-error-family` system
already provides every capability such an interface would promise.

## Rationale

### 1. The classification already exists

`ErrInvalidConfig` is `errorfamily.NewRejection(...)` with a stable code.
Callers have full structured access today:

| Need | Already provided by |
|------|---------------------|
| "Is this a validation error?" | `errors.Is(err, ErrInvalidConfig)` or `family == Rejection` |
| "Which field/value failed?" | `.ErrorContext()["max_buffer_size"]` |
| "Is it retryable?" | `Rejection` family → no |
| "Full structured type" | `errors.AsType[*errorfamily.Error](err)` |

A `ValidationError` interface would be a **third redundant taxonomy** on top of
code + family.

### 2. An interface with one implementation is dead weight

`Validate()` checks exactly one field (`MaxBufferSize`) and returns one
sentinel clone. An interface earns its existence at ≥2 implementations with
genuinely polymorphic behavior. One validation rule does not meet that bar — it
is premature generalization.

### 3. The name carries the anemic-model smell

`ValidationError` says "an error about validation" without saying *what* is
invalid. `ErrInvalidConfig` / `http.etag_config_invalid` is precise. Worse,
adding `Field() string` / `Value() any` methods to make the interface
non-ceremonial would reinvent what the context map already carries — and it
breaks down the moment *multiple* fields are invalid (which field does
`Field()` return?).

### 4. The `error` return signature is deliberately weak

`Validate()` returns `error`, not `*errorfamily.Error`, because the `ireturn`
linter forbids concrete return types. The *concrete runtime type* is strongly
typed (`*errorfamily.Error`), satisfying the type-awareness audit. Introducing
a marker interface purely to "strengthen" the signature would be solving a lint
concern with architecture — backwards.

## When to revisit

Revisit this decision **only if all of these become true**:

1. The config grows many independently-validatable fields, **and**
2. Callers need to branch on *which field failed* in a way `ErrorContext()`
   keys cannot express (e.g. compile-time exhaustiveness), **and**
3. The branching logic is common enough to justify a shared contract.

Until then, the `go-error-family` system **is** the validation-error system.
Adding a parallel layer fragments the clean "everything is
`*errorfamily.Error`" story for no capability gain.

## References

- `errors.go` — `ErrInvalidConfig` sentinel, `ErrCodeInvalidConfig` constant
- `etag.go:82` — `Validate()` implementation
- [go-error-family](https://github.com/larsartmann/go-error-family) —
  `Rejection` family, `WithContextf`, `*Error.Is` matching by code + family

# RFC 9111 Conformance — `client` package

Requirement-by-requirement account of how the conditional-GET transport
(`etagclient.Transport`) meets RFC 9111 (HTTP Caching). Scope: behavior the
transport implements. Each row cites the pinning test in `client/` (spec
tests cite the section they pin) or the documenting code.

Status legend:

| Mark      | Meaning                                                       |
| --------- | ------------------------------------------------------------- |
| Done      | Implemented and pinned by tests                               |
| Deviation | Deliberately different; documented contract                   |
| N/A       | Not applicable to a process-private always-revalidating cache |

## §3 Storing Responses

| §        | Requirement (abridged)                                                                                                           | Status | Evidence                                                                                     |
| -------- | -------------------------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------------------------------- |
| §3       | MUST NOT store any part of a response carrying `Cache-Control: no-store`                                                         | Done   | `TestSpecNoStoreResponseIsNeverCached`                                                       |
| §5.2.2.5 | Request `no-store`: MUST NOT store this request or any response to it                                                            | Done   | `TestSpecRequestNoStoreBypassesTheCache`                                                     |
| §3, §3.1 | MUST NOT store partial content; only complete 200s with a validator are stored                                                   | Done   | `TestSpecOnly200WithValidatorIsStored`, `TestSpec200WithoutValidatorIsNeverStored`           |
| §3.1     | MUST NOT store hop-by-hop fields or fields the `Connection` header lists                                                         | Done   | `TestSpecStoredHeaderShedsHopByHopFields`                                                    |
| §3.2     | Stored-response updates follow the §3.2 replace-and-append rules                                                                 | Done   | `TestSpecFresheningWearsThe304sValues`, `TestSpecFresheningAddsFieldsTheStoredResponseLacks` |
| §3.2     | `Content-Length` and `Content-Range` are excepted from updates; `Content-Encoding` when the stored body is transparently decoded | Done   | `TestSpecFresheningSkipsExceptedFields`                                                      |

## §4 Constructing Cache Keys

| §    | Requirement (abridged)                                                          | Status    | Evidence                                                                                                                                                                                                                             |
| ---- | ------------------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| §4.1 | Cache key comprises the request target and the selecting header fields (`Vary`) | Deviation | `Vary` is not negotiated; the key is `[Options.KeyFunc]` (URL by default). Documented in `client/doc.go` ("Vary") with the `KeyFunc` escape hatch; `TestRoundTripSeparatesCredentialsViaKeyFunc` pins the credential-scoping pattern |

## §4.3 Validation

| §      | Requirement (abridged)                                                                                                               | Status | Evidence                                                                           |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------ | ------ | ---------------------------------------------------------------------------------- |
| §4.3.1 | Send conditional requests using stored validators                                                                                    | Done   | Validator injection on GET; `TestSpecWeakFormOfStoredValidatorIsAdopted`           |
| §4.3.4 | 304: update the stored response's fields per §3.2; surface them on the rebuilt response                                              | Done   | `TestSpecFresheningWearsThe304sValues`, `TestSpecFresheningUpdatesStoredValidator` |
| §4.3.4 | A 304 only updates stored responses it actually validated (validator filtering)                                                      | Done   | `TestSpecMismatched304ValidatorIsNotAdopted`                                       |
| §4.3.4 | Freshened `Age` never runs backwards across revalidations                                                                            | Done   | `TestSpecAgeNeverRunsBackwardsAcrossRevalidations`                                 |
| §4.3.4 | `no-store` on a 304 does not block updating the stored response (an update is not storage)                                           | Done   | `TestSpecNoStoreOn304DoesNotBlockRebuild`                                          |
| §4.3.5 | HEAD 200: update stored responses whose validators (ETag weakly, Last-Modified exactly) and `Content-Length` match, using §3.2 rules | Done   | `TestSpecHeadFreshening`, `TestSpecHeadFresheningPersistsToTheStore`               |
| §4.3.5 | HEAD 200 otherwise: consider the stored response stale (unprovable identity included)                                                | Done   | `TestSpecHeadFreshening` (mismatch and no-comparable-validator cases)              |
| §4.3.5 | `no-store` on a HEAD leaves stored entries alone                                                                                     | Done   | `TestSpecHeadNoStoreLeavesTheEntryAlone`                                           |
| —      | HEAD is never answered from the cache and never made conditional (body ownership)                                                    | Done   | `TestSpecHeadIsNeverConditional`, `TestSpecHeadErrorAndCacheMissesPassThrough`     |

## §4.4 Invalidation

| §    | Requirement (abridged)                                                                                     | Status | Evidence                                     |
| ---- | ---------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------- |
| §4.4 | MUST invalidate the target URI when a non-error (2xx/3xx) response to an unsafe request method is received | Done   | `TestSpecUnsafeMethodInvalidatesEntry`       |
| §4.4 | Invalidation is scoped to the invalidated URI; error responses keep the entry                              | Done   | `TestSpecInvalidationIsScopedToTheTargetURI` |

## §5.1 Age

| §    | Requirement (abridged)                                                                                   | Status | Evidence                                                                                |
| ---- | -------------------------------------------------------------------------------------------------------- | ------ | --------------------------------------------------------------------------------------- |
| §5.1 | Age is the signal that a response was not generated or validated by the origin; it must reach the caller | Done   | `TestSpecStale200SurfacesAgeToCaller`; Age is relayed verbatim and freshened per §4.3.4 |

## §5.2.2 Cache Directives

| §        | Requirement (abridged)                                         | Status | Evidence                                                                                                                                                |
| -------- | -------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| §5.2.2.2 | Response `no-cache`: reusable only after successful validation | Done   | By construction — the transport revalidates before every replay; `TestSpecNoCacheResponseServesViaRevalidation`                                         |
| §5.2.2.5 | Response `no-store`: MUST NOT store                            | Done   | `TestSpecNoStoreResponseIsNeverCached` (quote-aware, case-insensitive parsing)                                                                          |
| §5.2.2.5 | Request `no-store`: MUST NOT store the request or its response | Done   | `TestSpecRequestNoStoreBypassesTheCache`                                                                                                                |
| §5.2.2.3 | `max-age`/freshness lifetimes govern reuse without validation  | N/A    | The transport never serves a stored body without revalidating, so freshness lifetimes are never consulted (strictly stronger than the spec's allowance) |
| §5.2.2.6 | `private`: shared caches MUST NOT store                        | N/A    | The cache is process-private in-memory state, not a shared cache                                                                                        |

## Interpretation decisions

Judgment calls where the RFC leaves room, recorded so future audits and issue
triage do not re-litigate settled choices.

| Decision                                                     | Ruling                                                                                                                                                                                                                               | Pinned by                                                             |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------- |
| A 304 carrying `no-store` still updates the stored response  | An update is not storage: §5.2.2.5 forbids storing the 304, not updating a response already validly stored; refusing would strand a validated entry with a dead validator                                                            | `TestSpecNoStoreOn304DoesNotBlockRebuild`                             |
| A `no-store` HEAD neither freshens nor invalidates the entry | Neutrality on weaker evidence: §4.3.5 makes the update optional and §3 forbids storage, and a HEAD gives no signal the representation changed, so we do nothing — contrast the 304, which is explicit proof the entry is still valid | `TestSpecHeadNoStoreLeavesTheEntryAlone`                              |
| A HEAD 200 that cannot prove identity marks the entry stale  | §4.3.5 offers update-or-stale; on validator mismatch or no comparable validator we invalidate, because the HEAD may describe a different representation and keeping the entry risks replaying replaced content                       | `TestSpecHeadFreshening` (mismatch and no-comparable-validator cases) |
| `Options.FreshenOn304` does not govern §4.3.5 HEAD updates   | The option's contract is 304-scoped by name; HEAD updates apply the §3.2 rules in full so the option cannot silently weaken origin-provided metadata                                                                                 | `freshenFromHead` applies `freshenedHeader` unconditionally           |

## Known limitations (documented, not violations)

- `Vary` is not negotiated — see §4.1 Deviation above and the credential
  warning in `client/doc.go`.
- One entry per cache key; a 304 updates only that entry (single stored
  response per key).
- `Age` persists when a 304 omits it (last known value is worn by rebuilds);
  see "Stale edge caches and Age" in `client/doc.go`.
- `Options.FreshenOn304` governs 304 freshening only; §4.3.5 HEAD updates
  always apply the §3.2 rules in full.

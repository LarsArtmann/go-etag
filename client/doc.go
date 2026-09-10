// Package etagclient provides client-side conditional GET caching: an
// http.RoundTripper that replays stored ETags as If-None-Match validators and
// converts 304 Not Modified responses into the 200 responses callers expect.
//
// Wrap any http.Client with it:
//
//	client := &http.Client{
//		Transport: etagclient.NewTransport(http.DefaultTransport, etagclient.Options{}),
//	}
//
// Revalidation still costs one request, but unchanged responses are served
// from memory: no body re-download, and APIs that meter by response size (for
// example GitHub's rate limits) count only the cheap 304.
//
// # Credential warning
//
// The default cache key is the request URL alone. If the same URL can return
// different responses for different credentials (Authorization header,
// cookies, API tokens), supply an [Options.KeyFunc] that scopes the key by
// credential, or one principal's cached response may be served to another.
//
// # Stale edge caches and Age
//
// The transport never serves a stored body without revalidating first, so it
// cannot serve content that changed at the origin. It can, however, relay a
// validator the server still considers current: an edge cache answering 200
// or 304 from a copy the origin has not seen in hours is indistinguishable
// from a fresh origin by ETag alone. RFC 9111 §5.1 makes Age the signal:
// "The presence of an Age header field implies that the response was not
// generated or validated by the origin server for this request." Age and
// every other field the 304 provides are surfaced on the rebuilt response
// (RFC 9111 §4.3.4 freshening), so callers can detect stale-at-the-edge
// content instead of trusting the validator. Responses carrying
// Cache-Control: no-store are never stored (RFC 9111 §3), and a non-error
// response to an unsafe request method invalidates the stored entry for
// that URI (RFC 9111 §4.4), so a mutation cannot leave a pre-mutation body
// waiting to be rebuilt.
//
// # Vary
//
// The cache key is derived by [Options.KeyFunc]; Vary is not consulted. When
// the same URL serves different representations selected by request headers
// (Vary: Accept-Language, Accept), supply a KeyFunc that includes those
// request headers, mirroring the credential warning above.
package etagclient

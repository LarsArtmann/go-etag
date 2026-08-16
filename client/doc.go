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
package etagclient

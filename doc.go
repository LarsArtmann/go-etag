// Package etag provides HTTP ETag middleware that generates ETag headers
// based on response body content and handles If-None-Match conditional
// requests with 304 Not Modified responses.
//
// The middleware uses FNV-64a by default for fast, collision-resistant
// ETag generation and implements RFC 7232 weak comparison for
// If-None-Match matching.
package etag

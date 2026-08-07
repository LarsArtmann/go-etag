// Package etag provides HTTP ETag middleware that generates entity-tag headers
// from response body content and handles If-None-Match conditional requests
// with 304 Not Modified responses.
//
// The middleware uses FNV-64a by default for fast, collision-resistant entity-tag
// generation and implements RFC 7232 weak comparison for If-None-Match matching.
//
// The [ETag] type represents an RFC 7232 §2.3 entity-tag with explicit strength
// ([Strong] or [Weak]) and provides both strong and weak comparison methods.
// The [New] function returns middleware that buffers GET and HEAD responses,
// computes entity-tags, and handles conditional requests.
package etag

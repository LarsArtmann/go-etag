// Package etag is a deprecated compatibility shim for the server middleware,
// which moved to github.com/larsartmann/go-etag/server.
//
// Migration is a pure import-path swap; the package name stays etag, so no
// call-site changes are needed:
//
//	// Before:
//	import "github.com/larsartmann/go-etag"
//
//	// After:
//	import "github.com/larsartmann/go-etag/server"
//
// A generic client-side conditional-GET transport is available as
// github.com/larsartmann/go-etag/client (package etagclient).
//
// Deprecated: the root package no longer contains implementation code. Import
// github.com/larsartmann/go-etag/server instead. This shim will be removed in
// v1.0.0.
package etag

package etagclient

import (
	"net/http"
	"testing"
)

func TestResponseCacheEvictsOldest(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(2)

	for range 5 {
		cache.set("key", cacheEntry{etag: `"v"`, body: []byte{0}})

		if entries := cache.stats().Entries; entries > 2 {
			t.Fatalf("entries = %d, exceeds max", entries)
		}
	}

	cache.set("k1", cacheEntry{etag: `"1"`})
	cache.set("k2", cacheEntry{etag: `"2"`})
	cache.set("k3", cacheEntry{etag: `"3"`})

	if entries := cache.stats().Entries; entries != 2 {
		t.Fatalf("entries = %d, want 2", entries)
	}

	if _, ok := cache.get("k1"); ok {
		t.Error("k1 should have been evicted first")
	}

	if _, ok := cache.get("k3"); !ok {
		t.Error("k3 must be present")
	}
}

func TestResponseCacheReplaceKeepsEntry(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(2)

	cache.set("k1", cacheEntry{etag: `"a"`})
	cache.set("k2", cacheEntry{etag: `"b"`})
	cache.set("k1", cacheEntry{etag: `"a2"`})

	if entries := cache.stats().Entries; entries != 2 {
		t.Fatalf("entries = %d, want 2 (replacement must not grow the cache)", entries)
	}

	entry, ok := cache.get("k1")
	if !ok {
		t.Fatal("k1 must be present")
	}

	if entry.etag != `"a2"` {
		t.Errorf("etag = %q, want the replaced value", entry.etag)
	}
}

func TestResponseCacheStats(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(4)

	cache.set("k1", cacheEntry{etag: `"a"`})
	cache.set("k2", cacheEntry{etag: `"b"`})
	cache.countHit()
	cache.countHit()

	got := cache.stats()

	want := Stats{Hits: 2, Stored: 2, Entries: 2}
	if got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

func TestNewTransportDefaultsToDefaultTransport(t *testing.T) {
	t.Parallel()

	transport := NewTransport(nil, Options{})

	if transport.next != http.DefaultTransport {
		t.Error("nil next must fall back to http.DefaultTransport")
	}
}

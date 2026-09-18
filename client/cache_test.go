package etagclient

import (
	"net/http"
	"testing"
)

func TestResponseCacheEvictsOldest(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(2)

	for range 5 {
		cache.set("key", storedResponse{validator: newStoredValidator(`"v"`), body: []byte{0}})

		if entries := cache.stats().Entries; entries > 2 {
			t.Fatalf("entries = %d, exceeds max", entries)
		}
	}

	cache.set("k1", storedResponse{validator: newStoredValidator(`"1"`)})
	cache.set("k2", storedResponse{validator: newStoredValidator(`"2"`)})
	cache.set("k3", storedResponse{validator: newStoredValidator(`"3"`)})

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

	cache.set("k1", storedResponse{validator: newStoredValidator(`"a"`)})
	cache.set("k2", storedResponse{validator: newStoredValidator(`"b"`)})
	cache.set("k1", storedResponse{validator: newStoredValidator(`"a2"`)})

	if entries := cache.stats().Entries; entries != 2 {
		t.Fatalf("entries = %d, want 2 (replacement must not grow the cache)", entries)
	}

	entry, ok := cache.get("k1")
	if !ok {
		t.Fatal("k1 must be present")
	}

	if entry.validator.wire != `"a2"` {
		t.Errorf("validator = %q, want the replaced value", entry.validator.wire)
	}
}

func TestResponseCacheStats(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(4)

	cache.set("k1", storedResponse{validator: newStoredValidator(`"a"`)})
	cache.set("k2", storedResponse{validator: newStoredValidator(`"b"`)})
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

func TestResponseCacheFreshenReplacesMatchingEntry(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(2)

	cache.set("k1", storedResponse{validator: newStoredValidator(`"a"`), body: []byte("old")})
	cache.freshen("k1", `"a"`, storedResponse{validator: newStoredValidator(`"a"`), body: []byte("new")})

	entry, ok := cache.get("k1")
	if !ok {
		t.Fatal("k1 must be present")
	}

	if string(entry.body) != "new" {
		t.Errorf("body = %q, want the freshened value", entry.body)
	}
}

func TestResponseCacheFreshenSkipsWhenValidatorMovedOn(t *testing.T) {
	t.Parallel()

	// RFC 9111 §4.3.4 updates the stored response that was validated; when a
	// concurrent 200 already replaced the entry, the older freshening must
	// not overwrite the newer response.
	cache := newResponseCache(2)

	cache.set("k1", storedResponse{validator: newStoredValidator(`"old"`), body: []byte("validated entry")})
	cache.set("k1", storedResponse{validator: newStoredValidator(`"new"`), body: []byte("concurrent 200")})
	cache.freshen("k1", `"old"`, storedResponse{validator: newStoredValidator(`"old"`), body: []byte("stale freshening")})

	entry, ok := cache.get("k1")
	if !ok {
		t.Fatal("k1 must be present")
	}

	if entry.validator.wire != `"new"` || string(entry.body) != "concurrent 200" {
		t.Errorf("entry = %s %q, want the concurrent 200 to win", entry.validator.wire, entry.body)
	}
}

func TestResponseCacheFreshenAbsentKeyIsInert(t *testing.T) {
	t.Parallel()

	cache := newResponseCache(2)

	cache.freshen("missing", `"a"`, storedResponse{validator: newStoredValidator(`"a"`)})

	if got := cache.stats().Entries; got != 0 {
		t.Errorf("entries = %d, want 0 (freshening must not create entries)", got)
	}
}

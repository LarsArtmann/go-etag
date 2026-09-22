package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-etag/metrics"
	etag "github.com/larsartmann/go-etag/server"
)

func newETagHandler(t *testing.T, cfg etag.ETagConfig) (http.Handler, *metrics.Counters) {
	t.Helper()

	cfg, counters := metrics.Attach(cfg)

	return etag.New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello metrics"))
	})), counters
}

func TestAttach_CountsFreshGET(t *testing.T) {
	t.Parallel()

	handler, counters := newETagHandler(t, etag.DefaultETagConfig())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if rec.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header on fresh GET")
	}

	got := counters.Snapshot()
	if got.Generated != 1 || got.NotModified != 0 || got.BufferOverflows != 0 {
		t.Fatalf("expected Generated=1 NotModified=0 Overflows=0, got %+v", got)
	}
}

func TestAttach_CountsConditional304(t *testing.T) {
	t.Parallel()

	handler, counters := newETagHandler(t, etag.DefaultETagConfig())

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	tag := first.Header().Get("ETag")
	if tag == "" {
		t.Fatal("expected ETag on first response")
	}

	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", tag)
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Fatalf("expected 304 on conditional GET, got %d", second.Code)
	}

	got := counters.Snapshot()
	if got.NotModified != 1 {
		t.Fatalf("expected NotModified=1, got %+v", got)
	}

	if got.Generated != 2 {
		t.Fatalf("expected Generated=2 (tag resolved per request), got %+v", got)
	}

	if ratio := counters.HitRatio(); ratio != 0.5 {
		t.Fatalf("expected hit ratio 1/2 (one 304 of two tag-computing responses), got %v", ratio)
	}
}

func TestAttach_CountsBufferOverflow(t *testing.T) {
	t.Parallel()

	cfg := etag.DefaultETagConfig()
	cfg.MaxBufferSize = 64

	cfg, counters := metrics.Attach(cfg)

	handler := etag.New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 1024)))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected streamed 200, got %d", rec.Code)
	}

	if rec.Header().Get("ETag") != "" {
		t.Fatal("expected no ETag on overflowed streamed response")
	}

	got := counters.Snapshot()
	if got.BufferOverflows != 1 {
		t.Fatalf("expected BufferOverflows=1, got %+v", got)
	}

	if got.Generated != 0 {
		t.Fatalf("expected Generated=0 on streamed response, got %+v", got)
	}
}

func TestAttach_ChainsExistingHooks(t *testing.T) {
	t.Parallel()

	var (
		userTag      []string
		userOverflow []int
		body         = "hello"
	)

	cfg := etag.DefaultETagConfig()
	cfg.MaxBufferSize = 64
	cfg.OnETagGenerated = func(e etag.ETag) {
		userTag = append(userTag, e.String())
	}
	cfg.OnBufferOverflow = func(limit int) {
		userOverflow = append(userOverflow, limit)
	}

	cfg, counters := metrics.Attach(cfg)

	handler := etag.New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	fresh := httptest.NewRecorder()
	handler.ServeHTTP(fresh, httptest.NewRequest(http.MethodGet, "/", nil))

	body = strings.Repeat("x", 1024)

	big := httptest.NewRecorder()
	handler.ServeHTTP(big, httptest.NewRequest(http.MethodGet, "/", nil))

	if len(userTag) != 1 {
		t.Fatalf("expected user OnETagGenerated to fire once, got %d", len(userTag))
	}

	if len(userOverflow) != 1 || userOverflow[0] != 64 {
		t.Fatalf("expected user OnBufferOverflow(64), got %v", userOverflow)
	}

	got := counters.Snapshot()
	if got.Generated != 1 || got.BufferOverflows != 1 {
		t.Fatalf("expected counters alongside user hooks, got %+v", got)
	}
}

func TestHitRatio_ZeroBeforeEvents(t *testing.T) {
	t.Parallel()

	var counters metrics.Counters

	if ratio := counters.HitRatio(); ratio != 0 {
		t.Fatalf("expected 0 hit ratio before any event, got %v", ratio)
	}
}

func TestHitRatio_AdoptedTag304DoesNotCountAsGenerated(t *testing.T) {
	t.Parallel()

	cfg := etag.DefaultETagConfig()
	cfg.SkipIfPresent = true

	cfg, counters := metrics.Attach(cfg)

	handler := etag.New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"manual"`)
		_, _ = w.Write([]byte("adopted"))
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))

	second := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", `"manual"`)
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusNotModified {
		t.Fatalf("expected 304 on conditional GET with adopted tag, got %d", second.Code)
	}

	got := counters.Snapshot()
	if got.Generated != 0 || got.NotModified != 1 {
		t.Fatalf("expected Generated=0 NotModified=1 (adopted tags are not generated), got %+v", got)
	}

	if ratio := counters.HitRatio(); ratio != 0 {
		t.Fatalf("expected 0 hit ratio with no computed tags, got %v", ratio)
	}
}

func TestSnapshot_ReflectsIndependentCounters(t *testing.T) {
	t.Parallel()

	var counters metrics.Counters

	counters.Generated.Store(3)
	counters.NotModified.Store(1)
	counters.BufferOverflows.Store(2)

	got := counters.Snapshot()
	if got.Generated != 3 || got.NotModified != 1 || got.BufferOverflows != 2 {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}

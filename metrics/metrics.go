import (
	"sync/atomic"

	"github.com/larsartmann/go-etag/server"
)

// Counters holds the atomic event counters for one [server.ETagConfig].
// Create one via [Attach]; all methods are safe for concurrent use.
type Counters struct {
	// Generated counts OnETagGenerated events: one per response whose ETag
	// the middleware computed and set. It fires for 200 responses and also
	// for 304s (the tag is resolved to answer the conditional), but not for
	// handler-provided tags adopted via SkipIfPresent, nor for streamed
	// responses that exceeded MaxBufferSize.
	Generated atomic.Int64
	// NotModified counts On304 events: one per conditional request answered
	// with 304 Not Modified, regardless of whether the matching tag was
	// computed or adopted from the handler.
	NotModified atomic.Int64
	// BufferOverflows counts OnBufferOverflow events: one per response whose
	// body exceeded MaxBufferSize and was streamed without an ETag.
	BufferOverflows atomic.Int64
}

// Snapshot is a point-in-time copy of [Counters] values.
type Snapshot struct {
	Generated       int64
	NotModified     int64
	BufferOverflows int64
}

// Attach installs counting hooks on a copy of cfg and returns the modified
// configuration (pass it to [server.New]) together with the counters it will
// update. Hooks already present on cfg are preserved and run after the
// counting hooks, so attaching never silently drops consumer instrumentation.
func Attach(cfg server.ETagConfig) (server.ETagConfig, *Counters) {
	c := &Counters{}

	generated := cfg.OnETagGenerated
	cfg.OnETagGenerated = func(e server.ETag) {
		c.Generated.Add(1)
		if generated != nil {
			generated(e)
		}
	}

	notModified := cfg.On304
	cfg.On304 = func(e server.ETag) {
		c.NotModified.Add(1)
		if notModified != nil {
			notModified(e)
		}
	}

	overflow := cfg.OnBufferOverflow
	cfg.OnBufferOverflow = func(limit int) {
		c.BufferOverflows.Add(1)
		if overflow != nil {
			overflow(limit)
		}
	}

	return cfg, c
}

// Snapshot returns the current counter values.
func (c *Counters) Snapshot() Snapshot {
	return Snapshot{
		Generated:       c.Generated.Load(),
		NotModified:     c.NotModified.Load(),
		BufferOverflows: c.BufferOverflows.Load(),
	}
}

// HitRatio returns the fraction of tag-computing responses answered with
// 304 Not Modified: NotModified / Generated. On304 fires in addition to
// OnETagGenerated for computed tags, so Generated alone counts every
// tag-computing response exactly once. It returns 0 before any tag is
// computed. Caveat: 304s on handler-provided tags (SkipIfPresent) fire
// On304 without OnETagGenerated and can push the ratio above 1; for exact
// accounting across adopted tags, install a dedicated hook.
func (c *Counters) HitRatio() float64 {
	gen := c.Generated.Load()
	if gen == 0 {
		return 0
	}

	return float64(c.NotModified.Load()) / float64(gen)
}

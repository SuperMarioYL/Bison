package opencost

import (
	"sync"
	"time"
)

type allocCacheEntry struct {
	ready  chan struct{} // closed when val/err are populated
	val    []Allocation
	err    error
	expiry time.Time // guarded by allocCache.mu; zero while in-flight
}

// allocCache is a small TTL cache that also coalesces concurrent identical
// allocation queries, so a burst of dashboard/billing requests for the same
// window/aggregate/filter hits OpenCost once instead of once per caller.
type allocCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]*allocCacheEntry
}

func newAllocCache(ttl time.Duration) *allocCache {
	return &allocCache{ttl: ttl, entries: make(map[string]*allocCacheEntry)}
}

// do returns a cached result if fresh, joins an in-flight fetch for the same key,
// or runs fetch once and caches the (successful) result for ttl. Errors are not
// cached so the next caller retries.
func (c *allocCache) do(key string, fetch func() ([]Allocation, error)) ([]Allocation, error) {
	c.mu.Lock()
	if e := c.entries[key]; e != nil && (e.expiry.IsZero() || time.Now().Before(e.expiry)) {
		// In-flight (zero expiry) or still-fresh cached result: reuse it.
		c.mu.Unlock()
		<-e.ready
		return e.val, e.err
	}
	e := &allocCacheEntry{ready: make(chan struct{})}
	c.entries[key] = e
	c.mu.Unlock()

	e.val, e.err = fetch()

	c.mu.Lock()
	if e.err != nil {
		// Do not cache failures; drop so the next caller retries.
		if c.entries[key] == e {
			delete(c.entries, key)
		}
	} else {
		e.expiry = time.Now().Add(c.ttl)
	}
	c.mu.Unlock()

	close(e.ready)
	return e.val, e.err
}

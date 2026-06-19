package opencost

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAllocCacheCoalescesConcurrentCalls(t *testing.T) {
	c := newAllocCache(time.Minute)
	var calls int32

	fetch := func() ([]Allocation, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(40 * time.Millisecond) // hold the in-flight window open
		return []Allocation{{Name: "ns"}}, nil
	}

	const n = 12
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			res, err := c.do("k", fetch)
			if err != nil || len(res) != 1 {
				t.Errorf("unexpected result: %v %v", res, err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 underlying fetch (coalesced), got %d", got)
	}
}

func TestAllocCacheTTL(t *testing.T) {
	c := newAllocCache(40 * time.Millisecond)
	var calls int32
	fetch := func() ([]Allocation, error) {
		atomic.AddInt32(&calls, 1)
		return nil, nil
	}

	_, _ = c.do("k", fetch)
	_, _ = c.do("k", fetch) // within TTL -> cached
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 fetch within TTL, got %d", got)
	}

	time.Sleep(60 * time.Millisecond) // let it expire
	_, _ = c.do("k", fetch)
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected re-fetch after TTL, got %d", got)
	}
}

func TestAllocCacheDoesNotCacheErrors(t *testing.T) {
	c := newAllocCache(time.Minute)
	var calls int32
	fetch := func() ([]Allocation, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("boom")
	}

	if _, err := c.do("k", fetch); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.do("k", fetch); err == nil {
		t.Fatal("expected error on retry")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("errors must not be cached; expected 2 fetches, got %d", got)
	}
}

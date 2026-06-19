package handler

import (
	"testing"
	"time"
)

func TestLoginLimiterBlocksAfterMaxFails(t *testing.T) {
	l := newLoginLimiter()
	now := time.Now()
	ip := "1.2.3.4"

	for i := 0; i < maxLoginFails-1; i++ {
		l.recordFailure(ip, now)
		if ok, _ := l.allowed(ip, now); !ok {
			t.Fatalf("blocked too early after %d fails", i+1)
		}
	}
	// The maxLoginFails-th failure triggers the block.
	l.recordFailure(ip, now)
	ok, retry := l.allowed(ip, now)
	if ok {
		t.Fatal("expected block after maxLoginFails failures")
	}
	if retry <= 0 {
		t.Fatalf("expected positive Retry-After, got %d", retry)
	}

	// Block clears after loginBlock elapses.
	if ok, _ := l.allowed(ip, now.Add(loginBlock+time.Second)); !ok {
		t.Fatal("expected unblock after loginBlock elapsed")
	}
}

func TestLoginLimiterSuccessResets(t *testing.T) {
	l := newLoginLimiter()
	now := time.Now()
	ip := "5.6.7.8"

	for i := 0; i < maxLoginFails; i++ {
		l.recordFailure(ip, now)
	}
	if ok, _ := l.allowed(ip, now); ok {
		t.Fatal("expected block")
	}
	// A successful login from a different (unblocked) state clears the record.
	l.recordSuccess(ip)
	if ok, _ := l.allowed(ip, now); !ok {
		t.Fatal("expected allowed after recordSuccess")
	}
}

func TestLoginLimiterWindowResets(t *testing.T) {
	l := newLoginLimiter()
	now := time.Now()
	ip := "9.9.9.9"

	// A few failures, then let the window expire before reaching the threshold.
	l.recordFailure(ip, now)
	l.recordFailure(ip, now)
	later := now.Add(loginWindow + time.Second)
	l.recordFailure(ip, later) // counter resets, so this is failure #1 in a new window
	if ok, _ := l.allowed(ip, later); !ok {
		t.Fatal("expected allowed; window should have reset the counter")
	}
}

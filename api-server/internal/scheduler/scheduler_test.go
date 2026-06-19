package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bison/api-server/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init(false)
	os.Exit(m.Run())
}

// TestSchedulerRestartable verifies the scheduler can be stopped and started
// again (required for leader-election re-acquisition) and that Start/Stop are
// idempotent and do not deadlock.
func TestSchedulerRestartable(t *testing.T) {
	s := NewScheduler(nil, nil, nil)
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		s.Start(ctx) // idempotent: second Start is a no-op
		s.Stop()
		s.Stop()     // idempotent: second Stop is a no-op
		s.Start(ctx) // re-startable after Stop
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start/Stop deadlocked")
	}
}

func TestStopBeforeStartIsSafe(t *testing.T) {
	s := NewScheduler(nil, nil, nil)
	s.Stop() // must not panic or block when never started
}

package scheduler

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// Scheduler handles scheduled tasks
type Scheduler struct {
	billingSvc *service.BillingService
	balanceSvc *service.BalanceService
	alertSvc   *service.AlertService

	executions   []service.TaskExecution
	executionsMu sync.RWMutex

	mu      sync.Mutex
	started bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler creates a new Scheduler
func NewScheduler(
	billingSvc *service.BillingService,
	balanceSvc *service.BalanceService,
	alertSvc *service.AlertService,
) *Scheduler {
	return &Scheduler{
		billingSvc: billingSvc,
		balanceSvc: balanceSvc,
		alertSvc:   alertSvc,
		executions: make([]service.TaskExecution, 0),
	}
}

// Start starts all scheduled tasks. It is idempotent and re-startable: calling
// Start after a Stop (e.g. when leadership is re-acquired) spins up a fresh set
// of tasks against a new stop channel.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return
	}
	s.started = true
	s.stopCh = make(chan struct{})
	logger.Info("Starting scheduler")

	s.wg.Add(3)
	go s.runBillingTask(ctx)
	go s.runAutoRechargeTask(ctx)
	go s.runAlertTask(ctx)
}

// Stop stops all scheduled tasks and waits for them to exit. Safe to call when
// not started.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	close(s.stopCh)
	s.mu.Unlock()

	logger.Info("Stopping scheduler")
	s.wg.Wait()
}

// GetExecutions returns recent task executions (implements service.TaskExecutionGetter)
func (s *Scheduler) GetExecutions(limit int) []service.TaskExecution {
	s.executionsMu.RLock()
	defer s.executionsMu.RUnlock()

	if limit <= 0 || limit > len(s.executions) {
		limit = len(s.executions)
	}

	// Return most recent executions
	start := len(s.executions) - limit
	if start < 0 {
		start = 0
	}

	result := make([]service.TaskExecution, limit)
	copy(result, s.executions[start:])

	// Reverse to show most recent first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// safeExecute runs a task body with panic recovery so a single failing task can
// never take down the whole api-server process.
func (s *Scheduler) safeExecute(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Scheduled task panicked and was recovered", "task", name, "panic", r)
		}
	}()
	fn()
}

// sleepWithJitter waits a random duration in [0, max) to desynchronize task
// firing across replicas, returning false if the scheduler is stopped meanwhile.
func (s *Scheduler) sleepWithJitter(stopCh <-chan struct{}, max time.Duration) bool {
	if max <= 0 {
		return true
	}
	timer := time.NewTimer(time.Duration(rand.Int63n(int64(max))))
	defer timer.Stop()
	select {
	case <-stopCh:
		return false
	case <-timer.C:
		return true
	}
}

func (s *Scheduler) runBillingTask(ctx context.Context) {
	defer s.wg.Done()
	stopCh := s.stopCh

	if !s.sleepWithJitter(stopCh, 60*time.Second) {
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.safeExecute("billing", func() { s.executeBillingTask(ctx) })
		}
	}
}

func (s *Scheduler) executeBillingTask(ctx context.Context) {
	exec := service.TaskExecution{
		TaskName:  "billing",
		StartTime: time.Now(),
		Status:    "success",
	}

	if s.billingSvc == nil {
		exec.Status = "skipped"
		exec.Error = "billing service not configured"
	} else {
		if err := s.billingSvc.ProcessBilling(ctx); err != nil {
			exec.Status = "failed"
			exec.Error = err.Error()
			logger.Error("Billing task failed", "error", err)
		} else {
			logger.Info("Billing task completed")
		}
	}

	exec.EndTime = time.Now()
	s.recordExecution(exec)
}

func (s *Scheduler) runAutoRechargeTask(ctx context.Context) {
	defer s.wg.Done()
	stopCh := s.stopCh

	if !s.sleepWithJitter(stopCh, 60*time.Second) {
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.safeExecute("auto_recharge", func() { s.executeAutoRechargeTask(ctx) })
		}
	}
}

func (s *Scheduler) executeAutoRechargeTask(ctx context.Context) {
	exec := service.TaskExecution{
		TaskName:  "auto_recharge",
		StartTime: time.Now(),
		Status:    "success",
	}

	if s.balanceSvc == nil {
		exec.Status = "skipped"
		exec.Error = "balance service not configured"
	} else {
		if err := s.balanceSvc.ProcessAutoRecharge(ctx); err != nil {
			exec.Status = "failed"
			exec.Error = err.Error()
			logger.Error("Auto-recharge task failed", "error", err)
		} else {
			logger.Info("Auto-recharge task completed")
		}
	}

	exec.EndTime = time.Now()
	s.recordExecution(exec)
}

func (s *Scheduler) runAlertTask(ctx context.Context) {
	defer s.wg.Done()
	stopCh := s.stopCh

	if !s.sleepWithJitter(stopCh, 30*time.Second) {
		return
	}

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.safeExecute("alert_check", func() { s.executeAlertTask(ctx) })
		}
	}
}

func (s *Scheduler) executeAlertTask(ctx context.Context) {
	exec := service.TaskExecution{
		TaskName:  "alert_check",
		StartTime: time.Now(),
		Status:    "success",
	}

	if s.alertSvc == nil {
		exec.Status = "skipped"
		exec.Error = "alert service not configured"
	} else {
		if err := s.alertSvc.CheckAndNotify(ctx); err != nil {
			exec.Status = "failed"
			exec.Error = err.Error()
			logger.Error("Alert check task failed", "error", err)
		} else {
			logger.Debug("Alert check task completed")
		}
	}

	exec.EndTime = time.Now()
	s.recordExecution(exec)
}

func (s *Scheduler) recordExecution(exec service.TaskExecution) {
	s.executionsMu.Lock()
	defer s.executionsMu.Unlock()

	s.executions = append(s.executions, exec)

	// Keep only last 1000 executions
	if len(s.executions) > 1000 {
		s.executions = s.executions[len(s.executions)-1000:]
	}
}

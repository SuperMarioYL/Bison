package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

const (
	// ConfigMap names
	BalancesConfigMap        = "bison-team-balances"
	RechargeHistoryConfigMap = "bison-recharge-history"
	AutoRechargeConfigMap    = "bison-auto-recharge"
	BisonNamespace           = "bison-system"
)

// Balance represents a team's balance
type Balance struct {
	TeamName           string     `json:"teamName"`
	Amount             float64    `json:"amount"`
	LastUpdated        time.Time  `json:"lastUpdated"`
	OverdueAt          *time.Time `json:"overdueAt,omitempty"`          // When balance first went negative
	EstimatedOverdueAt *time.Time `json:"estimatedOverdueAt,omitempty"` // Predicted time when balance will go negative
	DailyConsumption   float64    `json:"dailyConsumption,omitempty"`   // Average daily consumption
	GraceRemaining     string     `json:"graceRemaining,omitempty"`     // Remaining grace period (e.g., "2天 3小时")
}

// RechargeRecord represents a recharge or deduction record
type RechargeRecord struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`   // "recharge", "deduction", "auto_recharge"
	Amount    float64   `json:"amount"` // Positive for recharge, negative for deduction
	Operator  string    `json:"operator"`
	Reason    string    `json:"reason,omitempty"`
	Balance   float64   `json:"balance"` // Balance after this operation
}

// AutoRechargeConfig represents auto-recharge configuration for a team
type AutoRechargeConfig struct {
	Enabled       bool      `json:"enabled"`
	Amount        float64   `json:"amount"`
	Schedule      string    `json:"schedule"`   // "weekly" or "monthly"
	DayOfWeek     int       `json:"dayOfWeek"`  // 0-6 for weekly (0=Sunday)
	DayOfMonth    int       `json:"dayOfMonth"` // 1-31 for monthly
	NextExecution time.Time `json:"nextExecution"`
	LastExecuted  time.Time `json:"lastExecuted,omitempty"`
}

// BalanceService handles team balance operations
type BalanceService struct {
	k8sClient *k8s.Client
}

// NewBalanceService creates a new BalanceService
func NewBalanceService(k8sClient *k8s.Client) *BalanceService {
	return &BalanceService{
		k8sClient: k8sClient,
	}
}

// GetBalance returns the balance for a team
func (s *BalanceService) GetBalance(ctx context.Context, teamName string) (*Balance, error) {
	logger.Debug("Getting balance", "team", teamName)

	cm, err := s.getOrCreateConfigMap(ctx, BalancesConfigMap)
	if err != nil {
		return nil, err
	}

	data, ok := cm.Data[teamName]
	if !ok {
		// Return zero balance if not found
		return &Balance{
			TeamName:    teamName,
			Amount:      0,
			LastUpdated: time.Now(),
		}, nil
	}

	var balance Balance
	if err := json.Unmarshal([]byte(data), &balance); err != nil {
		logger.Error("Failed to unmarshal balance", "team", teamName, "error", err)
		return nil, fmt.Errorf("failed to parse balance: %w", err)
	}

	balance.TeamName = teamName
	return &balance, nil
}

// GetAllBalances returns balances for all teams
func (s *BalanceService) GetAllBalances(ctx context.Context) ([]*Balance, error) {
	logger.Debug("Getting all balances")

	cm, err := s.getOrCreateConfigMap(ctx, BalancesConfigMap)
	if err != nil {
		return nil, err
	}

	var balances []*Balance
	for teamName, data := range cm.Data {
		var balance Balance
		if err := json.Unmarshal([]byte(data), &balance); err != nil {
			logger.Warn("Failed to unmarshal balance", "team", teamName, "error", err)
			continue
		}
		balance.TeamName = teamName
		balances = append(balances, &balance)
	}

	return balances, nil
}

// Recharge adds balance to a team. The read-modify-write is performed under
// optimistic-concurrency retry so concurrent recharges/deductions on the shared
// balances ConfigMap cannot silently lose updates.
func (s *BalanceService) Recharge(ctx context.Context, teamName string, amount float64, operator, remark string) error {
	logger.Info("Recharging team", "team", teamName, "amount", amount, "operator", operator)

	if amount <= 0 {
		return fmt.Errorf("recharge amount must be positive")
	}

	newAmount, err := s.applyBalanceDelta(ctx, teamName, amount)
	if err != nil {
		return err
	}

	// Record history
	record := &RechargeRecord{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "recharge",
		Amount:    amount,
		Operator:  operator,
		Reason:    remark,
		Balance:   newAmount,
	}

	return s.addRechargeRecord(ctx, teamName, record)
}

// Deduct deducts balance from a team (negative balances are allowed) and returns
// the balance AFTER the write, so callers do not need a second racy read to decide
// suspension. The update is conflict-retried for concurrency safety.
func (s *BalanceService) Deduct(ctx context.Context, teamName string, amount float64, reason string) (float64, error) {
	logger.Info("Deducting from team", "team", teamName, "amount", amount, "reason", reason)

	if amount <= 0 {
		return 0, fmt.Errorf("deduction amount must be positive")
	}

	newAmount, err := s.applyBalanceDelta(ctx, teamName, -amount)
	if err != nil {
		return 0, err
	}

	// Record history
	record := &RechargeRecord{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "deduction",
		Amount:    -amount,
		Operator:  "system",
		Reason:    reason,
		Balance:   newAmount,
	}

	if err := s.addRechargeRecord(ctx, teamName, record); err != nil {
		return newAmount, err
	}
	return newAmount, nil
}

// GetRechargeHistory returns recharge/deduction history for a team
func (s *BalanceService) GetRechargeHistory(ctx context.Context, teamName string, limit int) ([]*RechargeRecord, error) {
	logger.Debug("Getting recharge history", "team", teamName, "limit", limit)

	cm, err := s.getOrCreateConfigMap(ctx, RechargeHistoryConfigMap)
	if err != nil {
		return nil, err
	}

	data, ok := cm.Data[teamName]
	if !ok {
		return []*RechargeRecord{}, nil
	}

	var records []*RechargeRecord
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		logger.Error("Failed to unmarshal history", "team", teamName, "error", err)
		return nil, fmt.Errorf("failed to parse history: %w", err)
	}

	// Sort by timestamp descending
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}

	return records, nil
}

// GetAutoRechargeConfig returns auto-recharge configuration for a team
func (s *BalanceService) GetAutoRechargeConfig(ctx context.Context, teamName string) (*AutoRechargeConfig, error) {
	logger.Debug("Getting auto-recharge config", "team", teamName)

	cm, err := s.getOrCreateConfigMap(ctx, AutoRechargeConfigMap)
	if err != nil {
		return nil, err
	}

	data, ok := cm.Data[teamName]
	if !ok {
		return &AutoRechargeConfig{Enabled: false}, nil
	}

	var config AutoRechargeConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		logger.Error("Failed to unmarshal auto-recharge config", "team", teamName, "error", err)
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SetAutoRechargeConfig sets auto-recharge configuration for a team
func (s *BalanceService) SetAutoRechargeConfig(ctx context.Context, teamName string, config *AutoRechargeConfig) error {
	logger.Info("Setting auto-recharge config", "team", teamName, "enabled", config.Enabled)

	// Reject a non-positive amount on an enabled config: it would otherwise be
	// applied every tick and silently deduct from the balance.
	if config.Enabled && config.Amount <= 0 {
		return fmt.Errorf("auto-recharge amount must be positive when enabled")
	}

	// Calculate next execution time
	if config.Enabled {
		config.NextExecution = s.calculateNextExecution(config)
	}

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	cm, err := s.getOrCreateConfigMap(ctx, AutoRechargeConfigMap)
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[teamName] = string(data)

	return s.updateConfigMap(ctx, cm)
}

// ProcessAutoRecharge processes auto-recharge for all teams
func (s *BalanceService) ProcessAutoRecharge(ctx context.Context) error {
	logger.Debug("Processing auto-recharge")

	cm, err := s.getOrCreateConfigMap(ctx, AutoRechargeConfigMap)
	if err != nil {
		return err
	}

	now := time.Now()
	for teamName, data := range cm.Data {
		var config AutoRechargeConfig
		if err := json.Unmarshal([]byte(data), &config); err != nil {
			logger.Warn("Failed to unmarshal auto-recharge config", "team", teamName, "error", err)
			continue
		}

		if !config.Enabled {
			continue
		}

		// Check if it's time to execute
		if now.Before(config.NextExecution) {
			continue
		}

		if config.Amount <= 0 {
			logger.Warn("Skipping auto-recharge with non-positive amount", "team", teamName, "amount", config.Amount)
			continue
		}

		logger.Info("Executing auto-recharge", "team", teamName, "amount", config.Amount)

		// Atomically add the recharge amount to the current balance.
		newAmount, err := s.applyBalanceDelta(ctx, teamName, config.Amount)
		if err != nil {
			logger.Error("Failed to update balance for auto-recharge", "team", teamName, "error", err)
			continue
		}

		// Record history
		record := &RechargeRecord{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp: now,
			Type:      "auto_recharge",
			Amount:    config.Amount,
			Operator:  "system",
			Reason:    fmt.Sprintf("Auto recharge (%s)", config.Schedule),
			Balance:   newAmount,
		}
		if err := s.addRechargeRecord(ctx, teamName, record); err != nil {
			logger.Error("Failed to record auto-recharge", "team", teamName, "error", err)
		}

		// Update config with next execution time
		config.LastExecuted = now
		config.NextExecution = s.calculateNextExecution(&config)
		if err := s.SetAutoRechargeConfig(ctx, teamName, &config); err != nil {
			logger.Error("Failed to update auto-recharge config", "team", teamName, "error", err)
		}
	}

	return nil
}

// GetLowBalanceTeams returns teams with balance below threshold
func (s *BalanceService) GetLowBalanceTeams(ctx context.Context, threshold float64) ([]*Balance, error) {
	balances, err := s.GetAllBalances(ctx)
	if err != nil {
		return nil, err
	}

	var lowBalanceTeams []*Balance
	for _, balance := range balances {
		if balance.Amount < threshold {
			lowBalanceTeams = append(lowBalanceTeams, balance)
		}
	}

	return lowBalanceTeams, nil
}

// GetTotalBalance returns the sum of all team balances
func (s *BalanceService) GetTotalBalance(ctx context.Context) (float64, error) {
	balances, err := s.GetAllBalances(ctx)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, balance := range balances {
		total += balance.Amount
	}

	return total, nil
}

// Helper methods

// mutateConfigMap performs an optimistic-concurrency read-modify-write on a Bison
// ConfigMap. It re-reads the ConfigMap and re-applies mutate on every resourceVersion
// conflict, so concurrent writers (recharge, billing deduction, auto-recharge,
// overdue marking) cannot silently lose each other's updates.
func (s *BalanceService) mutateConfigMap(ctx context.Context, name string, mutate func(cm *corev1.ConfigMap) error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, err := s.getOrCreateConfigMap(ctx, name)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		if err := mutate(cm); err != nil {
			return err
		}
		// Use the raw client so a conflict error is returned unwrapped for RetryOnConflict.
		return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
	})
}

// applyBalanceDelta atomically adds delta (negative to deduct) to a team's balance,
// preserving other persisted fields such as OverdueAt, and returns the new amount.
func (s *BalanceService) applyBalanceDelta(ctx context.Context, teamName string, delta float64) (float64, error) {
	var newAmount float64
	err := s.mutateConfigMap(ctx, BalancesConfigMap, func(cm *corev1.ConfigMap) error {
		balance := Balance{TeamName: teamName}
		if existing, ok := cm.Data[teamName]; ok {
			if err := json.Unmarshal([]byte(existing), &balance); err != nil {
				return fmt.Errorf("failed to parse balance: %w", err)
			}
		}
		balance.TeamName = teamName
		balance.Amount += delta
		balance.LastUpdated = time.Now()

		data, err := json.Marshal(&balance)
		if err != nil {
			return fmt.Errorf("failed to marshal balance: %w", err)
		}
		cm.Data[teamName] = string(data)
		newAmount = balance.Amount
		return nil
	})
	return newAmount, err
}

func (s *BalanceService) addRechargeRecord(ctx context.Context, teamName string, record *RechargeRecord) error {
	return s.mutateConfigMap(ctx, RechargeHistoryConfigMap, func(cm *corev1.ConfigMap) error {
		var records []*RechargeRecord
		if data, ok := cm.Data[teamName]; ok {
			if err := json.Unmarshal([]byte(data), &records); err != nil {
				logger.Warn("Failed to unmarshal existing history, starting fresh", "team", teamName)
				records = []*RechargeRecord{}
			}
		}

		// Add new record
		records = append(records, record)

		// Keep only last 1000 records
		if len(records) > 1000 {
			records = records[len(records)-1000:]
		}

		data, err := json.Marshal(records)
		if err != nil {
			return fmt.Errorf("failed to marshal history: %w", err)
		}
		cm.Data[teamName] = string(data)
		return nil
	})
}

func (s *BalanceService) getOrCreateConfigMap(ctx context.Context, name string) (*corev1.ConfigMap, error) {
	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the ConfigMap
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: BisonNamespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":      "bison",
						"app.kubernetes.io/component": "billing",
					},
				},
				Data: make(map[string]string),
			}
			if err := s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm); err != nil {
				return nil, fmt.Errorf("failed to create configmap: %w", err)
			}
			return cm, nil
		}
		return nil, fmt.Errorf("failed to get configmap: %w", err)
	}

	return cm, nil
}

func (s *BalanceService) updateConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	if err := s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm); err != nil {
		return fmt.Errorf("failed to update configmap: %w", err)
	}
	return nil
}

func (s *BalanceService) calculateNextExecution(config *AutoRechargeConfig) time.Time {
	now := time.Now()

	switch config.Schedule {
	case "weekly":
		// Find next occurrence of the specified day of week
		daysUntil := (config.DayOfWeek - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 {
			daysUntil = 7 // If today is the day, schedule for next week
		}
		return time.Date(now.Year(), now.Month(), now.Day()+daysUntil, 0, 0, 0, 0, now.Location())

	case "monthly":
		// Find next occurrence of the specified day of month
		next := time.Date(now.Year(), now.Month(), config.DayOfMonth, 0, 0, 0, 0, now.Location())
		if next.Before(now) || next.Equal(now) {
			next = next.AddDate(0, 1, 0)
		}
		// Handle months with fewer days
		for next.Day() != config.DayOfMonth {
			next = time.Date(next.Year(), next.Month()+1, config.DayOfMonth, 0, 0, 0, 0, now.Location())
		}
		return next

	default:
		// Default to monthly
		return time.Now().AddDate(0, 1, 0)
	}
}

// CalculateDailyConsumption estimates the average daily spend for a team from its
// recent deduction history.
//
// The denominator is the actual span of deduction activity within the last 7 days
// (capped at 7 days, floored to avoid wildly overestimating from a single very
// recent deduction) — NOT a fixed 7 days, which previously underestimated the burn
// rate whenever the real activity window was shorter. We also fetch enough records
// to cover a full week of hourly billing (168+), since the old 100-record cap
// truncated the 7-day window and undercounted total deductions.
func (s *BalanceService) CalculateDailyConsumption(ctx context.Context, teamName string) (float64, error) {
	const (
		lookbackDays = 7
		maxSpanDays  = 7.0
		minSpanDays  = 0.5
		fetchRecords = 400
	)

	records, err := s.GetRechargeHistory(ctx, teamName, fetchRecords)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	windowStart := now.AddDate(0, 0, -lookbackDays)

	var totalDeductions float64
	var oldestDeduction time.Time
	hasDeduction := false

	for _, record := range records {
		if record.Type != "deduction" || !record.Timestamp.After(windowStart) {
			continue
		}
		totalDeductions += -record.Amount // Amount is negative for deductions
		if !hasDeduction || record.Timestamp.Before(oldestDeduction) {
			oldestDeduction = record.Timestamp
			hasDeduction = true
		}
	}

	if !hasDeduction || totalDeductions <= 0 {
		return 0, nil
	}

	spanDays := now.Sub(oldestDeduction).Hours() / 24
	if spanDays > maxSpanDays {
		spanDays = maxSpanDays
	}
	if spanDays < minSpanDays {
		spanDays = minSpanDays
	}

	return totalDeductions / spanDays, nil
}

// SetOverdueAt records (or clears) when a team first went into negative balance.
// The update is conflict-retried and preserves the current amount, so it cannot
// clobber a concurrent deduction/recharge.
func (s *BalanceService) SetOverdueAt(ctx context.Context, teamName string, overdueAt *time.Time) error {
	return s.mutateConfigMap(ctx, BalancesConfigMap, func(cm *corev1.ConfigMap) error {
		balance := Balance{TeamName: teamName}
		if existing, ok := cm.Data[teamName]; ok {
			if err := json.Unmarshal([]byte(existing), &balance); err != nil {
				return fmt.Errorf("failed to parse balance: %w", err)
			}
		}
		balance.TeamName = teamName
		balance.OverdueAt = overdueAt

		data, err := json.Marshal(&balance)
		if err != nil {
			return fmt.Errorf("failed to marshal balance: %w", err)
		}
		cm.Data[teamName] = string(data)
		return nil
	})
}

// GetBalanceWithEstimate returns the balance with consumption and estimated overdue time calculated
func (s *BalanceService) GetBalanceWithEstimate(ctx context.Context, teamName string) (*Balance, error) {
	balance, err := s.GetBalance(ctx, teamName)
	if err != nil {
		return nil, err
	}

	// Calculate daily consumption
	dailyConsumption, err := s.CalculateDailyConsumption(ctx, teamName)
	if err != nil {
		logger.Warn("Failed to calculate daily consumption", "team", teamName, "error", err)
	}
	balance.DailyConsumption = dailyConsumption

	// Calculate estimated overdue time (only if balance is positive and there's consumption)
	if balance.Amount > 0 && dailyConsumption > 0 {
		daysRemaining := balance.Amount / dailyConsumption
		estimatedOverdue := time.Now().Add(time.Duration(daysRemaining*24) * time.Hour)
		balance.EstimatedOverdueAt = &estimatedOverdue
	}

	return balance, nil
}

// CalculateGraceRemaining calculates the remaining grace period for a team
func (s *BalanceService) CalculateGraceRemaining(overdueAt *time.Time, gracePeriodValue int, gracePeriodUnit string) string {
	if overdueAt == nil {
		return ""
	}

	// Calculate grace period end time
	var gracePeriodEnd time.Time
	if gracePeriodUnit == "hours" {
		gracePeriodEnd = overdueAt.Add(time.Duration(gracePeriodValue) * time.Hour)
	} else { // days
		gracePeriodEnd = overdueAt.AddDate(0, 0, gracePeriodValue)
	}

	remaining := time.Until(gracePeriodEnd)
	if remaining <= 0 {
		return "已到期"
	}

	days := int(remaining.Hours() / 24)
	hours := int(remaining.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时", days, hours)
	}
	return fmt.Sprintf("%d小时", hours)
}

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/pkg/logger"
)

const (
	BillingConfigMap = "bison-billing-config"
	// lastBilledKey stores (in the billing ConfigMap) the RFC3339 timestamp of the
	// last successful billing run, so billing is not duplicated when the ticker
	// fires more often than the configured interval or after a process restart.
	lastBilledKey = "lastBilledAt"
)

// BillingConfig represents the billing configuration
type BillingConfig struct {
	Enabled          bool                     `json:"enabled"`
	Interval         int                      `json:"interval"`         // Billing interval in hours
	Currency         string                   `json:"currency"`         // e.g., "CNY", "USD"
	CurrencySymbol   string                   `json:"currencySymbol"`   // e.g., "¥", "$"
	Pricing          map[string]ResourcePrice `json:"pricing"`          // Resource pricing
	GracePeriodValue int                      `json:"gracePeriodValue"` // Grace period value (e.g., 7)
	GracePeriodUnit  string                   `json:"gracePeriodUnit"`  // Grace period unit: "hours" or "days"
}

// ResourcePrice represents the price for a resource
type ResourcePrice struct {
	Price float64 `json:"price"` // Price per unit per hour
	Unit  string  `json:"unit"`  // e.g., "核·时", "GB·时", "卡·时"
}

// Bill represents a team/project/user bill
type Bill struct {
	Name          string             `json:"name"`
	Window        string             `json:"window"`
	TotalCost     float64            `json:"totalCost"`
	ResourceCosts map[string]float64 `json:"resourceCosts"` // Cost breakdown by resource
	UsageDetails  *UsageData         `json:"usageDetails"`
	GeneratedAt   time.Time          `json:"generatedAt"`
}

// BillingService handles billing operations
type BillingService struct {
	k8sClient         *k8s.Client
	opencostClient    *opencost.Client
	balanceSvc        *BalanceService
	tenantSvc         *TenantService
	projectSvc        *ProjectService
	resourceConfigSvc *ResourceConfigService
}

// NewBillingService creates a new BillingService
func NewBillingService(
	k8sClient *k8s.Client,
	opencostClient *opencost.Client,
	balanceSvc *BalanceService,
	tenantSvc *TenantService,
	projectSvc *ProjectService,
	resourceConfigSvc *ResourceConfigService,
) *BillingService {
	return &BillingService{
		k8sClient:         k8sClient,
		opencostClient:    opencostClient,
		balanceSvc:        balanceSvc,
		tenantSvc:         tenantSvc,
		projectSvc:        projectSvc,
		resourceConfigSvc: resourceConfigSvc,
	}
}

// GetConfig returns the billing configuration
func (s *BillingService) GetConfig(ctx context.Context) (*BillingConfig, error) {
	logger.Debug("Getting billing config")

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, BillingConfigMap)
	if err != nil {
		// Return default config if not found
		return s.getDefaultConfig(), nil
	}

	data, ok := cm.Data["config"]
	if !ok {
		return s.getDefaultConfig(), nil
	}

	var config BillingConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		logger.Error("Failed to unmarshal billing config", "error", err)
		return s.getDefaultConfig(), nil
	}

	return &config, nil
}

// SetConfig sets the billing configuration
func (s *BillingService) SetConfig(ctx context.Context, config *BillingConfig) error {
	logger.Info("Setting billing config")

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, BillingConfigMap)
	if err != nil {
		// Create if not exists
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      BillingConfigMap,
				Namespace: BisonNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bison",
					"app.kubernetes.io/component": "billing",
				},
			},
			Data: map[string]string{
				"config": string(data),
			},
		}
		return s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config"] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

// ProcessBilling processes billing for all teams
func (s *BillingService) ProcessBilling(ctx context.Context) error {
	logger.Info("Processing billing")

	config, err := s.GetConfig(ctx)
	if err != nil {
		return err
	}

	if !config.Enabled {
		logger.Debug("Billing is disabled")
		return nil
	}

	// Enforce the configured billing interval regardless of how often the
	// scheduler ticks, and survive restarts, by gating on a persisted timestamp.
	interval := config.Interval
	if interval <= 0 {
		interval = 1
	}
	minGap := time.Duration(interval) * time.Hour
	now := time.Now()
	lastBilled, _ := s.getLastBilled(ctx)
	if lastBilled.IsZero() {
		// First run on a fresh deployment: establish a baseline instead of billing
		// an unknown historical window.
		if err := s.setLastBilled(ctx, now); err != nil {
			logger.Warn("Failed to initialize billing baseline", "error", err)
		}
		logger.Info("Billing baseline initialized; skipping first cycle")
		return nil
	}
	// Tolerate scheduler jitter: require ~95% of the interval to have elapsed.
	if now.Sub(lastBilled) < time.Duration(float64(minGap)*0.95) {
		logger.Debug("Skipping billing: interval not yet elapsed",
			"sinceLastBilled", now.Sub(lastBilled).String(), "interval", minGap.String())
		return nil
	}

	// Get usage from OpenCost
	if s.opencostClient == nil || !s.opencostClient.IsEnabled() {
		logger.Warn("OpenCost not available, skipping billing")
		return nil
	}

	// Get usage for the billing interval
	window := fmt.Sprintf("%dh", config.Interval)
	allocations, err := s.opencostClient.GetAllocationByNamespace(ctx, window)
	if err != nil {
		logger.Error("Failed to get allocations", "error", err)
		return err
	}

	// Get all teams
	teams, err := s.tenantSvc.List(ctx)
	if err != nil {
		logger.Error("Failed to list teams", "error", err)
		return err
	}

	// Map namespace to team, and remember which teams are already suspended so we
	// don't re-run scale-down/orphan-pod cleanup for them every billing cycle.
	nsToTeam := make(map[string]string)
	suspendedByName := make(map[string]bool)
	for _, team := range teams {
		suspendedByName[team.Name] = team.Suspended
		projects, _ := s.projectSvc.ListByTeam(ctx, team.Name)
		for _, project := range projects {
			nsToTeam[project.Name] = team.Name
		}
	}

	// Aggregate costs by team. Prices are read once for the whole run.
	prices := s.loadPrices(ctx)
	teamCosts := make(map[string]float64)
	for _, alloc := range allocations {
		teamName, ok := nsToTeam[alloc.Name]
		if !ok {
			continue
		}

		cost := costFromPrices(config, prices, &alloc)
		teamCosts[teamName] += cost
	}

	// Deduct costs from team balances
	for teamName, cost := range teamCosts {
		if cost <= 0 {
			continue
		}

		reason := fmt.Sprintf("Usage billing for %s", window)
		// Deduct returns the authoritative post-write balance, so the suspension
		// decision below is no longer based on a racy second read.
		newBalance, err := s.balanceSvc.Deduct(ctx, teamName, cost, reason)
		if err != nil {
			logger.Error("Failed to deduct balance", "team", teamName, "cost", cost, "error", err)
			continue
		}

		if newBalance < 0 {
			logger.Warn("Team is in debt", "team", teamName, "balance", newBalance)

			// Determine the overdue start time, preserving any existing marker so the
			// grace period is measured from when the balance first went negative.
			cur, err := s.balanceSvc.GetBalance(ctx, teamName)
			if err != nil {
				logger.Error("Failed to read balance for overdue check", "team", teamName, "error", err)
				continue
			}
			overdueAt := cur.OverdueAt
			if overdueAt == nil {
				now := time.Now()
				overdueAt = &now
				if err := s.balanceSvc.SetOverdueAt(ctx, teamName, overdueAt); err != nil {
					logger.Error("Failed to set overdue time", "team", teamName, "error", err)
				}
			}

			// Check if grace period has passed
			if s.isGracePeriodExpired(config, overdueAt) {
				if suspendedByName[teamName] {
					logger.Debug("Team already suspended; skipping re-suspend", "team", teamName)
				} else {
					logger.Warn("Grace period expired, suspending team", "team", teamName, "overdueAt", overdueAt)
					if err := s.SuspendTeam(ctx, teamName); err != nil {
						logger.Error("Failed to suspend team", "team", teamName, "error", err)
					}
				}
			} else {
				remaining := s.balanceSvc.CalculateGraceRemaining(overdueAt, config.GracePeriodValue, config.GracePeriodUnit)
				logger.Info("Team in grace period", "team", teamName, "remaining", remaining)
			}
		} else {
			// Balance is non-negative again, clear any overdue marker.
			if cur, err := s.balanceSvc.GetBalance(ctx, teamName); err == nil && cur.OverdueAt != nil {
				if err := s.balanceSvc.SetOverdueAt(ctx, teamName, nil); err != nil {
					logger.Error("Failed to clear overdue time", "team", teamName, "error", err)
				}
			}
		}
	}

	// Record successful billing time so the next cycle bills the correct window.
	if err := s.setLastBilled(ctx, now); err != nil {
		logger.Error("Failed to update last-billed timestamp", "error", err)
	}

	return nil
}

// getLastBilled returns the timestamp of the last successful billing run, or the
// zero time if none has been recorded yet.
func (s *BillingService) getLastBilled(ctx context.Context) (time.Time, error) {
	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, BillingConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	v, ok := cm.Data[lastBilledKey]
	if !ok || v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		logger.Warn("Invalid lastBilledAt timestamp, treating as unset", "value", v)
		return time.Time{}, nil
	}
	return t, nil
}

// setLastBilled persists the last successful billing time, using optimistic
// concurrency so it cannot clobber a concurrent config update.
func (s *BillingService) setLastBilled(ctx context.Context, t time.Time) error {
	value := t.UTC().Format(time.RFC3339)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, BillingConfigMap)
		if err != nil {
			if errors.IsNotFound(err) {
				cm = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      BillingConfigMap,
						Namespace: BisonNamespace,
						Labels: map[string]string{
							"app.kubernetes.io/name":      "bison",
							"app.kubernetes.io/component": "billing",
						},
					},
					Data: map[string]string{lastBilledKey: value},
				}
				return s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm)
			}
			return err
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[lastBilledKey] = value
		return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
	})
}

// isGracePeriodExpired checks if the grace period has expired for a team
func (s *BillingService) isGracePeriodExpired(config *BillingConfig, overdueAt *time.Time) bool {
	if overdueAt == nil {
		return false
	}

	var gracePeriodEnd time.Time
	if config.GracePeriodUnit == "hours" {
		gracePeriodEnd = overdueAt.Add(time.Duration(config.GracePeriodValue) * time.Hour)
	} else { // days
		gracePeriodEnd = overdueAt.AddDate(0, 0, config.GracePeriodValue)
	}

	return time.Now().After(gracePeriodEnd)
}

// GetTeamBill returns a bill for a specific team
func (s *BillingService) GetTeamBill(ctx context.Context, teamName, window string) (*Bill, error) {
	if window == "" {
		window = "7d"
	}

	// Get projects for this team
	projects, err := s.projectSvc.ListByTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}

	// Get allocations for each project
	var totalCost float64
	var totalUsage UsageData
	resourceCosts := make(map[string]float64)

	config, _ := s.GetConfig(ctx)
	prices := s.loadPrices(ctx)

	if s.opencostClient != nil && s.opencostClient.IsEnabled() {
		for _, project := range projects {
			allocations, err := s.opencostClient.GetAllocationForNamespace(ctx, window, project.Name)
			if err != nil {
				logger.Warn("Failed to get allocations for project", "project", project.Name, "error", err)
				continue
			}

			for _, alloc := range allocations {
				totalUsage.CPUCoreHours += alloc.CPUCoreHours
				totalUsage.RAMGBHours += alloc.RAMGBHours
				totalUsage.GPUHours += alloc.GPUHours
				totalUsage.Minutes += alloc.Minutes

				cost := costFromPrices(config, prices, &alloc)
				totalCost += cost

				resourceCosts["cpu"] += alloc.CPUCost
				resourceCosts["memory"] += alloc.RAMCost
				resourceCosts["gpu"] += alloc.GPUCost
			}
		}
	}

	totalUsage.Name = teamName
	totalUsage.TotalCost = totalCost

	return &Bill{
		Name:          teamName,
		Window:        window,
		TotalCost:     totalCost,
		ResourceCosts: resourceCosts,
		UsageDetails:  &totalUsage,
		GeneratedAt:   time.Now(),
	}, nil
}

// GetProjectBill returns a bill for a specific project
func (s *BillingService) GetProjectBill(ctx context.Context, projectName, window string) (*Bill, error) {
	if window == "" {
		window = "7d"
	}

	var totalCost float64
	var usage UsageData
	resourceCosts := make(map[string]float64)

	config, _ := s.GetConfig(ctx)
	prices := s.loadPrices(ctx)

	if s.opencostClient != nil && s.opencostClient.IsEnabled() {
		allocations, err := s.opencostClient.GetAllocationForNamespace(ctx, window, projectName)
		if err != nil {
			return nil, err
		}

		for _, alloc := range allocations {
			usage.CPUCoreHours += alloc.CPUCoreHours
			usage.RAMGBHours += alloc.RAMGBHours
			usage.GPUHours += alloc.GPUHours
			usage.Minutes += alloc.Minutes

			cost := costFromPrices(config, prices, &alloc)
			totalCost += cost

			resourceCosts["cpu"] += alloc.CPUCost
			resourceCosts["memory"] += alloc.RAMCost
			resourceCosts["gpu"] += alloc.GPUCost
		}
	}

	usage.Name = projectName
	usage.TotalCost = totalCost

	return &Bill{
		Name:          projectName,
		Window:        window,
		TotalCost:     totalCost,
		ResourceCosts: resourceCosts,
		UsageDetails:  &usage,
		GeneratedAt:   time.Now(),
	}, nil
}

// SuspendTeam suspends a team due to insufficient balance
func (s *BillingService) SuspendTeam(ctx context.Context, teamName string) error {
	logger.Info("Suspending team", "team", teamName)

	// Mark team as suspended
	if err := s.tenantSvc.SetSuspended(ctx, teamName, true); err != nil {
		return err
	}

	// Get all projects for this team
	projects, err := s.projectSvc.ListByTeam(ctx, teamName)
	if err != nil {
		return err
	}

	// Scale down all deployments and statefulsets in each project
	for _, project := range projects {
		if err := s.scaleDownNamespace(ctx, project.Name); err != nil {
			logger.Error("Failed to scale down namespace", "namespace", project.Name, "error", err)
		}
	}

	return nil
}

// ResumeTeam resumes a suspended team
func (s *BillingService) ResumeTeam(ctx context.Context, teamName string) error {
	logger.Info("Resuming team", "team", teamName)

	// Check balance
	balance, err := s.balanceSvc.GetBalance(ctx, teamName)
	if err != nil {
		return err
	}

	if balance.Amount < 0 {
		return fmt.Errorf("cannot resume team with negative balance: %.2f", balance.Amount)
	}

	// Mark team as not suspended
	if err := s.tenantSvc.SetSuspended(ctx, teamName, false); err != nil {
		return err
	}

	// Get all projects for this team
	projects, err := s.projectSvc.ListByTeam(ctx, teamName)
	if err != nil {
		return err
	}

	// Scale up all deployments and statefulsets in each project
	for _, project := range projects {
		if err := s.scaleUpNamespace(ctx, project.Name); err != nil {
			logger.Error("Failed to scale up namespace", "namespace", project.Name, "error", err)
		}
	}

	return nil
}

// GetSuspendedTeams returns list of suspended teams
func (s *BillingService) GetSuspendedTeams(ctx context.Context) ([]string, error) {
	teams, err := s.tenantSvc.List(ctx)
	if err != nil {
		return nil, err
	}

	var suspended []string
	for _, team := range teams {
		if team.Suspended {
			suspended = append(suspended, team.Name)
		}
	}

	return suspended, nil
}

// Helper methods

func (s *BillingService) getDefaultConfig() *BillingConfig {
	return &BillingConfig{
		Enabled:          true,
		Interval:         1, // 1 hour
		Currency:         "CNY",
		CurrencySymbol:   "¥",
		GracePeriodValue: 3, // 3 days by default
		GracePeriodUnit:  "days",
		Pricing: map[string]ResourcePrice{
			"cpu":    {Price: 0.1, Unit: "核·时"},
			"memory": {Price: 0.05, Unit: "GB·时"},
		},
	}
}

// resourcePrices holds the per-unit prices used for cost computation, resolved
// once per billing/report operation instead of per allocation row.
type resourcePrices struct {
	cpu         float64
	memory      float64
	accelerator float64
}

// loadPrices reads the enabled resource configs once and builds a price table.
func (s *BillingService) loadPrices(ctx context.Context) resourcePrices {
	resourceConfigs, _ := s.resourceConfigSvc.GetEnabledResourceConfigs(ctx)
	var p resourcePrices
	for _, rc := range resourceConfigs {
		if rc.Price <= 0 {
			continue
		}
		switch rc.Name {
		case "cpu":
			p.cpu = rc.Price
		case "memory":
			p.memory = rc.Price
		default:
			// For accelerators (any non-cpu/memory resource), use the first priced one.
			if rc.Category == CategoryAccelerator && p.accelerator == 0 {
				p.accelerator = rc.Price
			}
		}
	}
	return p
}

// costFromPrices computes the cost of a single allocation from a precomputed price table.
func costFromPrices(config *BillingConfig, p resourcePrices, alloc *opencost.Allocation) float64 {
	if config == nil || !config.Enabled {
		return alloc.TotalCost
	}

	var cost float64
	if p.cpu > 0 {
		cost += alloc.CPUCoreHours * p.cpu
	} else {
		cost += alloc.CPUCost
	}
	if p.memory > 0 {
		cost += alloc.RAMGBHours * p.memory
	} else {
		cost += alloc.RAMCost
	}
	// OpenCost reports all accelerators as GPUHours.
	if p.accelerator > 0 {
		cost += alloc.GPUHours * p.accelerator
	} else {
		cost += alloc.GPUCost
	}
	return cost
}

// calculateCost computes the cost of a single allocation, loading prices each call.
// In loops prefer loadPrices + costFromPrices to avoid repeated ConfigMap reads.
func (s *BillingService) calculateCost(ctx context.Context, config *BillingConfig, alloc *opencost.Allocation) float64 {
	if config == nil || !config.Enabled {
		return alloc.TotalCost
	}
	return costFromPrices(config, s.loadPrices(ctx), alloc)
}

func (s *BillingService) scaleDownNamespace(ctx context.Context, namespace string) error {
	// Scale down deployments
	deployments, err := s.k8sClient.ListDeployments(ctx, namespace)
	if err != nil {
		return err
	}

	for _, deploy := range deployments.Items {
		if *deploy.Spec.Replicas == 0 {
			continue
		}

		// Save original replicas
		if deploy.Annotations == nil {
			deploy.Annotations = make(map[string]string)
		}
		deploy.Annotations["bison.io/original-replicas"] = fmt.Sprintf("%d", *deploy.Spec.Replicas)

		// Scale to 0
		zero := int32(0)
		deploy.Spec.Replicas = &zero

		if err := s.k8sClient.UpdateDeployment(ctx, namespace, &deploy); err != nil {
			logger.Error("Failed to scale down deployment", "namespace", namespace, "name", deploy.Name, "error", err)
		}
	}

	// Scale down statefulsets
	statefulsets, err := s.k8sClient.ListStatefulSets(ctx, namespace)
	if err != nil {
		return err
	}

	for _, sts := range statefulsets.Items {
		if *sts.Spec.Replicas == 0 {
			continue
		}

		// Save original replicas
		if sts.Annotations == nil {
			sts.Annotations = make(map[string]string)
		}
		sts.Annotations["bison.io/original-replicas"] = fmt.Sprintf("%d", *sts.Spec.Replicas)

		// Scale to 0
		zero := int32(0)
		sts.Spec.Replicas = &zero

		if err := s.k8sClient.UpdateStatefulSet(ctx, namespace, &sts); err != nil {
			logger.Error("Failed to scale down statefulset", "namespace", namespace, "name", sts.Name, "error", err)
		}
	}

	// Delete orphan pods (pods not managed by a controller)
	pods, err := s.k8sClient.ListPods(ctx, namespace, "")
	if err != nil {
		logger.Error("Failed to list pods", "namespace", namespace, "error", err)
		return nil // Don't fail the whole operation
	}

	for _, pod := range pods.Items {
		// Check if pod is managed by a controller
		if len(pod.OwnerReferences) == 0 {
			// Orphan pod - delete it
			logger.Info("Deleting orphan pod", "namespace", namespace, "name", pod.Name)
			if err := s.k8sClient.DeletePod(ctx, namespace, pod.Name); err != nil {
				logger.Error("Failed to delete orphan pod", "namespace", namespace, "name", pod.Name, "error", err)
			}
		}
	}

	return nil
}

func (s *BillingService) scaleUpNamespace(ctx context.Context, namespace string) error {
	// Scale up deployments
	deployments, err := s.k8sClient.ListDeployments(ctx, namespace)
	if err != nil {
		return err
	}

	for _, deploy := range deployments.Items {
		originalStr, ok := deploy.Annotations["bison.io/original-replicas"]
		if !ok {
			continue
		}

		original, err := strconv.ParseInt(originalStr, 10, 32)
		if err != nil {
			continue
		}

		// Restore original replicas
		replicas := int32(original)
		deploy.Spec.Replicas = &replicas
		delete(deploy.Annotations, "bison.io/original-replicas")

		if err := s.k8sClient.UpdateDeployment(ctx, namespace, &deploy); err != nil {
			logger.Error("Failed to scale up deployment", "namespace", namespace, "name", deploy.Name, "error", err)
		}
	}

	// Scale up statefulsets
	statefulsets, err := s.k8sClient.ListStatefulSets(ctx, namespace)
	if err != nil {
		return err
	}

	for _, sts := range statefulsets.Items {
		originalStr, ok := sts.Annotations["bison.io/original-replicas"]
		if !ok {
			continue
		}

		original, err := strconv.ParseInt(originalStr, 10, 32)
		if err != nil {
			continue
		}

		// Restore original replicas
		replicas := int32(original)
		sts.Spec.Replicas = &replicas
		delete(sts.Annotations, "bison.io/original-replicas")

		if err := s.k8sClient.UpdateStatefulSet(ctx, namespace, &sts); err != nil {
			logger.Error("Failed to scale up statefulset", "namespace", namespace, "name", sts.Name, "error", err)
		}
	}

	return nil
}

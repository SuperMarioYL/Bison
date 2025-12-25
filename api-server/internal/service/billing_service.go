package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/pkg/logger"
)

const (
	BillingConfigMap = "bison-billing-config"
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
	Name        string             `json:"name"`
	Window      string             `json:"window"`
	TotalCost   float64            `json:"totalCost"`
	ResourceCosts map[string]float64 `json:"resourceCosts"` // Cost breakdown by resource
	UsageDetails  *UsageData       `json:"usageDetails"`
	GeneratedAt time.Time          `json:"generatedAt"`
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

	// Map namespace to team
	nsToTeam := make(map[string]string)
	for _, team := range teams {
		projects, _ := s.projectSvc.ListByTeam(ctx, team.Name)
		for _, project := range projects {
			nsToTeam[project.Name] = team.Name
		}
	}

	// Aggregate costs by team
	teamCosts := make(map[string]float64)
	for _, alloc := range allocations {
		teamName, ok := nsToTeam[alloc.Name]
		if !ok {
			continue
		}

		// Calculate cost based on pricing config
		cost := s.calculateCost(ctx, config, &alloc)
		teamCosts[teamName] += cost
	}

	// Deduct costs from team balances
	for teamName, cost := range teamCosts {
		if cost <= 0 {
			continue
		}

		reason := fmt.Sprintf("Usage billing for %s", window)
		if err := s.balanceSvc.Deduct(ctx, teamName, cost, reason); err != nil {
			logger.Error("Failed to deduct balance", "team", teamName, "cost", cost, "error", err)
			continue
		}

		// Check if team is now in debt
		balance, _ := s.balanceSvc.GetBalance(ctx, teamName)
		if balance != nil && balance.Amount < 0 {
			logger.Warn("Team is in debt", "team", teamName, "balance", balance.Amount)

			// Record when balance first went negative
			if balance.OverdueAt == nil {
				now := time.Now()
				if err := s.balanceSvc.SetOverdueAt(ctx, teamName, &now); err != nil {
					logger.Error("Failed to set overdue time", "team", teamName, "error", err)
				}
				balance.OverdueAt = &now
			}

			// Check if grace period has passed
			if s.isGracePeriodExpired(config, balance.OverdueAt) {
				logger.Warn("Grace period expired, suspending team", "team", teamName, "overdueAt", balance.OverdueAt)
				if err := s.SuspendTeam(ctx, teamName); err != nil {
					logger.Error("Failed to suspend team", "team", teamName, "error", err)
				}
			} else {
				remaining := s.balanceSvc.CalculateGraceRemaining(balance.OverdueAt, config.GracePeriodValue, config.GracePeriodUnit)
				logger.Info("Team in grace period", "team", teamName, "remaining", remaining)
			}
		} else if balance != nil && balance.Amount >= 0 && balance.OverdueAt != nil {
			// Balance is positive again, clear overdue time
			if err := s.balanceSvc.SetOverdueAt(ctx, teamName, nil); err != nil {
				logger.Error("Failed to clear overdue time", "team", teamName, "error", err)
			}
		}
	}

	return nil
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

				cost := s.calculateCost(ctx, config, &alloc)
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

			cost := s.calculateCost(ctx, config, &alloc)
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

func (s *BillingService) calculateCost(ctx context.Context, config *BillingConfig, alloc *opencost.Allocation) float64 {
	if config == nil || !config.Enabled {
		return alloc.TotalCost
	}

	var cost float64

	// Get resource configs for pricing
	resourceConfigs, _ := s.resourceConfigSvc.GetEnabledResourceConfigs(ctx)

	// Build price map and find accelerator price
	cpuPrice := float64(0)
	memoryPrice := float64(0)
	acceleratorPrice := float64(0)

	for _, rc := range resourceConfigs {
		if rc.Price <= 0 {
			continue
		}
		switch rc.Name {
		case "cpu":
			cpuPrice = rc.Price
		case "memory":
			memoryPrice = rc.Price
		default:
			// For accelerators (any non-cpu/memory resource), use the first one with price
			if rc.Category == CategoryAccelerator && acceleratorPrice == 0 {
				acceleratorPrice = rc.Price
			}
		}
	}

	// CPU cost
	if cpuPrice > 0 {
		cost += alloc.CPUCoreHours * cpuPrice
	} else {
		cost += alloc.CPUCost
	}

	// Memory cost
	if memoryPrice > 0 {
		cost += alloc.RAMGBHours * memoryPrice
	} else {
		cost += alloc.RAMCost
	}

	// GPU/Accelerator cost (OpenCost reports all accelerators as GPUHours)
	if acceleratorPrice > 0 {
		cost += alloc.GPUHours * acceleratorPrice
	} else {
		cost += alloc.GPUCost
	}

	return cost
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

// Unused import fix
var _ = appsv1.Deployment{}


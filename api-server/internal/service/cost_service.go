package service

import (
	"context"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/pkg/logger"
)

// UsageData represents usage statistics for an entity
type UsageData struct {
	Name         string  `json:"name"`
	CPUCoreHours float64 `json:"cpuCoreHours"`
	RAMGBHours   float64 `json:"ramGBHours"`
	GPUHours     float64 `json:"gpuHours"`
	TotalCost    float64 `json:"totalCost"`
	CPUCost      float64 `json:"cpuCost"`
	RAMCost      float64 `json:"ramCost"`
	GPUCost      float64 `json:"gpuCost"`
	Minutes      float64 `json:"minutes"`
}

// UsageReport represents a usage report
type UsageReport struct {
	Window      string       `json:"window"`
	AggregateBy string       `json:"aggregateBy"`
	Data        []*UsageData `json:"data"`
	TotalCost   float64      `json:"totalCost"`
}

// CostService handles cost and usage statistics using OpenCost
type CostService struct {
	opencostClient *opencost.Client
	k8sClient      *k8s.Client
	enabled        bool
}

// NewCostService creates a new CostService
func NewCostService(opencostURL string, k8sClient *k8s.Client) *CostService {
	if opencostURL == "" {
		logger.Warn("OpenCost URL not configured, cost service disabled")
		return &CostService{enabled: false, k8sClient: k8sClient}
	}

	client := opencost.NewClient(opencostURL)
	logger.Info("OpenCost client initialized", "url", opencostURL)

	return &CostService{
		opencostClient: client,
		k8sClient:      k8sClient,
		enabled:        true,
	}
}

// IsEnabled returns whether the cost service is enabled
func (s *CostService) IsEnabled() bool {
	return s.enabled
}

// GetClient returns the OpenCost client
func (s *CostService) GetClient() *opencost.Client {
	return s.opencostClient
}

// GetTeamUsage returns usage statistics for all teams (aggregated from namespaces)
func (s *CostService) GetTeamUsage(ctx context.Context, window string) (*UsageReport, error) {
	if !s.enabled {
		return &UsageReport{
			Window:      window,
			AggregateBy: "team",
			Data:        []*UsageData{},
		}, nil
	}

	if window == "" {
		window = "7d"
	}

	logger.Debug("Getting team usage", "window", window)

	// Get namespace-level usage from OpenCost
	summaries, err := s.opencostClient.GetProjectUsage(ctx, window)
	if err != nil {
		logger.Error("Failed to get namespace usage", "error", err)
		return nil, err
	}

	// Build namespace to team mapping from Capsule Tenants
	nsToTeam := make(map[string]string)
	if s.k8sClient != nil {
		tenantList, err := s.k8sClient.ListTenants(ctx)
		if err != nil {
			logger.Warn("Failed to list Capsule tenants for team mapping", "error", err)
		} else {
			for _, tenant := range tenantList.Items {
				teamName := tenant.GetName()
				// Get namespaces belonging to this tenant from status
				if status, ok := tenant.Object["status"].(map[string]interface{}); ok {
					if namespaces, ok := status["namespaces"].([]interface{}); ok {
						for _, ns := range namespaces {
							if nsName, ok := ns.(string); ok {
								nsToTeam[nsName] = teamName
							}
						}
					}
				}
			}
		}
	}

	// Aggregate by team
	teamData := make(map[string]*UsageData)
	for _, summary := range summaries {
		if summary.Name == "__idle__" || summary.Name == "__unmounted__" {
			continue
		}

		// Find the team for this namespace
		teamName := nsToTeam[summary.Name]
		if teamName == "" {
			teamName = "未分配" // Namespace not belonging to any team
		}

		if _, exists := teamData[teamName]; !exists {
			teamData[teamName] = &UsageData{Name: teamName}
		}

		// Aggregate the usage
		teamData[teamName].CPUCoreHours += summary.CPUCoreHours
		teamData[teamName].RAMGBHours += summary.RAMGBHours
		teamData[teamName].GPUHours += summary.GPUHours
		teamData[teamName].TotalCost += summary.TotalCost
		teamData[teamName].CPUCost += summary.CPUCost
		teamData[teamName].RAMCost += summary.RAMCost
		teamData[teamName].GPUCost += summary.GPUCost
		teamData[teamName].Minutes += summary.Minutes
	}

	// Convert to report
	report := &UsageReport{
		Window:      window,
		AggregateBy: "team",
		Data:        make([]*UsageData, 0, len(teamData)),
	}
	for _, data := range teamData {
		report.Data = append(report.Data, data)
		report.TotalCost += data.TotalCost
	}

	return report, nil
}

// GetProjectUsage returns usage statistics for all projects
func (s *CostService) GetProjectUsage(ctx context.Context, window string) (*UsageReport, error) {
	if !s.enabled {
		return &UsageReport{
			Window:      window,
			AggregateBy: "project",
			Data:        []*UsageData{},
		}, nil
	}

	if window == "" {
		window = "7d"
	}

	logger.Debug("Getting project usage", "window", window)

	summaries, err := s.opencostClient.GetProjectUsage(ctx, window)
	if err != nil {
		logger.Error("Failed to get project usage", "error", err)
		return nil, err
	}

	return s.summariesToReport(summaries, window, "project"), nil
}

// GetUserUsage returns usage statistics for all users
func (s *CostService) GetUserUsage(ctx context.Context, window string) (*UsageReport, error) {
	if !s.enabled {
		return &UsageReport{
			Window:      window,
			AggregateBy: "user",
			Data:        []*UsageData{},
		}, nil
	}

	if window == "" {
		window = "7d"
	}

	logger.Debug("Getting user usage", "window", window)

	summaries, err := s.opencostClient.GetUserUsage(ctx, window)
	if err != nil {
		logger.Error("Failed to get user usage", "error", err)
		return nil, err
	}

	return s.summariesToReport(summaries, window, "user"), nil
}

// GetTeamUsageByName returns usage statistics for a specific team
func (s *CostService) GetTeamUsageByName(ctx context.Context, teamName, window string) (*UsageData, error) {
	report, err := s.GetTeamUsage(ctx, window)
	if err != nil {
		return nil, err
	}

	for _, data := range report.Data {
		if data.Name == teamName {
			return data, nil
		}
	}

	// Return empty data if team not found
	return &UsageData{Name: teamName}, nil
}

// GetProjectUsageByName returns usage statistics for a specific project
func (s *CostService) GetProjectUsageByName(ctx context.Context, projectName, window string) (*UsageData, error) {
	report, err := s.GetProjectUsage(ctx, window)
	if err != nil {
		return nil, err
	}

	for _, data := range report.Data {
		if data.Name == projectName {
			return data, nil
		}
	}

	// Return empty data if project not found
	return &UsageData{Name: projectName}, nil
}

// GetTotalCost returns the total cost for a window
func (s *CostService) GetTotalCost(ctx context.Context, window string) (float64, error) {
	if !s.enabled {
		return 0, nil
	}

	return s.opencostClient.GetTotalCost(ctx, window)
}

// CostTrendPoint represents a daily cost point
type CostTrendPoint struct {
	Date      string  `json:"date"`
	TotalCost float64 `json:"totalCost"`
}

// GetCostTrend returns daily cost trend data
func (s *CostService) GetCostTrend(ctx context.Context, window string) ([]CostTrendPoint, error) {
	if !s.enabled {
		return []CostTrendPoint{}, nil
	}

	trend, err := s.opencostClient.GetCostTrend(ctx, window)
	if err != nil {
		return nil, err
	}

	result := make([]CostTrendPoint, 0, len(trend))
	for _, point := range trend {
		result = append(result, CostTrendPoint{
			Date:      point.Date,
			TotalCost: point.TotalCost,
		})
	}

	return result, nil
}

// summariesToReport converts OpenCost summaries to a UsageReport
func (s *CostService) summariesToReport(summaries []opencost.UsageSummary, window, aggregateBy string) *UsageReport {
	report := &UsageReport{
		Window:      window,
		AggregateBy: aggregateBy,
		Data:        make([]*UsageData, 0, len(summaries)),
	}

	for _, summary := range summaries {
		// Skip idle/system entries
		if summary.Name == "__idle__" || summary.Name == "__unmounted__" {
			continue
		}

		data := &UsageData{
			Name:         summary.Name,
			CPUCoreHours: summary.CPUCoreHours,
			RAMGBHours:   summary.RAMGBHours,
			GPUHours:     summary.GPUHours,
			TotalCost:    summary.TotalCost,
			CPUCost:      summary.CPUCost,
			RAMCost:      summary.RAMCost,
			GPUCost:      summary.GPUCost,
			Minutes:      summary.Minutes,
		}
		report.Data = append(report.Data, data)
		report.TotalCost += summary.TotalCost
	}

	return report
}

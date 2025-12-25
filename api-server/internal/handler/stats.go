package handler

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// StatsHandler handles statistics-related API requests
type StatsHandler struct {
	k8sClient   *k8s.Client
	tenantSvc   *service.TenantService
	projectSvc  *service.ProjectService
	costSvc     *service.CostService
	resourceSvc *service.ResourceService
	nodeSvc     *service.NodeService
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(k8sClient *k8s.Client, tenantSvc *service.TenantService, projectSvc *service.ProjectService, costSvc *service.CostService, resourceSvc *service.ResourceService, nodeSvc *service.NodeService) *StatsHandler {
	return &StatsHandler{
		k8sClient:   k8sClient,
		tenantSvc:   tenantSvc,
		projectSvc:  projectSvc,
		costSvc:     costSvc,
		resourceSvc: resourceSvc,
		nodeSvc:     nodeSvc,
	}
}

// Overview represents the dashboard overview
type Overview struct {
	TotalNodes    int                       `json:"totalNodes"`
	TotalTeams    int                       `json:"totalTeams"`
	TotalProjects int                       `json:"totalProjects"`
	Resources     []service.ResourceType    `json:"resources"`
	NodesByArch   []ArchSummary             `json:"nodesByArch"`
	NodesByStatus map[string]int            `json:"nodesByStatus"`
	CostEnabled   bool                      `json:"costEnabled"`
}

// ArchSummary represents node count by architecture
type ArchSummary struct {
	Arch  string `json:"arch"`
	Count int    `json:"count"`
}

// QuotaAlert represents an alert for quota usage exceeding threshold
type QuotaAlert struct {
	Type         string  `json:"type"`         // "team" or "project"
	Name         string  `json:"name"`
	DisplayName  string  `json:"displayName,omitempty"`
	Resource     string  `json:"resource"`
	Used         string  `json:"used"`
	Limit        string  `json:"limit"`
	UsagePercent float64 `json:"usagePercent"`
}

// CostTrendPoint represents a point in the cost trend chart
type CostTrendPoint struct {
	Date      string  `json:"date"`
	TotalCost float64 `json:"totalCost"`
}

// TopConsumer represents a top resource consumer
type TopConsumer struct {
	Type        string  `json:"type"` // "team" or "project"
	Name        string  `json:"name"`
	DisplayName string  `json:"displayName,omitempty"`
	TotalCost   float64 `json:"totalCost"`
	CPUHours    float64 `json:"cpuHours"`
	MemoryGBH   float64 `json:"memoryGBH"`
	GPUHours    float64 `json:"gpuHours"`
}

// GetOverview returns the dashboard overview
func (h *StatsHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()

	overview := &Overview{
		Resources:     []service.ResourceType{},
		NodesByArch:   []ArchSummary{},
		NodesByStatus: make(map[string]int),
		CostEnabled:   h.costSvc.IsEnabled(),
	}

	// Get configured resources from ResourceService
	resources, err := h.resourceSvc.GetClusterResources(ctx)
	if err != nil {
		logger.Error("Failed to get cluster resources", "error", err)
	} else {
		overview.Resources = resources
	}

	// Get nodes for count and architecture distribution
	nodes, err := h.k8sClient.ListNodes(ctx)
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
	} else {
		overview.TotalNodes = len(nodes.Items)

		// Aggregate architectures
		archMap := make(map[string]int)
		for _, node := range nodes.Items {
			arch := node.Status.NodeInfo.Architecture
			archMap[arch]++
		}
		for arch, count := range archMap {
			overview.NodesByArch = append(overview.NodesByArch, ArchSummary{Arch: arch, Count: count})
		}
	}

	// Get node status distribution
	if h.nodeSvc != nil {
		statusSummary, err := h.nodeSvc.GetNodeStatusSummary(ctx)
		if err == nil {
			for status, count := range statusSummary {
				overview.NodesByStatus[string(status)] = count
			}
		}
	}

	// Get teams
	teams, err := h.tenantSvc.List(ctx)
	if err != nil {
		logger.Error("Failed to list teams", "error", err)
	} else {
		overview.TotalTeams = len(teams)
	}

	// Get projects
	projects, err := h.projectSvc.List(ctx)
	if err != nil {
		logger.Error("Failed to list projects", "error", err)
	} else {
		overview.TotalProjects = len(projects)
	}

	c.JSON(http.StatusOK, overview)
}

// GetTeamUsage returns usage statistics for teams
func (h *StatsHandler) GetTeamUsage(c *gin.Context) {
	window := c.DefaultQuery("window", "7d")

	report, err := h.costSvc.GetTeamUsage(c.Request.Context(), window)
	if err != nil {
		logger.Error("Failed to get team usage", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetProjectUsage returns usage statistics for projects
func (h *StatsHandler) GetProjectUsage(c *gin.Context) {
	window := c.DefaultQuery("window", "7d")

	report, err := h.costSvc.GetProjectUsage(c.Request.Context(), window)
	if err != nil {
		logger.Error("Failed to get project usage", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetUserUsage returns usage statistics for users
func (h *StatsHandler) GetUserUsage(c *gin.Context) {
	window := c.DefaultQuery("window", "7d")

	report, err := h.costSvc.GetUserUsage(c.Request.Context(), window)
	if err != nil {
		logger.Error("Failed to get user usage", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetCostStatus returns whether cost tracking is enabled
func (h *StatsHandler) GetCostStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled": h.costSvc.IsEnabled(),
	})
}

// GetQuotaAlerts returns alerts for quotas exceeding threshold (default 80%)
func (h *StatsHandler) GetQuotaAlerts(c *gin.Context) {
	ctx := c.Request.Context()
	thresholdStr := c.DefaultQuery("threshold", "80")
	threshold, _ := strconv.ParseFloat(thresholdStr, 64)
	if threshold <= 0 {
		threshold = 80
	}

	var alerts []QuotaAlert

	// Check team quotas
	teams, err := h.tenantSvc.List(ctx)
	if err == nil {
		for _, team := range teams {
			for resource, limitStr := range team.Quota {
				usedStr, ok := team.QuotaUsed[resource]
				if !ok {
					continue
				}
				limit, _ := strconv.ParseFloat(limitStr, 64)
				used, _ := strconv.ParseFloat(usedStr, 64)
				if limit > 0 {
					percent := (used / limit) * 100
					if percent >= threshold {
						alerts = append(alerts, QuotaAlert{
							Type:         "team",
							Name:         team.Name,
							DisplayName:  team.DisplayName,
							Resource:     resource,
							Used:         usedStr,
							Limit:        limitStr,
							UsagePercent: percent,
						})
					}
				}
			}
		}
	}

	// Note: Project quotas are no longer supported (projects share team quota)
	// Quota alerts are only generated at team level

	// Sort by usage percent descending
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].UsagePercent > alerts[j].UsagePercent
	})

	c.JSON(http.StatusOK, gin.H{"items": alerts})
}

// GetCostTrend returns cost trend data
func (h *StatsHandler) GetCostTrend(c *gin.Context) {
	window := c.DefaultQuery("window", "7d")

	trend, err := h.costSvc.GetCostTrend(c.Request.Context(), window)
	if err != nil {
		logger.Error("Failed to get cost trend", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": trend})
}

// GetTopConsumers returns top resource consumers
func (h *StatsHandler) GetTopConsumers(c *gin.Context) {
	window := c.DefaultQuery("window", "7d")
	limitStr := c.DefaultQuery("limit", "5")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 5
	}

	var consumers []TopConsumer

	// Get team usage
	teamReport, err := h.costSvc.GetTeamUsage(c.Request.Context(), window)
	if err == nil && teamReport != nil {
		for _, item := range teamReport.Data {
			consumers = append(consumers, TopConsumer{
				Type:        "team",
				Name:        item.Name,
				TotalCost:   item.TotalCost,
				CPUHours:    item.CPUCoreHours,
				MemoryGBH:   item.RAMGBHours,
				GPUHours:    item.GPUHours,
			})
		}
	}

	// Sort by total cost descending
	sort.Slice(consumers, func(i, j int) bool {
		return consumers[i].TotalCost > consumers[j].TotalCost
	})

	// Limit results
	if len(consumers) > limit {
		consumers = consumers[:limit]
	}

	c.JSON(http.StatusOK, gin.H{"items": consumers})
}


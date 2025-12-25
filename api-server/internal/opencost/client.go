package opencost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bison/api-server/pkg/logger"
)

// Client is an OpenCost API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new OpenCost client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsEnabled returns true if OpenCost is configured
func (c *Client) IsEnabled() bool {
	return c.baseURL != ""
}

// Allocation represents a cost allocation from OpenCost
type Allocation struct {
	Name           string             `json:"name"`
	Properties     AllocationProps    `json:"properties"`
	Window         Window             `json:"window"`
	Start          string             `json:"start"`
	End            string             `json:"end"`
	Minutes        float64            `json:"minutes"`
	CPUCores       float64            `json:"cpuCores"`
	CPUCoreHours   float64            `json:"cpuCoreHours"`
	CPUCost        float64            `json:"cpuCost"`
	GPUCount       float64            `json:"gpuCount"`
	GPUHours       float64            `json:"gpuHours"`
	GPUCost        float64            `json:"gpuCost"`
	RAMBytes       float64            `json:"ramBytes"`
	RAMByteHours   float64            `json:"ramByteHours"`
	RAMGBHours     float64            `json:"ramGBHours"`
	RAMCost        float64            `json:"ramCost"`
	PVBytes        float64            `json:"pvBytes"`
	PVByteHours    float64            `json:"pvByteHours"`
	PVCost         float64            `json:"pvCost"`
	NetworkCost    float64            `json:"networkCost"`
	TotalCost      float64            `json:"totalCost"`
	TotalEfficiency float64           `json:"totalEfficiency"`
}

// AllocationProps contains allocation properties
type AllocationProps struct {
	Cluster    string            `json:"cluster"`
	Node       string            `json:"node"`
	Namespace  string            `json:"namespace"`
	Pod        string            `json:"pod"`
	Container  string            `json:"container"`
	Controller string            `json:"controller"`
	Labels     map[string]string `json:"labels"`
}

// Window represents a time window
type Window struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// AllocationResponse is the response from OpenCost allocation API
type AllocationResponse struct {
	Code    int                       `json:"code"`
	Status  string                    `json:"status"`
	Data    []map[string]*Allocation  `json:"data"`
	Message string                    `json:"message"`
}

// GetAllocationByNamespace returns allocations aggregated by namespace
func (c *Client) GetAllocationByNamespace(ctx context.Context, window string) ([]Allocation, error) {
	return c.getAllocation(ctx, window, "namespace", "")
}

// GetAllocationByPod returns allocations aggregated by pod
func (c *Client) GetAllocationByPod(ctx context.Context, window string) ([]Allocation, error) {
	return c.getAllocation(ctx, window, "pod", "")
}

// GetAllocationByLabel returns allocations aggregated by a specific label
func (c *Client) GetAllocationByLabel(ctx context.Context, window, label string) ([]Allocation, error) {
	return c.getAllocation(ctx, window, "label:"+label, "")
}

// GetAllocationByController returns allocations aggregated by controller
func (c *Client) GetAllocationByController(ctx context.Context, window string) ([]Allocation, error) {
	return c.getAllocation(ctx, window, "controller", "")
}

// GetAllocationForNamespace returns allocations for a specific namespace
func (c *Client) GetAllocationForNamespace(ctx context.Context, window, namespace string) ([]Allocation, error) {
	return c.getAllocation(ctx, window, "namespace", fmt.Sprintf("namespace:\"%s\"", namespace))
}

// getAllocation is the internal method to query allocations
func (c *Client) getAllocation(ctx context.Context, window, aggregate, filter string) ([]Allocation, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("opencost not configured")
	}

	// Build URL
	params := url.Values{}
	params.Set("window", window)
	params.Set("aggregate", aggregate)
	params.Set("accumulate", "true")
	if filter != "" {
		params.Set("filter", filter)
	}

	reqURL := fmt.Sprintf("%s/allocation/compute?%s", c.baseURL, params.Encode())
	logger.Debug("OpenCost request", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("OpenCost request failed", "error", err)
		return nil, fmt.Errorf("failed to call opencost: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opencost returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AllocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("opencost error: %s", result.Message)
	}

	// Flatten the response
	var allocations []Allocation
	for _, dataMap := range result.Data {
		for name, alloc := range dataMap {
			if alloc != nil {
				alloc.Name = name
				// Calculate RAMGBHours from RAMByteHours
				if alloc.RAMGBHours == 0 && alloc.RAMByteHours > 0 {
					alloc.RAMGBHours = alloc.RAMByteHours / (1024 * 1024 * 1024)
				}
				allocations = append(allocations, *alloc)
			}
		}
	}

	return allocations, nil
}

// UsageSummary represents a summary of usage for display
type UsageSummary struct {
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

// ToUsageSummary converts an Allocation to a UsageSummary
func (a *Allocation) ToUsageSummary() UsageSummary {
	return UsageSummary{
		Name:         a.Name,
		CPUCoreHours: a.CPUCoreHours,
		RAMGBHours:   a.RAMGBHours,
		GPUHours:     a.GPUHours,
		TotalCost:    a.TotalCost,
		CPUCost:      a.CPUCost,
		RAMCost:      a.RAMCost,
		GPUCost:      a.GPUCost,
		Minutes:      a.Minutes,
	}
}

// GetTeamUsage returns usage summary for teams (by tenant label)
func (c *Client) GetTeamUsage(ctx context.Context, window string) ([]UsageSummary, error) {
	// Get by namespace and then group by tenant
	allocations, err := c.GetAllocationByNamespace(ctx, window)
	if err != nil {
		return nil, err
	}

	// For now, return by namespace (which corresponds to projects)
	// Team-level aggregation would need to be done in the service layer
	var summaries []UsageSummary
	for _, a := range allocations {
		summaries = append(summaries, a.ToUsageSummary())
	}
	return summaries, nil
}

// GetProjectUsage returns usage summary for projects (namespaces)
func (c *Client) GetProjectUsage(ctx context.Context, window string) ([]UsageSummary, error) {
	allocations, err := c.GetAllocationByNamespace(ctx, window)
	if err != nil {
		return nil, err
	}

	var summaries []UsageSummary
	for _, a := range allocations {
		summaries = append(summaries, a.ToUsageSummary())
	}
	return summaries, nil
}

// GetUserUsage returns usage summary for users (by owner label)
func (c *Client) GetUserUsage(ctx context.Context, window string) ([]UsageSummary, error) {
	// Try to get by user label if available
	allocations, err := c.GetAllocationByLabel(ctx, window, "bison.io/user")
	if err != nil {
		// Fallback to pod-level
		allocations, err = c.GetAllocationByPod(ctx, window)
		if err != nil {
			return nil, err
		}
	}

	var summaries []UsageSummary
	for _, a := range allocations {
		summaries = append(summaries, a.ToUsageSummary())
	}
	return summaries, nil
}

// GetTotalCost returns the total cost for a window
func (c *Client) GetTotalCost(ctx context.Context, window string) (float64, error) {
	allocations, err := c.GetAllocationByNamespace(ctx, window)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, a := range allocations {
		total += a.TotalCost
	}
	return total, nil
}

// CostTrendPoint represents a daily cost point
type CostTrendPoint struct {
	Date      string  `json:"date"`
	TotalCost float64 `json:"totalCost"`
}

// GetCostTrend returns daily cost data for a window
func (c *Client) GetCostTrend(ctx context.Context, window string) ([]CostTrendPoint, error) {
	if !c.IsEnabled() {
		return []CostTrendPoint{}, nil
	}

	// Parse window to determine number of days
	days := 7
	switch window {
	case "1d", "today":
		days = 1
	case "2d", "yesterday":
		days = 2
	case "7d", "week":
		days = 7
	case "30d", "month":
		days = 30
	}

	// Query daily allocation data
	params := url.Values{}
	params.Set("window", window)
	params.Set("aggregate", "namespace")
	params.Set("accumulate", "false") // Don't accumulate to get daily data
	params.Set("step", "1d")          // Daily step

	reqURL := fmt.Sprintf("%s/allocation/compute?%s", c.baseURL, params.Encode())
	logger.Debug("OpenCost cost trend request", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("OpenCost request failed", "error", err)
		return nil, fmt.Errorf("failed to call opencost: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opencost returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AllocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("opencost error: %s", result.Message)
	}

	// Aggregate daily costs
	var trend []CostTrendPoint
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		
		// Sum cost for this day from all allocations
		var dayCost float64
		if i < len(result.Data) {
			for _, alloc := range result.Data[len(result.Data)-1-i] {
				if alloc != nil {
					dayCost += alloc.TotalCost
				}
			}
		}
		
		trend = append(trend, CostTrendPoint{
			Date:      dateStr,
			TotalCost: dayCost,
		})
	}

	return trend, nil
}


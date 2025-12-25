package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/pkg/logger"
)

// TaskExecution represents a task execution record (moved from scheduler to avoid import cycle)
type TaskExecution struct {
	TaskName  string    `json:"taskName"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Status    string    `json:"status"` // "success", "failed"
	Error     string    `json:"error,omitempty"`
}

// TaskExecutionGetter interface for getting task executions (to avoid import cycle)
type TaskExecutionGetter interface {
	GetExecutions(limit int) []TaskExecution
}

// ServiceStatus represents the status of an external service
type ServiceStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Message   string `json:"message,omitempty"`
	URL       string `json:"url,omitempty"`
}

// SystemStatus represents overall system status
type SystemStatus struct {
	OpenCost   ServiceStatus     `json:"opencost"`
	Capsule    ServiceStatus     `json:"capsule"`
	Prometheus ServiceStatus     `json:"prometheus"`
	Tasks      []TaskExecution   `json:"tasks"`
	Statistics SystemStatistics  `json:"statistics"`
}

// SystemStatistics represents system-wide statistics
type SystemStatistics struct {
	TotalTeams     int     `json:"totalTeams"`
	TotalProjects  int     `json:"totalProjects"`
	TotalUsers     int     `json:"totalUsers"`
	TotalNodes     int     `json:"totalNodes"`
	TotalBalance   float64 `json:"totalBalance"`
	SuspendedTeams int     `json:"suspendedTeams"`
}

// StatusService handles system status
type StatusService struct {
	k8sClient      *k8s.Client
	opencostClient *opencost.Client
	taskGetter     TaskExecutionGetter
	tenantSvc      *TenantService
	projectSvc     *ProjectService
	userSvc        *UserService
	balanceSvc     *BalanceService
	prometheusURL  string
	httpClient     *http.Client
}

// NewStatusService creates a new StatusService
func NewStatusService(
	k8sClient *k8s.Client,
	opencostClient *opencost.Client,
	taskGetter TaskExecutionGetter,
	tenantSvc *TenantService,
	projectSvc *ProjectService,
	userSvc *UserService,
	balanceSvc *BalanceService,
	prometheusURL string,
) *StatusService {
	return &StatusService{
		k8sClient:      k8sClient,
		opencostClient: opencostClient,
		taskGetter:     taskGetter,
		tenantSvc:      tenantSvc,
		projectSvc:     projectSvc,
		userSvc:        userSvc,
		balanceSvc:     balanceSvc,
		prometheusURL:  prometheusURL,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

// GetStatus returns overall system status
func (s *StatusService) GetStatus(ctx context.Context) (*SystemStatus, error) {
	logger.Debug("Getting system status")

	status := &SystemStatus{
		OpenCost:   s.checkOpenCost(ctx),
		Capsule:    s.checkCapsule(ctx),
		Prometheus: s.checkPrometheus(ctx),
		Tasks:      []TaskExecution{},
		Statistics: SystemStatistics{},
	}

	// Get task executions
	if s.taskGetter != nil {
		status.Tasks = s.taskGetter.GetExecutions(20)
	}

	// Get statistics
	status.Statistics = s.getStatistics(ctx)

	return status, nil
}

// GetTaskHistory returns recent task executions
func (s *StatusService) GetTaskHistory(ctx context.Context, limit int) ([]TaskExecution, error) {
	if s.taskGetter == nil {
		return []TaskExecution{}, nil
	}
	return s.taskGetter.GetExecutions(limit), nil
}

func (s *StatusService) checkOpenCost(ctx context.Context) ServiceStatus {
	status := ServiceStatus{
		Name: "OpenCost",
	}

	if s.opencostClient == nil || !s.opencostClient.IsEnabled() {
		status.Available = false
		status.Message = "Not configured"
		return status
	}

	// Try to make a request
	_, err := s.opencostClient.GetTotalCost(ctx, "1h")
	if err != nil {
		status.Available = false
		status.Message = fmt.Sprintf("Error: %v", err)
	} else {
		status.Available = true
		status.Message = "Connected"
	}

	return status
}

func (s *StatusService) checkCapsule(ctx context.Context) ServiceStatus {
	status := ServiceStatus{
		Name: "Capsule",
	}

	// Try to list tenants
	_, err := s.k8sClient.ListTenants(ctx)
	if err != nil {
		status.Available = false
		status.Message = fmt.Sprintf("Error: %v", err)
	} else {
		status.Available = true
		status.Message = "Connected"
	}

	return status
}

func (s *StatusService) checkPrometheus(ctx context.Context) ServiceStatus {
	status := ServiceStatus{
		Name: "Prometheus",
		URL:  s.prometheusURL,
	}

	if s.prometheusURL == "" {
		status.Available = false
		status.Message = "Not configured"
		return status
	}

	// Try to access Prometheus
	req, _ := http.NewRequestWithContext(ctx, "GET", s.prometheusURL+"/-/healthy", nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		status.Available = false
		status.Message = fmt.Sprintf("Error: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		status.Available = true
		status.Message = "Connected"
	} else {
		status.Available = false
		status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return status
}

func (s *StatusService) getStatistics(ctx context.Context) SystemStatistics {
	stats := SystemStatistics{}

	// Get team count
	if s.tenantSvc != nil {
		teams, _ := s.tenantSvc.List(ctx)
		stats.TotalTeams = len(teams)
		
		// Count suspended teams
		for _, team := range teams {
			if team.Suspended {
				stats.SuspendedTeams++
			}
		}
	}

	// Get project count
	if s.projectSvc != nil {
		projects, _ := s.projectSvc.List(ctx)
		stats.TotalProjects = len(projects)
	}

	// Get user count
	if s.userSvc != nil {
		users, _ := s.userSvc.List(ctx)
		stats.TotalUsers = len(users)
	}

	// Get node count
	nodes, _ := s.k8sClient.ListNodes(ctx)
	stats.TotalNodes = len(nodes.Items)

	// Get total balance
	if s.balanceSvc != nil {
		total, _ := s.balanceSvc.GetTotalBalance(ctx)
		stats.TotalBalance = total
	}

	return stats
}

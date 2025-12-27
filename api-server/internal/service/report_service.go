package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/pkg/logger"
)

// Report represents a cost report
type Report struct {
	Type           string             `json:"type"` // team, project, summary
	Name           string             `json:"name"` // Entity name
	Window         string             `json:"window"`
	GeneratedAt    time.Time          `json:"generatedAt"`
	TotalCost      float64            `json:"totalCost"`
	CostByDay      []DailyCost        `json:"costByDay,omitempty"`
	CostByResource map[string]float64 `json:"costByResource"`
	UsageSummary   *UsageData         `json:"usageSummary"`
}

// DailyCost represents cost for a single day
type DailyCost struct {
	Date    string  `json:"date"`
	Cost    float64 `json:"cost"`
	CPUCost float64 `json:"cpuCost"`
	RAMCost float64 `json:"ramCost"`
	GPUCost float64 `json:"gpuCost"`
}

// SummaryReport represents an overall summary report
type SummaryReport struct {
	Window        string            `json:"window"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	TotalCost     float64           `json:"totalCost"`
	TotalTeams    int               `json:"totalTeams"`
	TotalProjects int               `json:"totalProjects"`
	TopTeams      []TeamCostRank    `json:"topTeams"`
	TopProjects   []ProjectCostRank `json:"topProjects"`
	CostTrend     []DailyCost       `json:"costTrend"`
}

// TeamCostRank represents a team in cost ranking
type TeamCostRank struct {
	Rank       int     `json:"rank"`
	TeamName   string  `json:"teamName"`
	Cost       float64 `json:"cost"`
	Percentage float64 `json:"percentage"`
}

// ProjectCostRank represents a project in cost ranking
type ProjectCostRank struct {
	Rank        int     `json:"rank"`
	ProjectName string  `json:"projectName"`
	TeamName    string  `json:"teamName"`
	Cost        float64 `json:"cost"`
	Percentage  float64 `json:"percentage"`
}

// ReportService handles report generation
type ReportService struct {
	opencostClient *opencost.Client
	tenantSvc      *TenantService
	projectSvc     *ProjectService
	billingSvc     *BillingService
}

// NewReportService creates a new ReportService
func NewReportService(
	opencostClient *opencost.Client,
	tenantSvc *TenantService,
	projectSvc *ProjectService,
	billingSvc *BillingService,
) *ReportService {
	return &ReportService{
		opencostClient: opencostClient,
		tenantSvc:      tenantSvc,
		projectSvc:     projectSvc,
		billingSvc:     billingSvc,
	}
}

// GenerateTeamReport generates a report for a specific team
func (s *ReportService) GenerateTeamReport(ctx context.Context, teamName, window string) (*Report, error) {
	logger.Debug("Generating team report", "team", teamName, "window", window)

	if window == "" {
		window = "30d"
	}

	bill, err := s.billingSvc.GetTeamBill(ctx, teamName, window)
	if err != nil {
		return nil, err
	}

	report := &Report{
		Type:           "team",
		Name:           teamName,
		Window:         window,
		GeneratedAt:    time.Now(),
		TotalCost:      bill.TotalCost,
		CostByResource: bill.ResourceCosts,
		UsageSummary:   bill.UsageDetails,
	}

	return report, nil
}

// GenerateProjectReport generates a report for a specific project
func (s *ReportService) GenerateProjectReport(ctx context.Context, projectName, window string) (*Report, error) {
	logger.Debug("Generating project report", "project", projectName, "window", window)

	if window == "" {
		window = "30d"
	}

	bill, err := s.billingSvc.GetProjectBill(ctx, projectName, window)
	if err != nil {
		return nil, err
	}

	report := &Report{
		Type:           "project",
		Name:           projectName,
		Window:         window,
		GeneratedAt:    time.Now(),
		TotalCost:      bill.TotalCost,
		CostByResource: bill.ResourceCosts,
		UsageSummary:   bill.UsageDetails,
	}

	return report, nil
}

// GenerateSummaryReport generates an overall summary report
func (s *ReportService) GenerateSummaryReport(ctx context.Context, window string) (*SummaryReport, error) {
	logger.Debug("Generating summary report", "window", window)

	if window == "" {
		window = "30d"
	}

	teams, err := s.tenantSvc.List(ctx)
	if err != nil {
		return nil, err
	}

	projects, err := s.projectSvc.List(ctx)
	if err != nil {
		return nil, err
	}

	report := &SummaryReport{
		Window:        window,
		GeneratedAt:   time.Now(),
		TotalTeams:    len(teams),
		TotalProjects: len(projects),
		TopTeams:      []TeamCostRank{},
		TopProjects:   []ProjectCostRank{},
	}

	// Calculate costs
	var totalCost float64
	teamCosts := make(map[string]float64)

	for _, team := range teams {
		bill, _ := s.billingSvc.GetTeamBill(ctx, team.Name, window)
		if bill != nil {
			teamCosts[team.Name] = bill.TotalCost
			totalCost += bill.TotalCost
		}
	}

	report.TotalCost = totalCost

	// Top teams
	rank := 1
	for name, cost := range teamCosts {
		percentage := 0.0
		if totalCost > 0 {
			percentage = (cost / totalCost) * 100
		}
		report.TopTeams = append(report.TopTeams, TeamCostRank{
			Rank:       rank,
			TeamName:   name,
			Cost:       cost,
			Percentage: percentage,
		})
		rank++
	}

	// Sort by cost descending and limit to top 10
	sortTeamCostRank(report.TopTeams)
	if len(report.TopTeams) > 10 {
		report.TopTeams = report.TopTeams[:10]
	}
	// Re-assign ranks
	for i := range report.TopTeams {
		report.TopTeams[i].Rank = i + 1
	}

	return report, nil
}

// ExportCSV exports a report as CSV
func (s *ReportService) ExportCSV(ctx context.Context, reportType, name, window string) ([]byte, error) {
	logger.Debug("Exporting CSV", "type", reportType, "name", name, "window", window)

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	switch reportType {
	case "team":
		report, err := s.GenerateTeamReport(ctx, name, window)
		if err != nil {
			return nil, err
		}
		return s.teamReportToCSV(writer, report)

	case "project":
		report, err := s.GenerateProjectReport(ctx, name, window)
		if err != nil {
			return nil, err
		}
		return s.projectReportToCSV(writer, report)

	case "summary":
		report, err := s.GenerateSummaryReport(ctx, window)
		if err != nil {
			return nil, err
		}
		return s.summaryReportToCSV(writer, report)

	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}
}

func (s *ReportService) teamReportToCSV(writer *csv.Writer, report *Report) ([]byte, error) {
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	// Header
	csvWriter.Write([]string{"Team Report", report.Name})
	csvWriter.Write([]string{"Window", report.Window})
	csvWriter.Write([]string{"Generated At", report.GeneratedAt.Format(time.RFC3339)})
	csvWriter.Write([]string{})

	// Usage summary
	csvWriter.Write([]string{"Resource", "Usage", "Cost"})
	if report.UsageSummary != nil {
		csvWriter.Write([]string{"CPU", fmt.Sprintf("%.2f core-hours", report.UsageSummary.CPUCoreHours), fmt.Sprintf("%.2f", report.UsageSummary.CPUCost)})
		csvWriter.Write([]string{"Memory", fmt.Sprintf("%.2f GB-hours", report.UsageSummary.RAMGBHours), fmt.Sprintf("%.2f", report.UsageSummary.RAMCost)})
		csvWriter.Write([]string{"GPU", fmt.Sprintf("%.2f hours", report.UsageSummary.GPUHours), fmt.Sprintf("%.2f", report.UsageSummary.GPUCost)})
	}
	csvWriter.Write([]string{})
	csvWriter.Write([]string{"Total Cost", fmt.Sprintf("%.2f", report.TotalCost)})

	csvWriter.Flush()
	return buf.Bytes(), csvWriter.Error()
}

func (s *ReportService) projectReportToCSV(writer *csv.Writer, report *Report) ([]byte, error) {
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	// Header
	csvWriter.Write([]string{"Project Report", report.Name})
	csvWriter.Write([]string{"Window", report.Window})
	csvWriter.Write([]string{"Generated At", report.GeneratedAt.Format(time.RFC3339)})
	csvWriter.Write([]string{})

	// Usage summary
	csvWriter.Write([]string{"Resource", "Usage", "Cost"})
	if report.UsageSummary != nil {
		csvWriter.Write([]string{"CPU", fmt.Sprintf("%.2f core-hours", report.UsageSummary.CPUCoreHours), fmt.Sprintf("%.2f", report.UsageSummary.CPUCost)})
		csvWriter.Write([]string{"Memory", fmt.Sprintf("%.2f GB-hours", report.UsageSummary.RAMGBHours), fmt.Sprintf("%.2f", report.UsageSummary.RAMCost)})
		csvWriter.Write([]string{"GPU", fmt.Sprintf("%.2f hours", report.UsageSummary.GPUHours), fmt.Sprintf("%.2f", report.UsageSummary.GPUCost)})
	}
	csvWriter.Write([]string{})
	csvWriter.Write([]string{"Total Cost", fmt.Sprintf("%.2f", report.TotalCost)})

	csvWriter.Flush()
	return buf.Bytes(), csvWriter.Error()
}

func (s *ReportService) summaryReportToCSV(writer *csv.Writer, report *SummaryReport) ([]byte, error) {
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	// Header
	csvWriter.Write([]string{"Summary Report"})
	csvWriter.Write([]string{"Window", report.Window})
	csvWriter.Write([]string{"Generated At", report.GeneratedAt.Format(time.RFC3339)})
	csvWriter.Write([]string{})

	// Overview
	csvWriter.Write([]string{"Total Teams", fmt.Sprintf("%d", report.TotalTeams)})
	csvWriter.Write([]string{"Total Projects", fmt.Sprintf("%d", report.TotalProjects)})
	csvWriter.Write([]string{"Total Cost", fmt.Sprintf("%.2f", report.TotalCost)})
	csvWriter.Write([]string{})

	// Top teams
	csvWriter.Write([]string{"Top Teams"})
	csvWriter.Write([]string{"Rank", "Team", "Cost", "Percentage"})
	for _, team := range report.TopTeams {
		csvWriter.Write([]string{
			fmt.Sprintf("%d", team.Rank),
			team.TeamName,
			fmt.Sprintf("%.2f", team.Cost),
			fmt.Sprintf("%.1f%%", team.Percentage),
		})
	}

	csvWriter.Flush()
	return buf.Bytes(), csvWriter.Error()
}

func sortTeamCostRank(ranks []TeamCostRank) {
	for i := 0; i < len(ranks); i++ {
		for j := i + 1; j < len(ranks); j++ {
			if ranks[i].Cost < ranks[j].Cost {
				ranks[i], ranks[j] = ranks[j], ranks[i]
			}
		}
	}
}

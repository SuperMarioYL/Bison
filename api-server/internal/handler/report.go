package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// ReportHandler handles report-related requests
type ReportHandler struct {
	reportSvc *service.ReportService
}

// NewReportHandler creates a new ReportHandler
func NewReportHandler(reportSvc *service.ReportService) *ReportHandler {
	return &ReportHandler{
		reportSvc: reportSvc,
	}
}

// GetTeamReport returns a report for a team
func (h *ReportHandler) GetTeamReport(c *gin.Context) {
	teamName := c.Param("name")
	window := c.DefaultQuery("window", "30d")

	report, err := h.reportSvc.GenerateTeamReport(c.Request.Context(), teamName, window)
	if err != nil {
		logger.Error("Failed to generate team report", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportTeamReport exports a team report as CSV
func (h *ReportHandler) ExportTeamReport(c *gin.Context) {
	teamName := c.Param("name")
	window := c.DefaultQuery("window", "30d")
	format := c.DefaultQuery("format", "csv")

	if format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only csv format is supported"})
		return
	}

	data, err := h.reportSvc.ExportCSV(c.Request.Context(), "team", teamName, window)
	if err != nil {
		logger.Error("Failed to export team report", "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-report.csv", teamName))
	c.Data(http.StatusOK, "text/csv", data)
}

// GetProjectReport returns a report for a project
func (h *ReportHandler) GetProjectReport(c *gin.Context) {
	projectName := c.Param("name")
	window := c.DefaultQuery("window", "30d")

	report, err := h.reportSvc.GenerateProjectReport(c.Request.Context(), projectName, window)
	if err != nil {
		logger.Error("Failed to generate project report", "project", projectName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportProjectReport exports a project report as CSV
func (h *ReportHandler) ExportProjectReport(c *gin.Context) {
	projectName := c.Param("name")
	window := c.DefaultQuery("window", "30d")
	format := c.DefaultQuery("format", "csv")

	if format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only csv format is supported"})
		return
	}

	data, err := h.reportSvc.ExportCSV(c.Request.Context(), "project", projectName, window)
	if err != nil {
		logger.Error("Failed to export project report", "project", projectName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-report.csv", projectName))
	c.Data(http.StatusOK, "text/csv", data)
}

// GetSummaryReport returns an overall summary report
func (h *ReportHandler) GetSummaryReport(c *gin.Context) {
	window := c.DefaultQuery("window", "30d")

	report, err := h.reportSvc.GenerateSummaryReport(c.Request.Context(), window)
	if err != nil {
		logger.Error("Failed to generate summary report", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportSummaryReport exports a summary report as CSV
func (h *ReportHandler) ExportSummaryReport(c *gin.Context) {
	window := c.DefaultQuery("window", "30d")
	format := c.DefaultQuery("format", "csv")

	if format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only csv format is supported"})
		return
	}

	data, err := h.reportSvc.ExportCSV(c.Request.Context(), "summary", "", window)
	if err != nil {
		logger.Error("Failed to export summary report", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=summary-report.csv")
	c.Data(http.StatusOK, "text/csv", data)
}

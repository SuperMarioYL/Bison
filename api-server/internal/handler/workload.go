package handler

import (
	"net/http"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
	"github.com/gin-gonic/gin"
)

// WorkloadHandler handles workload-related requests
type WorkloadHandler struct {
	workloadSvc *service.WorkloadService
	projectSvc  *service.ProjectService
}

// NewWorkloadHandler creates a new WorkloadHandler
func NewWorkloadHandler(workloadSvc *service.WorkloadService, projectSvc *service.ProjectService) *WorkloadHandler {
	return &WorkloadHandler{
		workloadSvc: workloadSvc,
		projectSvc:  projectSvc,
	}
}

// GetWorkloadSummary returns the workload summary for a project
func (h *WorkloadHandler) GetWorkloadSummary(c *gin.Context) {
	projectName := c.Param("name")

	// Verify project exists
	project, err := h.projectSvc.Get(c.Request.Context(), projectName)
	if err != nil {
		logger.Error("Failed to get project", "project", projectName, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	summary, err := h.workloadSvc.GetWorkloadSummary(c.Request.Context(), project.Name)
	if err != nil {
		logger.Error("Failed to get workload summary", "project", projectName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// ListWorkloads returns all workloads for a project
func (h *WorkloadHandler) ListWorkloads(c *gin.Context) {
	projectName := c.Param("name")

	// Verify project exists
	project, err := h.projectSvc.Get(c.Request.Context(), projectName)
	if err != nil {
		logger.Error("Failed to get project", "project", projectName, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	workloads, err := h.workloadSvc.ListWorkloads(c.Request.Context(), project.Name)
	if err != nil {
		logger.Error("Failed to list workloads", "project", projectName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": workloads})
}


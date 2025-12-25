package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// ProjectHandler handles project-related API requests
type ProjectHandler struct {
	projectSvc        *service.ProjectService
	costSvc           *service.CostService
	resourceConfigSvc *service.ResourceConfigService
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler(projectSvc *service.ProjectService, costSvc *service.CostService, resourceConfigSvc *service.ResourceConfigService) *ProjectHandler {
	return &ProjectHandler{
		projectSvc:        projectSvc,
		costSvc:           costSvc,
		resourceConfigSvc: resourceConfigSvc,
	}
}

// ListProjects returns all projects
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	teamName := c.Query("team")

	var projects []*service.Project
	var err error

	if teamName != "" {
		projects, err = h.projectSvc.ListByTeam(c.Request.Context(), teamName)
	} else {
		projects, err = h.projectSvc.List(c.Request.Context())
	}

	if err != nil {
		logger.Error("Failed to list projects", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
}

// GetProject returns a specific project
func (h *ProjectHandler) GetProject(c *gin.Context) {
	name := c.Param("name")

	project, err := h.projectSvc.Get(c.Request.Context(), name)
	if err != nil {
		logger.Error("Failed to get project", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Get usage data
	window := c.DefaultQuery("window", "7d")
	usage, _ := h.costSvc.GetProjectUsageByName(c.Request.Context(), name, window)

	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"usage":   usage,
	})
}

// CreateProject creates a new project
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req struct {
		Name        string                  `json:"name" binding:"required"`
		Team        string                  `json:"team" binding:"required"`
		DisplayName string                  `json:"displayName"`
		Description string                  `json:"description"`
		Members     []service.ProjectMember `json:"members"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request for CreateProject", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := &service.Project{
		Name:        req.Name,
		Team:        req.Team,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Members:     req.Members,
	}

	if project.DisplayName == "" {
		project.DisplayName = project.Name
	}

	if err := h.projectSvc.Create(c.Request.Context(), project); err != nil {
		logger.Error("Failed to create project", "name", req.Name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// UpdateProject updates an existing project
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		DisplayName string                  `json:"displayName"`
		Description string                  `json:"description"`
		Members     []service.ProjectMember `json:"members"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request for UpdateProject", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := &service.Project{
		Name:        name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Members:     req.Members,
	}

	if err := h.projectSvc.Update(c.Request.Context(), name, project); err != nil {
		logger.Error("Failed to update project", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	name := c.Param("name")

	if err := h.projectSvc.Delete(c.Request.Context(), name); err != nil {
		logger.Error("Failed to delete project", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// GetProjectUsage returns resource usage for a project (dynamically based on resource config)
func (h *ProjectHandler) GetProjectUsage(c *gin.Context) {
	name := c.Param("name")

	// Get enabled resource configs
	resourceConfigs, err := h.resourceConfigSvc.GetEnabledResourceConfigs(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get resource configs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get project usage
	usage, err := h.projectSvc.GetProjectUsage(c.Request.Context(), name, resourceConfigs)
	if err != nil {
		logger.Error("Failed to get project usage", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}

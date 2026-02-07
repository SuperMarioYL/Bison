package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// OnboardingHandler handles node onboarding requests
type OnboardingHandler struct {
	onboardingSvc   *service.OnboardingService
	initScriptSvc   *service.InitScriptService
}

// NewOnboardingHandler creates a new OnboardingHandler
func NewOnboardingHandler(onboardingSvc *service.OnboardingService, initScriptSvc *service.InitScriptService) *OnboardingHandler {
	return &OnboardingHandler{
		onboardingSvc:   onboardingSvc,
		initScriptSvc:   initScriptSvc,
	}
}

// StartOnboarding starts a new node onboarding job
// POST /api/v1/nodes/onboard
func (h *OnboardingHandler) StartOnboarding(c *gin.Context) {
	var req service.OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.onboardingSvc.StartOnboarding(c.Request.Context(), &req)
	if err != nil {
		logger.Error("Failed to start onboarding", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, job)
}

// GetOnboardingJob returns a specific onboarding job
// GET /api/v1/nodes/onboard/:jobId
func (h *OnboardingHandler) GetOnboardingJob(c *gin.Context) {
	jobID := c.Param("jobId")

	job, err := h.onboardingSvc.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

// ListOnboardingJobs returns all onboarding jobs
// GET /api/v1/nodes/onboard
func (h *OnboardingHandler) ListOnboardingJobs(c *gin.Context) {
	jobs, err := h.onboardingSvc.ListJobs(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list onboarding jobs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": jobs})
}

// CancelOnboardingJob cancels a running onboarding job
// DELETE /api/v1/nodes/onboard/:jobId
func (h *OnboardingHandler) CancelOnboardingJob(c *gin.Context) {
	jobID := c.Param("jobId")

	err := h.onboardingSvc.CancelJob(c.Request.Context(), jobID)
	if err != nil {
		logger.Error("Failed to cancel onboarding job", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled"})
}

// GetControlPlaneConfig returns the control plane configuration
// GET /api/v1/settings/control-plane
func (h *OnboardingHandler) GetControlPlaneConfig(c *gin.Context) {
	config, err := h.initScriptSvc.GetControlPlaneConfig(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get control plane config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mask sensitive data
	response := gin.H{
		"host":       config.Host,
		"sshPort":    config.SSHPort,
		"sshUser":    config.SSHUser,
		"authMethod": config.AuthMethod,
		"hasPassword":   config.Password != "",
		"hasPrivateKey": config.PrivateKey != "",
	}

	c.JSON(http.StatusOK, response)
}

// UpdateControlPlaneConfig updates the control plane configuration
// PUT /api/v1/settings/control-plane
func (h *OnboardingHandler) UpdateControlPlaneConfig(c *gin.Context) {
	var config service.ControlPlaneConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing config to preserve credentials if not provided
	existing, _ := h.initScriptSvc.GetControlPlaneConfig(c.Request.Context())
	if existing != nil {
		if config.Password == "" && existing.Password != "" {
			config.Password = existing.Password
		}
		if config.PrivateKey == "" && existing.PrivateKey != "" {
			config.PrivateKey = existing.PrivateKey
		}
	}

	err := h.initScriptSvc.SaveControlPlaneConfig(c.Request.Context(), &config)
	if err != nil {
		logger.Error("Failed to save control plane config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Control plane configuration saved"})
}

// TestControlPlaneConnection tests the control plane SSH connection
// POST /api/v1/settings/control-plane/test
func (h *OnboardingHandler) TestControlPlaneConnection(c *gin.Context) {
	err := h.onboardingSvc.TestControlPlaneConnection(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Connection successful"})
}

// ListInitScripts returns all init script groups
// GET /api/v1/settings/init-scripts
func (h *OnboardingHandler) ListInitScripts(c *gin.Context) {
	groups, err := h.initScriptSvc.GetAllScriptGroups(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list init scripts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": groups})
}

// GetInitScript returns a specific init script group
// GET /api/v1/settings/init-scripts/:id
func (h *OnboardingHandler) GetInitScript(c *gin.Context) {
	id := c.Param("id")

	group, err := h.initScriptSvc.GetScriptGroup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// CreateInitScript creates a new init script group
// POST /api/v1/settings/init-scripts
func (h *OnboardingHandler) CreateInitScript(c *gin.Context) {
	var group service.ScriptGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.initScriptSvc.CreateScriptGroup(c.Request.Context(), &group)
	if err != nil {
		logger.Error("Failed to create init script", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateInitScript updates an init script group
// PUT /api/v1/settings/init-scripts/:id
func (h *OnboardingHandler) UpdateInitScript(c *gin.Context) {
	id := c.Param("id")

	var group service.ScriptGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.initScriptSvc.UpdateScriptGroup(c.Request.Context(), id, &group)
	if err != nil {
		logger.Error("Failed to update init script", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteInitScript deletes an init script group
// DELETE /api/v1/settings/init-scripts/:id
func (h *OnboardingHandler) DeleteInitScript(c *gin.Context) {
	id := c.Param("id")

	err := h.initScriptSvc.DeleteScriptGroup(c.Request.Context(), id)
	if err != nil {
		logger.Error("Failed to delete init script", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Script group deleted"})
}

// ToggleInitScript enables or disables an init script group
// PUT /api/v1/settings/init-scripts/:id/toggle
func (h *OnboardingHandler) ToggleInitScript(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.initScriptSvc.ToggleScriptGroup(c.Request.Context(), id, req.Enabled)
	if err != nil {
		logger.Error("Failed to toggle init script", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Script group toggled"})
}

// ReorderInitScripts updates the order of init script groups
// PUT /api/v1/settings/init-scripts/reorder
func (h *OnboardingHandler) ReorderInitScripts(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.initScriptSvc.ReorderScriptGroups(c.Request.Context(), req.IDs)
	if err != nil {
		logger.Error("Failed to reorder init scripts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Script groups reordered"})
}


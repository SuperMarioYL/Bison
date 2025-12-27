package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// ResourceConfigHandler handles resource configuration requests
type ResourceConfigHandler struct {
	resourceConfigSvc *service.ResourceConfigService
}

// NewResourceConfigHandler creates a new ResourceConfigHandler
func NewResourceConfigHandler(resourceConfigSvc *service.ResourceConfigService) *ResourceConfigHandler {
	return &ResourceConfigHandler{
		resourceConfigSvc: resourceConfigSvc,
	}
}

// ListResourceConfigs returns all resource configurations
// GET /api/v1/resource-configs
func (h *ResourceConfigHandler) ListResourceConfigs(c *gin.Context) {
	configs, err := h.resourceConfigSvc.GetResourceConfigs(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get resource configs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": configs})
}

// GetEnabledResourceConfigs returns only enabled resource configurations
// GET /api/v1/resource-configs/enabled
func (h *ResourceConfigHandler) GetEnabledResourceConfigs(c *gin.Context) {
	configs, err := h.resourceConfigSvc.GetEnabledResourceConfigs(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get enabled resource configs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": configs})
}

// GetQuotaResourceConfigs returns resources for quota settings
// GET /api/v1/resource-configs/quota
func (h *ResourceConfigHandler) GetQuotaResourceConfigs(c *gin.Context) {
	configs, err := h.resourceConfigSvc.GetQuotaResourceConfigs(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get quota resource configs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": configs})
}

// DiscoverClusterResources discovers all resources in the cluster
// GET /api/v1/resource-configs/discover
func (h *ResourceConfigHandler) DiscoverClusterResources(c *gin.Context) {
	resources, err := h.resourceConfigSvc.DiscoverClusterResources(c.Request.Context())
	if err != nil {
		logger.Error("Failed to discover cluster resources", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": resources})
}

// GetResourceConfig returns a single resource configuration
// GET /api/v1/resource-configs/:name
func (h *ResourceConfigHandler) GetResourceConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource name is required"})
		return
	}

	config, err := h.resourceConfigSvc.GetResourceConfig(c.Request.Context(), name)
	if err != nil {
		logger.Error("Failed to get resource config", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// SaveResourceConfigs saves all resource configurations
// PUT /api/v1/resource-configs
func (h *ResourceConfigHandler) SaveResourceConfigs(c *gin.Context) {
	var req struct {
		Items []service.ResourceDefinition `json:"items"`
	}

	// Read raw body for debugging
	bodyBytes, _ := c.GetRawData()
	logger.Debug("SaveResourceConfigs request body", "body", string(bodyBytes))

	// Re-bind since we consumed the body
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to parse request body", "error", err, "body", string(bodyBytes))
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	logger.Info("Saving resource configs", "count", len(req.Items))

	if err := h.resourceConfigSvc.SaveResourceConfigs(c.Request.Context(), req.Items); err != nil {
		logger.Error("Failed to save resource configs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resource configs saved successfully"})
}

// UpdateResourceConfig updates a single resource configuration
// PUT /api/v1/resource-configs/:name
func (h *ResourceConfigHandler) UpdateResourceConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource name is required"})
		return
	}

	var config service.ResourceDefinition
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure name matches
	config.Name = name

	if err := h.resourceConfigSvc.UpdateResourceConfig(c.Request.Context(), name, config); err != nil {
		logger.Error("Failed to update resource config", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Resource config updated successfully"})
}

// AddResourceConfig adds a new resource configuration
// POST /api/v1/resource-configs
func (h *ResourceConfigHandler) AddResourceConfig(c *gin.Context) {
	var config service.ResourceDefinition
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if config.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource name is required"})
		return
	}

	// Check if already exists
	existing, _ := h.resourceConfigSvc.GetResourceConfig(c.Request.Context(), config.Name)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "resource config already exists"})
		return
	}

	if err := h.resourceConfigSvc.UpdateResourceConfig(c.Request.Context(), config.Name, config); err != nil {
		logger.Error("Failed to add resource config", "name", config.Name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Resource config added successfully"})
}

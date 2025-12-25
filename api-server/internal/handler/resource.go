package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// ResourceHandler handles cluster resource requests
type ResourceHandler struct {
	resourceSvc *service.ResourceService
}

// NewResourceHandler creates a new ResourceHandler
func NewResourceHandler(resourceSvc *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{
		resourceSvc: resourceSvc,
	}
}

// GetClusterResources returns all available resource types in the cluster
func (h *ResourceHandler) GetClusterResources(c *gin.Context) {
	resources, err := h.resourceSvc.GetClusterResources(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get cluster resources", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": resources})
}


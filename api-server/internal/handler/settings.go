package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// SettingsHandler handles settings requests
type SettingsHandler struct {
	settingsSvc *service.SettingsService
}

// NewSettingsHandler creates a new SettingsHandler
func NewSettingsHandler(settingsSvc *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		settingsSvc: settingsSvc,
	}
}

// GetSettings returns current system settings (read-only, configured via Helm)
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	settings := h.settingsSvc.GetSettings()
	c.JSON(http.StatusOK, settings)
}

// GetNodeMetrics returns Prometheus metrics for a node
func (h *SettingsHandler) GetNodeMetrics(c *gin.Context) {
	nodeName := c.Param("name")
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))

	metrics, err := h.settingsSvc.GetNodeMetrics(c.Request.Context(), nodeName, hours)
	if err != nil {
		logger.Error("Failed to get node metrics", "node", nodeName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

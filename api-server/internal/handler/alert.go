package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// AlertHandler handles alert-related requests
type AlertHandler struct {
	alertSvc *service.AlertService
}

// NewAlertHandler creates a new AlertHandler
func NewAlertHandler(alertSvc *service.AlertService) *AlertHandler {
	return &AlertHandler{
		alertSvc: alertSvc,
	}
}

// GetAlertConfig returns the alert configuration
func (h *AlertHandler) GetAlertConfig(c *gin.Context) {
	config, err := h.alertSvc.GetConfig(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get alert config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateAlertConfig updates the alert configuration
func (h *AlertHandler) UpdateAlertConfig(c *gin.Context) {
	var config service.AlertConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.alertSvc.SetConfig(c.Request.Context(), &config); err != nil {
		logger.Error("Failed to update alert config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

// TestChannel tests a notification channel
func (h *AlertHandler) TestChannel(c *gin.Context) {
	var channel service.NotifyChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.alertSvc.TestChannel(c.Request.Context(), &channel); err != nil {
		logger.Error("Failed to test channel", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test notification sent"})
}

// GetAlertHistory returns alert history
func (h *AlertHandler) GetAlertHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	history, err := h.alertSvc.GetHistory(c.Request.Context(), limit)
	if err != nil {
		logger.Error("Failed to get alert history", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": history})
}


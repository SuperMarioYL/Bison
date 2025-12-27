package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// StatusHandler handles system status requests
type StatusHandler struct {
	statusSvc *service.StatusService
}

// NewStatusHandler creates a new StatusHandler
func NewStatusHandler(statusSvc *service.StatusService) *StatusHandler {
	return &StatusHandler{
		statusSvc: statusSvc,
	}
}

// GetStatus returns overall system status
func (h *StatusHandler) GetStatus(c *gin.Context) {
	status, err := h.statusSvc.GetStatus(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get system status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetTaskHistory returns recent task executions
func (h *StatusHandler) GetTaskHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	tasks, err := h.statusSvc.GetTaskHistory(c.Request.Context(), limit)
	if err != nil {
		logger.Error("Failed to get task history", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": tasks})
}

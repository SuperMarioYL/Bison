package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// AuditHandler handles audit-related requests
type AuditHandler struct {
	auditSvc *service.AuditService
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditSvc: auditSvc,
	}
}

// ListLogs returns audit logs with filtering
func (h *AuditHandler) ListLogs(c *gin.Context) {
	filter := &service.AuditFilter{
		Action:   c.Query("action"),
		Resource: c.Query("resource"),
		Operator: c.Query("operator"),
		Target:   c.Query("target"),
	}

	// Parse date filters
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = t
		}
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	result, err := h.auditSvc.Query(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		logger.Error("Failed to query audit logs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRecentLogs returns recent audit logs
func (h *AuditHandler) GetRecentLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	logs, err := h.auditSvc.GetRecent(c.Request.Context(), limit)
	if err != nil {
		logger.Error("Failed to get recent audit logs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": logs})
}


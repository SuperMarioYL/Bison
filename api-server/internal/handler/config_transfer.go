package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// ConfigTransferHandler handles configuration import/export requests
type ConfigTransferHandler struct {
	configTransferSvc *service.ConfigTransferService
}

// NewConfigTransferHandler creates a new ConfigTransferHandler
func NewConfigTransferHandler(svc *service.ConfigTransferService) *ConfigTransferHandler {
	return &ConfigTransferHandler{
		configTransferSvc: svc,
	}
}

// ExportConfig exports configuration as a JSON file download
func (h *ConfigTransferHandler) ExportConfig(c *gin.Context) {
	sectionsParam := c.DefaultQuery("sections", strings.Join(service.AllSections, ","))
	includeSensitive := c.DefaultQuery("includeSensitive", "false") == "true"

	sections := strings.Split(sectionsParam, ",")
	for i := range sections {
		sections[i] = strings.TrimSpace(sections[i])
	}

	operator := "admin"
	if username, exists := c.Get("username"); exists {
		operator = username.(string)
	}

	config, err := h.configTransferSvc.Export(c.Request.Context(), sections, includeSensitive, operator)
	if err != nil {
		logger.Error("Failed to export config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal export config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "序列化配置失败"})
		return
	}

	filename := fmt.Sprintf("bison-config-%s.json", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/json", data)
}

// PreviewImport validates and previews an import configuration
func (h *ConfigTransferHandler) PreviewImport(c *gin.Context) {
	var config service.ExportConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 JSON 格式: " + err.Error()})
		return
	}

	if config.Version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 version 字段"})
		return
	}
	if config.Sections == nil || len(config.Sections) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 sections 字段"})
		return
	}

	result, err := h.configTransferSvc.Preview(c.Request.Context(), &config)
	if err != nil {
		logger.Error("Failed to preview import", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ApplyImport applies the imported configuration
func (h *ConfigTransferHandler) ApplyImport(c *gin.Context) {
	var req service.ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式: " + err.Error()})
		return
	}

	if len(req.Sections) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择至少一个配置模块"})
		return
	}

	if req.Config.Version == "" || req.Config.Sections == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置数据"})
		return
	}

	result, err := h.configTransferSvc.Apply(c.Request.Context(), &req)
	if err != nil {
		logger.Error("Failed to apply import", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

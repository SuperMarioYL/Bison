package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// NodeHandler handles node management requests
type NodeHandler struct {
	nodeSvc *service.NodeService
}

// NewNodeHandler creates a new NodeHandler
func NewNodeHandler(nodeSvc *service.NodeService) *NodeHandler {
	return &NodeHandler{
		nodeSvc: nodeSvc,
	}
}

// ListNodes returns all nodes with their Bison status
// GET /api/v1/nodes
func (h *NodeHandler) ListNodes(c *gin.Context) {
	nodes, err := h.nodeSvc.ListNodes(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

// GetNode returns detailed information about a node
// GET /api/v1/nodes/:name
func (h *NodeHandler) GetNode(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node name is required"})
		return
	}

	node, err := h.nodeSvc.GetNode(c.Request.Context(), name)
	if err != nil {
		logger.Error("Failed to get node", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, node)
}

// EnableNode enables a node for Bison management
// POST /api/v1/nodes/:name/enable
func (h *NodeHandler) EnableNode(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node name is required"})
		return
	}

	if err := h.nodeSvc.EnableNode(c.Request.Context(), name); err != nil {
		logger.Error("Failed to enable node", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node enabled successfully"})
}

// DisableNode disables a node from Bison management
// POST /api/v1/nodes/:name/disable
func (h *NodeHandler) DisableNode(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node name is required"})
		return
	}

	if err := h.nodeSvc.DisableNode(c.Request.Context(), name); err != nil {
		logger.Error("Failed to disable node", "name", name, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node disabled successfully"})
}

// AssignNodeToTeam exclusively assigns a node to a team
// POST /api/v1/nodes/:name/assign
func (h *NodeHandler) AssignNodeToTeam(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node name is required"})
		return
	}

	var req struct {
		Team string `json:"team" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team name is required"})
		return
	}

	if err := h.nodeSvc.AssignNodeToTeam(c.Request.Context(), name, req.Team); err != nil {
		logger.Error("Failed to assign node to team", "node", name, "team", req.Team, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node assigned to team successfully"})
}

// ReleaseNode releases a node from exclusive assignment
// POST /api/v1/nodes/:name/release
func (h *NodeHandler) ReleaseNode(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node name is required"})
		return
	}

	if err := h.nodeSvc.ReleaseNodeFromTeam(c.Request.Context(), name); err != nil {
		logger.Error("Failed to release node", "name", name, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node released successfully"})
}

// GetSharedNodes returns all nodes in the shared pool
// GET /api/v1/nodes/shared
func (h *NodeHandler) GetSharedNodes(c *gin.Context) {
	nodes, err := h.nodeSvc.GetSharedNodes(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get shared nodes", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

// GetTeamNodes returns all nodes exclusively assigned to a team
// GET /api/v1/nodes/team/:team
func (h *NodeHandler) GetTeamNodes(c *gin.Context) {
	team := c.Param("team")
	if team == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "team name is required"})
		return
	}

	nodes, err := h.nodeSvc.GetTeamNodes(c.Request.Context(), team)
	if err != nil {
		logger.Error("Failed to get team nodes", "team", team, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

// GetNodeStatusSummary returns a summary of node statuses
// GET /api/v1/nodes/summary
func (h *NodeHandler) GetNodeStatusSummary(c *gin.Context) {
	summary, err := h.nodeSvc.GetNodeStatusSummary(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get node status summary", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}


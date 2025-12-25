package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// TeamHandler handles team-related API requests
type TeamHandler struct {
	tenantSvc *service.TenantService
	costSvc   *service.CostService
	nodeSvc   *service.NodeService
}

// NewTeamHandler creates a new TeamHandler
func NewTeamHandler(tenantSvc *service.TenantService, costSvc *service.CostService, nodeSvc *service.NodeService) *TeamHandler {
	return &TeamHandler{
		tenantSvc: tenantSvc,
		costSvc:   costSvc,
		nodeSvc:   nodeSvc,
	}
}

// ListTeams returns all teams
func (h *TeamHandler) ListTeams(c *gin.Context) {
	teams, err := h.tenantSvc.List(c.Request.Context())
	if err != nil {
		logger.Error("Failed to list teams", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich with usage data if cost service is enabled
	if h.costSvc.IsEnabled() {
		window := c.DefaultQuery("window", "7d")
		for _, team := range teams {
			usage, _ := h.costSvc.GetTeamUsageByName(c.Request.Context(), team.Name, window)
			if usage != nil {
				// Add usage info (could extend Team struct or return separately)
				_ = usage
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": teams})
}

// GetTeam returns a specific team
func (h *TeamHandler) GetTeam(c *gin.Context) {
	name := c.Param("name")

	team, err := h.tenantSvc.Get(c.Request.Context(), name)
	if err != nil {
		logger.Error("Failed to get team", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Get usage data
	window := c.DefaultQuery("window", "7d")
	usage, _ := h.costSvc.GetTeamUsageByName(c.Request.Context(), name, window)

	c.JSON(http.StatusOK, gin.H{
		"team":  team,
		"usage": usage,
	})
}

// CreateTeam creates a new team
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var req struct {
		Name           string             `json:"name" binding:"required"`
		DisplayName    string             `json:"displayName"`
		Description    string             `json:"description"`
		Owners         []service.OwnerRef `json:"owners" binding:"required"`
		Mode           service.TeamMode   `json:"mode"` // "shared" or "exclusive"
		ExclusiveNodes []string           `json:"exclusiveNodes"`
		Quota          map[string]string  `json:"quota"` // Dynamic quota
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request for CreateTeam", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to shared mode
	if req.Mode == "" {
		req.Mode = service.TeamModeShared
	}

	team := &service.Team{
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Owners:         req.Owners,
		Mode:           req.Mode,
		ExclusiveNodes: req.ExclusiveNodes,
		Quota:          req.Quota,
	}

	if team.DisplayName == "" {
		team.DisplayName = team.Name
	}

	// Validate exclusive mode
	if team.Mode == service.TeamModeExclusive && len(team.ExclusiveNodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exclusive mode requires at least one node"})
		return
	}

	// Create the tenant first
	if err := h.tenantSvc.Create(c.Request.Context(), team); err != nil {
		logger.Error("Failed to create team", "name", req.Name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Assign nodes if exclusive mode
	if team.Mode == service.TeamModeExclusive && h.nodeSvc != nil {
		for _, nodeName := range team.ExclusiveNodes {
			if err := h.nodeSvc.AssignNodeToTeam(c.Request.Context(), nodeName, team.Name); err != nil {
				logger.Warn("Failed to assign node to team", "node", nodeName, "team", team.Name, "error", err)
				// Continue with other nodes, don't fail the whole operation
			}
		}
	}

	c.JSON(http.StatusCreated, team)
}

// UpdateTeam updates an existing team
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		DisplayName    string             `json:"displayName"`
		Description    string             `json:"description"`
		Owners         []service.OwnerRef `json:"owners"`
		Mode           service.TeamMode   `json:"mode"`
		ExclusiveNodes []string           `json:"exclusiveNodes"`
		Quota          map[string]string  `json:"quota"` // Dynamic quota
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Invalid request for UpdateTeam", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing team to compare mode changes
	existingTeam, err := h.tenantSvc.Get(c.Request.Context(), name)
	if err != nil {
		logger.Error("Failed to get existing team", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Default mode if not specified
	if req.Mode == "" {
		req.Mode = existingTeam.Mode
	}

	team := &service.Team{
		Name:           name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Owners:         req.Owners,
		Mode:           req.Mode,
		ExclusiveNodes: req.ExclusiveNodes,
		Quota:          req.Quota,
	}

	// Validate exclusive mode
	if team.Mode == service.TeamModeExclusive && len(team.ExclusiveNodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exclusive mode requires at least one node"})
		return
	}

	// Handle node assignments based on mode change
	if h.nodeSvc != nil {
		ctx := c.Request.Context()

		// If switching from exclusive to shared, release old nodes
		if existingTeam.Mode == service.TeamModeExclusive && team.Mode == service.TeamModeShared {
			for _, nodeName := range existingTeam.ExclusiveNodes {
				if err := h.nodeSvc.ReleaseNodeFromTeam(ctx, nodeName); err != nil {
					logger.Warn("Failed to release node from team", "node", nodeName, "error", err)
				}
			}
		}

		// If in exclusive mode, update node assignments
		if team.Mode == service.TeamModeExclusive {
			// Release nodes that are no longer in the list
			oldNodes := make(map[string]bool)
			for _, n := range existingTeam.ExclusiveNodes {
				oldNodes[n] = true
			}
			newNodes := make(map[string]bool)
			for _, n := range team.ExclusiveNodes {
				newNodes[n] = true
			}

			// Release removed nodes
			for nodeName := range oldNodes {
				if !newNodes[nodeName] {
					if err := h.nodeSvc.ReleaseNodeFromTeam(ctx, nodeName); err != nil {
						logger.Warn("Failed to release node", "node", nodeName, "error", err)
					}
				}
			}

			// Assign new nodes
			for nodeName := range newNodes {
				if !oldNodes[nodeName] {
					if err := h.nodeSvc.AssignNodeToTeam(ctx, nodeName, team.Name); err != nil {
						logger.Warn("Failed to assign node", "node", nodeName, "team", team.Name, "error", err)
					}
				}
			}
		}
	}

	if err := h.tenantSvc.Update(c.Request.Context(), name, team); err != nil {
		logger.Error("Failed to update team", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, team)
}

// DeleteTeam deletes a team
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	name := c.Param("name")

	// Get team to check for exclusive nodes
	team, err := h.tenantSvc.Get(c.Request.Context(), name)
	if err != nil {
		logger.Error("Failed to get team for deletion", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Release exclusive nodes back to shared pool
	if team.Mode == service.TeamModeExclusive && h.nodeSvc != nil {
		for _, nodeName := range team.ExclusiveNodes {
			if err := h.nodeSvc.ReleaseNodeFromTeam(c.Request.Context(), nodeName); err != nil {
				logger.Warn("Failed to release node during team deletion", "node", nodeName, "error", err)
			}
		}
	}

	if err := h.tenantSvc.Delete(c.Request.Context(), name); err != nil {
		logger.Error("Failed to delete team", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
}

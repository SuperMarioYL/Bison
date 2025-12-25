package handler

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/bison/api-server/internal/service"
	"github.com/bison/api-server/pkg/logger"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userSvc    *service.UserService
	tenantSvc  *service.TenantService
	projectSvc *service.ProjectService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userSvc *service.UserService, tenantSvc *service.TenantService, projectSvc *service.ProjectService) *UserHandler {
	return &UserHandler{
		userSvc:    userSvc,
		tenantSvc:  tenantSvc,
		projectSvc: projectSvc,
	}
}

// ListUsers returns all users with optional filtering
func (h *UserHandler) ListUsers(c *gin.Context) {
	query := c.Query("q")
	status := c.Query("status")
	source := c.Query("source")

	users, err := h.userSvc.Search(c.Request.Context(), query, status, source)
	if err != nil {
		logger.Error("Failed to list users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": users})
}

// GetUser returns a specific user by email
func (h *UserHandler) GetUser(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	user, err := h.userSvc.GetDetail(c.Request.Context(), email, h.tenantSvc, h.projectSvc)
	if err != nil {
		logger.Error("Failed to get user", "email", email, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Get usage statistics
	usage, _ := h.userSvc.GetUsage(c.Request.Context(), email, "7d")
	user.Usage = usage

	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		DisplayName string `json:"displayName"`
		Status      string `json:"status"`
		InitialTeam string `json:"initialTeam,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &service.User{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Source:      "manual",
		Status:      req.Status,
	}

	if err := h.userSvc.Create(c.Request.Context(), user); err != nil {
		logger.Error("Failed to create user", "email", req.Email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add to initial team if specified
	if req.InitialTeam != "" && h.tenantSvc != nil {
		owner := service.OwnerRef{
			Kind: "User",
			Name: req.Email,
		}
		if err := h.tenantSvc.AddOwner(c.Request.Context(), req.InitialTeam, owner); err != nil {
			logger.Warn("Failed to add user to initial team", "user", req.Email, "team", req.InitialTeam, "error", err)
		}
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	var req struct {
		DisplayName string `json:"displayName"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing user
	existing, err := h.userSvc.Get(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	if err := h.userSvc.Update(c.Request.Context(), email, existing); err != nil {
		logger.Error("Failed to update user", "email", email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	// Remove user from all teams
	if h.tenantSvc != nil {
		teams, _ := h.tenantSvc.List(c.Request.Context())
		for _, team := range teams {
			for _, owner := range team.Owners {
				if owner.Kind == "User" && owner.Name == email {
					h.tenantSvc.RemoveOwner(c.Request.Context(), team.Name, owner)
					break
				}
			}
		}
	}

	// Remove user from all projects
	if h.projectSvc != nil {
		projects, _ := h.projectSvc.List(c.Request.Context())
		for _, project := range projects {
			for _, member := range project.Members {
				if member.User == email {
					h.projectSvc.RemoveMember(c.Request.Context(), project.Name, email)
					break
				}
			}
		}
	}

	if err := h.userSvc.Delete(c.Request.Context(), email); err != nil {
		logger.Error("Failed to delete user", "email", email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

// SetUserStatus sets the status of a user (active/disabled)
func (h *UserHandler) SetUserStatus(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userSvc.SetStatus(c.Request.Context(), email, req.Status); err != nil {
		logger.Error("Failed to set user status", "email", email, "status", req.Status, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// GetUserUsage returns usage statistics for a user
func (h *UserHandler) GetUserUsage(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	window := c.DefaultQuery("window", "7d")

	usage, err := h.userSvc.GetUsage(c.Request.Context(), email, window)
	if err != nil {
		logger.Error("Failed to get user usage", "email", email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// AddUserToTeam adds a user to a team
func (h *UserHandler) AddUserToTeam(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	var req struct {
		TeamName string `json:"teamName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	owner := service.OwnerRef{
		Kind: "User",
		Name: email,
	}

	if err := h.tenantSvc.AddOwner(c.Request.Context(), req.TeamName, owner); err != nil {
		logger.Error("Failed to add user to team", "user", email, "team", req.TeamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user added to team"})
}

// RemoveUserFromTeam removes a user from a team
func (h *UserHandler) RemoveUserFromTeam(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	teamName := c.Param("teamName")

	owner := service.OwnerRef{
		Kind: "User",
		Name: email,
	}

	if err := h.tenantSvc.RemoveOwner(c.Request.Context(), teamName, owner); err != nil {
		logger.Error("Failed to remove user from team", "user", email, "team", teamName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed from team"})
}

// AddUserToProject adds a user to a project with a role
func (h *UserHandler) AddUserToProject(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}

	var req struct {
		ProjectName string `json:"projectName" binding:"required"`
		Role        string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member := service.ProjectMember{
		User: email,
		Role: req.Role,
	}

	if err := h.projectSvc.AddMember(c.Request.Context(), req.ProjectName, member); err != nil {
		logger.Error("Failed to add user to project", "user", email, "project", req.ProjectName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user added to project"})
}

// RemoveUserFromProject removes a user from a project
func (h *UserHandler) RemoveUserFromProject(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	projectName := c.Param("projectName")

	if err := h.projectSvc.RemoveMember(c.Request.Context(), projectName, email); err != nil {
		logger.Error("Failed to remove user from project", "user", email, "project", projectName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user removed from project"})
}

// UpdateUserProjectRole updates a user's role in a project
func (h *UserHandler) UpdateUserProjectRole(c *gin.Context) {
	email, err := url.PathUnescape(c.Param("email"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	projectName := c.Param("projectName")

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.projectSvc.UpdateMemberRole(c.Request.Context(), projectName, email, req.Role); err != nil {
		logger.Error("Failed to update user project role", "user", email, "project", projectName, "role", req.Role, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

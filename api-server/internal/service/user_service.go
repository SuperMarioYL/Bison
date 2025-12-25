package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/opencost"
	"github.com/bison/api-server/pkg/logger"
)

const (
	usersConfigMapName      = "bison-users"
	usersConfigMapNamespace = "bison-system"
	usersDataKey            = "users.json"
)

// User represents a user in the system
type User struct {
	Email       string `json:"email"`                 // Unique identifier
	DisplayName string `json:"displayName"`           // Display name
	Source      string `json:"source"`                // "manual" or "oidc"
	Status      string `json:"status"`                // "active" or "disabled"
	CreatedAt   string `json:"createdAt"`             // ISO 8601 timestamp
	LastLogin   string `json:"lastLogin,omitempty"`   // ISO 8601 timestamp
}

// UserData represents the data stored in ConfigMap
type UserData struct {
	Users []User `json:"users"`
}

// UserDetail represents detailed user information
type UserDetail struct {
	User
	Teams    []UserTeamDetail    `json:"teams"`
	Projects []UserProjectDetail `json:"projects"`
	Usage    *UsageData          `json:"usage,omitempty"`
}

// UserTeamDetail represents a user's relationship with a team
type UserTeamDetail struct {
	TeamName    string `json:"teamName"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"` // "owner"
	JoinedAt    string `json:"joinedAt,omitempty"`
}

// UserProjectDetail represents a user's relationship with a project
type UserProjectDetail struct {
	ProjectName string `json:"projectName"`
	DisplayName string `json:"displayName"`
	TeamName    string `json:"teamName"`
	Role        string `json:"role"` // "admin", "edit", "view"
}

// UserService handles user operations
type UserService struct {
	k8sClient      *k8s.Client
	opencostClient *opencost.Client
}

// NewUserService creates a new UserService
func NewUserService(k8sClient *k8s.Client, opencostClient *opencost.Client) *UserService {
	return &UserService{
		k8sClient:      k8sClient,
		opencostClient: opencostClient,
	}
}

// List returns all users
func (s *UserService) List(ctx context.Context) ([]*User, error) {
	logger.Debug("Listing users")

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return nil, err
	}

	var users []*User
	for i := range userData.Users {
		users = append(users, &userData.Users[i])
	}

	return users, nil
}

// Get returns a specific user by email
func (s *UserService) Get(ctx context.Context, email string) (*User, error) {
	logger.Debug("Getting user", "email", email)

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return nil, err
	}

	for _, user := range userData.Users {
		if user.Email == email {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", email)
}

// GetDetail returns detailed user information including teams and projects
func (s *UserService) GetDetail(ctx context.Context, email string, tenantSvc *TenantService, projectSvc *ProjectService) (*UserDetail, error) {
	logger.Debug("Getting user detail", "email", email)

	user, err := s.Get(ctx, email)
	if err != nil {
		return nil, err
	}

	detail := &UserDetail{
		User:     *user,
		Teams:    []UserTeamDetail{},
		Projects: []UserProjectDetail{},
	}

	// Get teams this user belongs to
	if tenantSvc != nil {
		teams, err := tenantSvc.List(ctx)
		if err == nil {
			for _, team := range teams {
				for _, owner := range team.Owners {
					if owner.Kind == "User" && owner.Name == email {
						detail.Teams = append(detail.Teams, UserTeamDetail{
							TeamName:    team.Name,
							DisplayName: team.DisplayName,
							Role:        "owner",
						})
						break
					}
				}
			}
		}
	}

	// Get projects this user has access to
	if projectSvc != nil {
		projects, err := projectSvc.List(ctx)
		if err == nil {
			for _, project := range projects {
				for _, member := range project.Members {
					if member.User == email {
						detail.Projects = append(detail.Projects, UserProjectDetail{
							ProjectName: project.Name,
							DisplayName: project.DisplayName,
							TeamName:    project.Team,
							Role:        member.Role,
						})
						break
					}
				}
			}
		}
	}

	return detail, nil
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, user *User) error {
	logger.Info("Creating user", "email", user.Email)

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return err
	}

	// Check if user already exists
	for _, u := range userData.Users {
		if u.Email == user.Email {
			return fmt.Errorf("user already exists: %s", user.Email)
		}
	}

	// Set defaults
	if user.Source == "" {
		user.Source = "manual"
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.CreatedAt == "" {
		user.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	userData.Users = append(userData.Users, *user)

	return s.saveUserData(ctx, userData)
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, email string, updates *User) error {
	logger.Info("Updating user", "email", email)

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return err
	}

	found := false
	for i, u := range userData.Users {
		if u.Email == email {
			// Preserve immutable fields
			updates.Email = email
			updates.CreatedAt = u.CreatedAt
			if updates.Source == "" {
				updates.Source = u.Source
			}
			if updates.LastLogin == "" {
				updates.LastLogin = u.LastLogin
			}
			userData.Users[i] = *updates
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("user not found: %s", email)
	}

	return s.saveUserData(ctx, userData)
}

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, email string) error {
	logger.Info("Deleting user", "email", email)

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return err
	}

	found := false
	for i, u := range userData.Users {
		if u.Email == email {
			userData.Users = append(userData.Users[:i], userData.Users[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("user not found: %s", email)
	}

	return s.saveUserData(ctx, userData)
}

// UpdateLastLogin updates the last login time for a user
func (s *UserService) UpdateLastLogin(ctx context.Context, email string) error {
	logger.Debug("Updating last login", "email", email)

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return err
	}

	for i, u := range userData.Users {
		if u.Email == email {
			userData.Users[i].LastLogin = time.Now().UTC().Format(time.RFC3339)
			return s.saveUserData(ctx, userData)
		}
	}

	// User not found - create if OIDC login
	newUser := User{
		Email:     email,
		DisplayName: extractDisplayName(email),
		Source:    "oidc",
		Status:    "active",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		LastLogin: time.Now().UTC().Format(time.RFC3339),
	}
	userData.Users = append(userData.Users, newUser)

	return s.saveUserData(ctx, userData)
}

// SetStatus sets the status of a user (active/disabled)
func (s *UserService) SetStatus(ctx context.Context, email string, status string) error {
	logger.Info("Setting user status", "email", email, "status", status)

	if status != "active" && status != "disabled" {
		return fmt.Errorf("invalid status: %s", status)
	}

	userData, err := s.loadUserData(ctx)
	if err != nil {
		return err
	}

	for i, u := range userData.Users {
		if u.Email == email {
			userData.Users[i].Status = status
			return s.saveUserData(ctx, userData)
		}
	}

	return fmt.Errorf("user not found: %s", email)
}

// Search searches users by query
func (s *UserService) Search(ctx context.Context, query string, status string, source string) ([]*User, error) {
	logger.Debug("Searching users", "query", query, "status", status, "source", source)

	users, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var results []*User
	query = strings.ToLower(query)

	for _, user := range users {
		// Filter by status
		if status != "" && status != "all" && user.Status != status {
			continue
		}

		// Filter by source
		if source != "" && source != "all" && user.Source != source {
			continue
		}

		// Filter by query (email or displayName)
		if query != "" {
			emailMatch := strings.Contains(strings.ToLower(user.Email), query)
			nameMatch := strings.Contains(strings.ToLower(user.DisplayName), query)
			if !emailMatch && !nameMatch {
				continue
			}
		}

		results = append(results, user)
	}

	return results, nil
}

// GetUsage returns usage statistics for a user
func (s *UserService) GetUsage(ctx context.Context, email, window string) (*UsageData, error) {
	logger.Debug("Getting user usage", "email", email, "window", window)

	if s.opencostClient == nil || !s.opencostClient.IsEnabled() {
		return &UsageData{Name: email}, nil
	}

	if window == "" {
		window = "7d"
	}

	// Try to get usage by user label
	summaries, err := s.opencostClient.GetUserUsage(ctx, window)
	if err != nil {
		logger.Warn("Failed to get user usage from OpenCost", "error", err)
		return &UsageData{Name: email}, nil
	}

	// Find this user's usage
	for _, summary := range summaries {
		if summary.Name == email {
			return &UsageData{
				Name:         summary.Name,
				CPUCoreHours: summary.CPUCoreHours,
				RAMGBHours:   summary.RAMGBHours,
				GPUHours:     summary.GPUHours,
				TotalCost:    summary.TotalCost,
				CPUCost:      summary.CPUCost,
				RAMCost:      summary.RAMCost,
				GPUCost:      summary.GPUCost,
				Minutes:      summary.Minutes,
			}, nil
		}
	}

	return &UsageData{Name: email}, nil
}

// loadUserData loads user data from ConfigMap
func (s *UserService) loadUserData(ctx context.Context) (*UserData, error) {
	cm, err := s.k8sClient.GetConfigMap(ctx, usersConfigMapNamespace, usersConfigMapName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty data if ConfigMap doesn't exist
			return &UserData{Users: []User{}}, nil
		}
		return nil, fmt.Errorf("failed to get users ConfigMap: %w", err)
	}

	data := cm.Data[usersDataKey]
	if data == "" {
		return &UserData{Users: []User{}}, nil
	}

	var userData UserData
	if err := json.Unmarshal([]byte(data), &userData); err != nil {
		return nil, fmt.Errorf("failed to parse users data: %w", err)
	}

	return &userData, nil
}

// saveUserData saves user data to ConfigMap
func (s *UserService) saveUserData(ctx context.Context, userData *UserData) error {
	data, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal users data: %w", err)
	}

	cm, err := s.k8sClient.GetConfigMap(ctx, usersConfigMapNamespace, usersConfigMapName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create ConfigMap if it doesn't exist
			newCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      usersConfigMapName,
					Namespace: usersConfigMapNamespace,
				},
				Data: map[string]string{
					usersDataKey: string(data),
				},
			}
			return s.k8sClient.CreateConfigMap(ctx, usersConfigMapNamespace, newCM)
		}
		return fmt.Errorf("failed to get users ConfigMap: %w", err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[usersDataKey] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, usersConfigMapNamespace, cm)
}

// Helper function to extract display name from email
func extractDisplayName(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		name := parts[0]
		// Capitalize first letter
		if len(name) > 0 {
			return strings.ToUpper(string(name[0])) + name[1:]
		}
		return name
	}
	return email
}

package service

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

// Project represents a project (Namespace under a Capsule Tenant)
type Project struct {
	Name        string          `json:"name"`
	Team        string          `json:"team"` // Parent team (Tenant)
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
	Members     []ProjectMember `json:"members,omitempty"`
	Status      string          `json:"status"`
}

// ProjectMember represents a member of a project
type ProjectMember struct {
	User string `json:"user"` // User email
	Role string `json:"role"` // admin, edit, view
}

// RoleMapping maps project roles to ClusterRoles
var RoleMapping = map[string]string{
	"admin": "admin", // Full control
	"edit":  "edit",  // Edit most resources, no RBAC
	"view":  "view",  // Read-only access
}

// ProjectService handles project (Namespace) operations
type ProjectService struct {
	k8sClient *k8s.Client
}

// NewProjectService creates a new ProjectService
func NewProjectService(k8sClient *k8s.Client) *ProjectService {
	return &ProjectService{
		k8sClient: k8sClient,
	}
}

// List returns all projects
func (s *ProjectService) List(ctx context.Context) ([]*Project, error) {
	return s.ListByTeam(ctx, "")
}

// ListByTeam returns all projects for a specific team
func (s *ProjectService) ListByTeam(ctx context.Context, teamName string) ([]*Project, error) {
	logger.Debug("Listing projects", "team", teamName)

	labelSelector := "bison.io/managed=true"
	if teamName != "" {
		labelSelector = fmt.Sprintf("capsule.clastix.io/tenant=%s,bison.io/managed=true", teamName)
	}

	namespaces, err := s.k8sClient.ListNamespaces(ctx, labelSelector)
	if err != nil {
		logger.Error("Failed to list namespaces", "error", err)
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var projects []*Project
	for _, ns := range namespaces.Items {
		project := s.namespaceToProject(&ns)

		// Get members from annotations
		project.Members = s.getMembersFromAnnotations(&ns)

		projects = append(projects, project)
	}

	return projects, nil
}

// Get returns a specific project by name
func (s *ProjectService) Get(ctx context.Context, name string) (*Project, error) {
	logger.Debug("Getting project", "name", name)

	ns, err := s.k8sClient.GetNamespace(ctx, name)
	if err != nil {
		logger.Error("Failed to get namespace", "name", name, "error", err)
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project := s.namespaceToProject(ns)

	// Get members from annotations
	project.Members = s.getMembersFromAnnotations(ns)

	return project, nil
}

// Create creates a new project (Namespace)
func (s *ProjectService) Create(ctx context.Context, project *Project) error {
	logger.Info("Creating project", "name", project.Name, "team", project.Team)

	labels := map[string]string{
		"bison.io/managed":          "true",
		"bison.io/project":          project.Name,
		"capsule.clastix.io/tenant": project.Team,
	}

	// Create namespace
	if err := s.k8sClient.CreateNamespace(ctx, project.Name, labels); err != nil {
		logger.Error("Failed to create namespace", "name", project.Name, "error", err)
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Update annotations
	ns, _ := s.k8sClient.GetNamespace(ctx, project.Name)
	if ns != nil {
		if ns.Annotations == nil {
			ns.Annotations = make(map[string]string)
		}
		ns.Annotations["bison.io/display-name"] = project.DisplayName
		ns.Annotations["bison.io/description"] = project.Description

		// Store members in annotations
		if len(project.Members) > 0 {
			membersJSON, _ := json.Marshal(project.Members)
			ns.Annotations["bison.io/members"] = string(membersJSON)
		}

		// Merge existing labels with our labels
		for k, v := range labels {
			ns.Labels[k] = v
		}
		if err := s.k8sClient.UpdateNamespaceLabels(ctx, project.Name, ns.Labels); err != nil {
			logger.Warn("Failed to update namespace labels", "name", project.Name, "error", err)
		}
	}

	// Create RoleBindings for members
	for _, member := range project.Members {
		if err := s.createMemberRoleBinding(ctx, project.Name, member); err != nil {
			logger.Warn("Failed to create role binding for member", "project", project.Name, "user", member.User, "error", err)
		}
	}

	return nil
}

// Update updates an existing project
func (s *ProjectService) Update(ctx context.Context, name string, project *Project) error {
	logger.Info("Updating project", "name", name)

	ns, err := s.k8sClient.GetNamespace(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Update labels
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels["bison.io/managed"] = "true"
	ns.Labels["bison.io/project"] = name

	// Update annotations
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	ns.Annotations["bison.io/display-name"] = project.DisplayName
	ns.Annotations["bison.io/description"] = project.Description

	// Store members in annotations
	if len(project.Members) > 0 {
		membersJSON, _ := json.Marshal(project.Members)
		ns.Annotations["bison.io/members"] = string(membersJSON)
	} else {
		delete(ns.Annotations, "bison.io/members")
	}

	// Update namespace (including labels and annotations)
	if err := s.k8sClient.UpdateNamespace(ctx, ns); err != nil {
		logger.Error("Failed to update namespace", "name", name, "error", err)
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// Delete deletes a project
func (s *ProjectService) Delete(ctx context.Context, name string) error {
	logger.Info("Deleting project", "name", name)

	if err := s.k8sClient.DeleteNamespace(ctx, name); err != nil {
		logger.Error("Failed to delete namespace", "name", name, "error", err)
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// AddMember adds a member to a project
func (s *ProjectService) AddMember(ctx context.Context, projectName string, member ProjectMember) error {
	logger.Info("Adding member to project", "project", projectName, "user", member.User, "role", member.Role)

	project, err := s.Get(ctx, projectName)
	if err != nil {
		return err
	}

	// Check if member already exists
	for _, m := range project.Members {
		if m.User == member.User {
			return fmt.Errorf("member already exists: %s", member.User)
		}
	}

	// Add member
	project.Members = append(project.Members, member)

	// Update namespace annotations
	ns, err := s.k8sClient.GetNamespace(ctx, projectName)
	if err != nil {
		return err
	}

	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	membersJSON, _ := json.Marshal(project.Members)
	ns.Annotations["bison.io/members"] = string(membersJSON)

	// Update namespace (including annotations)
	if err := s.k8sClient.UpdateNamespace(ctx, ns); err != nil {
		return err
	}

	// Create RoleBinding
	return s.createMemberRoleBinding(ctx, projectName, member)
}

// RemoveMember removes a member from a project
func (s *ProjectService) RemoveMember(ctx context.Context, projectName string, userEmail string) error {
	logger.Info("Removing member from project", "project", projectName, "user", userEmail)

	project, err := s.Get(ctx, projectName)
	if err != nil {
		return err
	}

	// Find and remove member
	found := false
	var removedMember ProjectMember
	for i, m := range project.Members {
		if m.User == userEmail {
			removedMember = m
			project.Members = append(project.Members[:i], project.Members[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("member not found: %s", userEmail)
	}

	// Update namespace annotations
	ns, err := s.k8sClient.GetNamespace(ctx, projectName)
	if err != nil {
		return err
	}

	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}

	if len(project.Members) > 0 {
		membersJSON, _ := json.Marshal(project.Members)
		ns.Annotations["bison.io/members"] = string(membersJSON)
	} else {
		delete(ns.Annotations, "bison.io/members")
	}

	// Update namespace (including annotations)
	if err := s.k8sClient.UpdateNamespace(ctx, ns); err != nil {
		return err
	}

	// Delete RoleBinding
	return s.deleteMemberRoleBinding(ctx, projectName, removedMember)
}

// UpdateMemberRole updates a member's role in a project
func (s *ProjectService) UpdateMemberRole(ctx context.Context, projectName string, userEmail string, newRole string) error {
	logger.Info("Updating member role", "project", projectName, "user", userEmail, "role", newRole)

	project, err := s.Get(ctx, projectName)
	if err != nil {
		return err
	}

	// Find and update member
	found := false
	var oldMember ProjectMember
	for i, m := range project.Members {
		if m.User == userEmail {
			oldMember = m
			project.Members[i].Role = newRole
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("member not found: %s", userEmail)
	}

	// Update namespace annotations
	ns, err := s.k8sClient.GetNamespace(ctx, projectName)
	if err != nil {
		return err
	}

	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	membersJSON, _ := json.Marshal(project.Members)
	ns.Annotations["bison.io/members"] = string(membersJSON)

	// Update namespace (including annotations)
	if err := s.k8sClient.UpdateNamespace(ctx, ns); err != nil {
		return err
	}

	// Delete old RoleBinding
	if err := s.deleteMemberRoleBinding(ctx, projectName, oldMember); err != nil {
		logger.Warn("Failed to delete old role binding", "error", err)
	}

	// Create new RoleBinding
	return s.createMemberRoleBinding(ctx, projectName, ProjectMember{User: userEmail, Role: newRole})
}

// createMemberRoleBinding creates a RoleBinding for a project member
func (s *ProjectService) createMemberRoleBinding(ctx context.Context, namespace string, member ProjectMember) error {
	clusterRole, ok := RoleMapping[member.Role]
	if !ok {
		clusterRole = "view" // Default to view
	}

	bindingName := fmt.Sprintf("bison-%s-%s", sanitizeForK8s(member.User), member.Role)

	subjects := []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     member.User,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return s.k8sClient.CreateRoleBinding(ctx, namespace, bindingName, clusterRole, subjects)
}

// deleteMemberRoleBinding deletes a RoleBinding for a project member
func (s *ProjectService) deleteMemberRoleBinding(ctx context.Context, namespace string, member ProjectMember) error {
	bindingName := fmt.Sprintf("bison-%s-%s", sanitizeForK8s(member.User), member.Role)
	return s.k8sClient.DeleteRoleBinding(ctx, namespace, bindingName)
}

// getMembersFromAnnotations gets project members from namespace annotations
func (s *ProjectService) getMembersFromAnnotations(ns *corev1.Namespace) []ProjectMember {
	var members []ProjectMember

	if ns.Annotations == nil {
		return members
	}

	membersJSON := ns.Annotations["bison.io/members"]
	if membersJSON == "" {
		return members
	}

	if err := json.Unmarshal([]byte(membersJSON), &members); err != nil {
		logger.Warn("Failed to parse members annotation", "namespace", ns.Name, "error", err)
		return members
	}

	return members
}

// namespaceToProject converts a Namespace to a Project
func (s *ProjectService) namespaceToProject(ns *corev1.Namespace) *Project {
	project := &Project{
		Name:   ns.Name,
		Status: string(ns.Status.Phase),
	}

	// Get team from Capsule label
	if ns.Labels != nil {
		project.Team = ns.Labels["capsule.clastix.io/tenant"]
	}

	// Get display name and description from annotations
	if ns.Annotations != nil {
		project.DisplayName = ns.Annotations["bison.io/display-name"]
		project.Description = ns.Annotations["bison.io/description"]
	}
	if project.DisplayName == "" {
		project.DisplayName = project.Name
	}

	return project
}

// ResourceUsage represents usage of a single resource
type ResourceUsage struct {
	Name        string  `json:"name"`        // K8s resource name
	DisplayName string  `json:"displayName"` // Display name from config
	Unit        string  `json:"unit"`        // Display unit from config
	Used        float64 `json:"used"`        // Current usage (after divisor applied)
	RawUsed     float64 `json:"rawUsed"`     // Raw usage value
}

// ProjectUsage represents resource usage of a project
type ProjectUsage struct {
	ProjectName string          `json:"projectName"`
	Resources   []ResourceUsage `json:"resources"`
}

// GetProjectUsage returns dynamic resource usage for a project
func (s *ProjectService) GetProjectUsage(ctx context.Context, namespace string, resourceConfigs []ResourceDefinition) (*ProjectUsage, error) {
	logger.Debug("Getting project usage", "namespace", namespace)

	// Get all pods in namespace
	pods, err := s.k8sClient.ListPods(ctx, namespace, "")
	if err != nil {
		logger.Error("Failed to list pods", "namespace", namespace, "error", err)
		return nil, err
	}

	// Aggregate resource usage from all pods
	usageMap := make(map[string]float64)
	for _, pod := range pods.Items {
		// Skip pods that are not running
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		for _, container := range pod.Spec.Containers {
			for resourceName, quantity := range container.Resources.Requests {
				name := string(resourceName)
				usageMap[name] += quantity.AsApproximateFloat64()
			}
		}
	}

	// Build result based on enabled resource configs
	result := &ProjectUsage{
		ProjectName: namespace,
		Resources:   []ResourceUsage{},
	}

	for _, cfg := range resourceConfigs {
		if !cfg.Enabled {
			continue
		}

		rawUsed := usageMap[cfg.Name]
		divisor := cfg.Divisor
		if divisor <= 0 {
			divisor = 1
		}

		result.Resources = append(result.Resources, ResourceUsage{
			Name:        cfg.Name,
			DisplayName: cfg.DisplayName,
			Unit:        cfg.Unit,
			Used:        rawUsed / divisor,
			RawUsed:     rawUsed,
		})
	}

	return result, nil
}

// sanitizeForK8s sanitizes a string for use in K8s resource names
func sanitizeForK8s(s string) string {
	// Replace @ and . with -
	result := ""
	for _, c := range s {
		if c == '@' || c == '.' {
			result += "-"
		} else if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32) // Convert to lowercase
		}
	}
	// Ensure it doesn't start or end with -
	for len(result) > 0 && result[0] == '-' {
		result = result[1:]
	}
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	// Truncate to max length
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

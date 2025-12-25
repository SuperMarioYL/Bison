package service

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

// TeamMode represents the resource mode of a team
type TeamMode string

const (
	TeamModeShared    TeamMode = "shared"    // Team uses shared node pool
	TeamModeExclusive TeamMode = "exclusive" // Team has exclusive nodes
)

// Reserved team names that cannot be used
var reservedTeamNames = map[string]bool{
	"shared":    true,
	"disabled":  true,
	"unmanaged": true,
	"system":    true,
	"default":   true,
	"admin":     true,
}

// IsReservedTeamName checks if a team name is reserved
func IsReservedTeamName(name string) bool {
	return reservedTeamNames[name]
}

// GetExclusivePoolLabel returns the pool label value for exclusive mode
// Format: team-<team-name>
func GetExclusivePoolLabel(teamName string) string {
	return "team-" + teamName
}

// ParseExclusivePoolLabel extracts team name from exclusive pool label
// Returns empty string if not an exclusive label
func ParseExclusivePoolLabel(poolValue string) string {
	if len(poolValue) > 5 && poolValue[:5] == "team-" {
		return poolValue[5:]
	}
	return ""
}

// OwnerRef represents an owner reference (User or Group)
type OwnerRef struct {
	Kind string `json:"kind"` // "User" or "Group"
	Name string `json:"name"` // User email or group name
}

// Team represents a team (Capsule Tenant) in the system
type Team struct {
	Name           string            `json:"name"`
	DisplayName    string            `json:"displayName"`
	Description    string            `json:"description,omitempty"`
	Owners         []OwnerRef        `json:"owners"`                   // User or Group owners
	Mode           TeamMode          `json:"mode"`                     // "shared" or "exclusive"
	ExclusiveNodes []string          `json:"exclusiveNodes,omitempty"` // Node names for exclusive mode
	NodeSelector   map[string]string `json:"nodeSelector,omitempty"`   // Auto-generated based on mode
	Quota          map[string]string `json:"quota"`                    // Dynamic quota: {"cpu": "10", "memory": "20Gi", "nvidia.com/gpu": "4"}
	QuotaUsed      map[string]string `json:"quotaUsed,omitempty"`      // Aggregated quota usage from all projects
	ProjectCount   int               `json:"projectCount"`
	Status         TeamStatus        `json:"status,omitempty"`
	Suspended      bool              `json:"suspended"` // Whether team is suspended due to insufficient balance
}

// TeamStatus represents the current status of a team
type TeamStatus struct {
	Ready      bool   `json:"ready"`
	Namespaces int    `json:"namespaces"`
	State      string `json:"state"`
}

// TenantService handles Capsule Tenant operations
type TenantService struct {
	k8sClient *k8s.Client
}

// NewTenantService creates a new TenantService
func NewTenantService(k8sClient *k8s.Client) *TenantService {
	return &TenantService{
		k8sClient: k8sClient,
	}
}

// List returns all teams (Capsule Tenants)
func (s *TenantService) List(ctx context.Context) ([]*Team, error) {
	logger.Debug("Listing all tenants")

	tenants, err := s.k8sClient.ListTenants(ctx)
	if err != nil {
		logger.Error("Failed to list tenants", "error", err)
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	var teams []*Team
	for _, t := range tenants.Items {
		team, err := s.tenantToTeam(&t)
		if err != nil {
			logger.Warn("Failed to convert tenant to team", "name", t.GetName(), "error", err)
			continue
		}

		// For exclusive mode, calculate quota from node resources
		if team.Mode == TeamModeExclusive && len(team.ExclusiveNodes) > 0 {
			team.Quota = s.getExclusiveNodeResources(ctx, team.ExclusiveNodes)
		}

		// Aggregate resource usage from all projects (from Pods)
		team.QuotaUsed = s.getTeamResourceUsage(ctx, team.Name)
		teams = append(teams, team)
	}

	return teams, nil
}

// Get returns a specific team by name
func (s *TenantService) Get(ctx context.Context, name string) (*Team, error) {
	logger.Debug("Getting tenant", "name", name)

	tenant, err := s.k8sClient.GetTenant(ctx, name)
	if err != nil {
		logger.Error("Failed to get tenant", "name", name, "error", err)
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	team, err := s.tenantToTeam(tenant)
	if err != nil {
		return nil, err
	}

	// For exclusive mode, calculate quota from node resources
	if team.Mode == TeamModeExclusive && len(team.ExclusiveNodes) > 0 {
		team.Quota = s.getExclusiveNodeResources(ctx, team.ExclusiveNodes)
	}

	// Aggregate resource usage from all projects (from Pods)
	team.QuotaUsed = s.getTeamResourceUsage(ctx, name)

	return team, nil
}

// Create creates a new team (Capsule Tenant)
func (s *TenantService) Create(ctx context.Context, team *Team) error {
	logger.Info("Creating tenant", "name", team.Name)

	// Validate team name is not reserved
	if IsReservedTeamName(team.Name) {
		return fmt.Errorf("team name '%s' is reserved and cannot be used", team.Name)
	}

	tenant := s.teamToTenant(team)
	if err := s.k8sClient.CreateTenant(ctx, tenant); err != nil {
		logger.Error("Failed to create tenant", "name", team.Name, "error", err)
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// Update updates an existing team
func (s *TenantService) Update(ctx context.Context, name string, team *Team) error {
	logger.Info("Updating tenant", "name", name)

	// Get existing tenant to preserve resource version
	existing, err := s.k8sClient.GetTenant(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get existing tenant: %w", err)
	}

	// Update with new values
	updated := s.teamToTenant(team)
	updated.SetResourceVersion(existing.GetResourceVersion())
	updated.SetName(name) // Ensure name matches

	if err := s.k8sClient.UpdateTenant(ctx, updated); err != nil {
		logger.Error("Failed to update tenant", "name", name, "error", err)
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// Delete deletes a team and all its associated resources
func (s *TenantService) Delete(ctx context.Context, name string) error {
	logger.Info("Deleting tenant", "name", name)

	if err := s.k8sClient.DeleteTenant(ctx, name); err != nil {
		logger.Error("Failed to delete tenant", "name", name, "error", err)
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// AddOwner adds an owner to a team
func (s *TenantService) AddOwner(ctx context.Context, teamName string, owner OwnerRef) error {
	logger.Info("Adding owner to team", "team", teamName, "owner", owner.Name, "kind", owner.Kind)

	team, err := s.Get(ctx, teamName)
	if err != nil {
		return err
	}

	// Check if owner already exists
	for _, o := range team.Owners {
		if o.Kind == owner.Kind && o.Name == owner.Name {
			return fmt.Errorf("owner already exists: %s (%s)", owner.Name, owner.Kind)
		}
	}

	team.Owners = append(team.Owners, owner)
	return s.Update(ctx, teamName, team)
}

// RemoveOwner removes an owner from a team
func (s *TenantService) RemoveOwner(ctx context.Context, teamName string, owner OwnerRef) error {
	logger.Info("Removing owner from team", "team", teamName, "owner", owner.Name, "kind", owner.Kind)

	team, err := s.Get(ctx, teamName)
	if err != nil {
		return err
	}

	// Find and remove owner
	found := false
	for i, o := range team.Owners {
		if o.Kind == owner.Kind && o.Name == owner.Name {
			team.Owners = append(team.Owners[:i], team.Owners[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("owner not found: %s (%s)", owner.Name, owner.Kind)
	}

	return s.Update(ctx, teamName, team)
}

// SetSuspended sets the suspended status of a team
func (s *TenantService) SetSuspended(ctx context.Context, name string, suspended bool) error {
	logger.Info("Setting tenant suspended status", "name", name, "suspended", suspended)

	tenant, err := s.k8sClient.GetTenant(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Update annotation
	annotations := tenant.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	if suspended {
		annotations["bison.io/suspended"] = "true"
	} else {
		delete(annotations, "bison.io/suspended")
	}
	tenant.SetAnnotations(annotations)

	if err := s.k8sClient.UpdateTenant(ctx, tenant); err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// tenantToTeam converts a Capsule Tenant to a Team
func (s *TenantService) tenantToTeam(tenant *unstructured.Unstructured) (*Team, error) {
	team := &Team{
		Name:           tenant.GetName(),
		Quota:          make(map[string]string),
		Owners:         []OwnerRef{},
		Mode:           TeamModeShared, // Default to shared
		ExclusiveNodes: []string{},
	}

	// Get annotations for display name, description, mode, and suspended status
	annotations := tenant.GetAnnotations()
	if annotations != nil {
		team.DisplayName = annotations["bison.io/display-name"]
		team.Description = annotations["bison.io/description"]
		team.Suspended = annotations["bison.io/suspended"] == "true"

		// Parse mode
		if mode := annotations["bison.io/mode"]; mode == string(TeamModeExclusive) {
			team.Mode = TeamModeExclusive
		}

		// Parse exclusive nodes (comma-separated)
		if nodes := annotations["bison.io/exclusive-nodes"]; nodes != "" {
			team.ExclusiveNodes = splitNodes(nodes)
		}
	}
	if team.DisplayName == "" {
		team.DisplayName = team.Name
	}

	// Parse spec
	spec, _, _ := unstructured.NestedMap(tenant.Object, "spec")
	if spec != nil {
		// Get owners with kind
		owners, _, _ := unstructured.NestedSlice(spec, "owners")
		for _, owner := range owners {
			if o, ok := owner.(map[string]interface{}); ok {
				ownerRef := OwnerRef{}
				if kind, ok := o["kind"].(string); ok {
					ownerRef.Kind = kind
				} else {
					ownerRef.Kind = "Group" // Default to Group for backward compatibility
				}
				if name, ok := o["name"].(string); ok {
					ownerRef.Name = name
				}
				if ownerRef.Name != "" {
					team.Owners = append(team.Owners, ownerRef)
				}
			}
		}

		// Get nodeSelector
		nodeSelector, _, _ := unstructured.NestedStringMap(spec, "nodeSelector")
		team.NodeSelector = nodeSelector

		// Get quota from resourceQuotas (dynamic)
		resourceQuotas, _, _ := unstructured.NestedMap(spec, "resourceQuotas")
		if resourceQuotas != nil {
			items, _, _ := unstructured.NestedSlice(resourceQuotas, "items")
			if len(items) > 0 {
				if item, ok := items[0].(map[string]interface{}); ok {
					hard, _, _ := unstructured.NestedStringMap(item, "hard")
					if hard != nil {
						// Copy all quota values (dynamic)
						for k, v := range hard {
							// Convert K8s resource names to simpler names for frontend
							key := simplifyResourceName(k)
							team.Quota[key] = v
						}
					}
				}
			}
		}
	}

	// Parse status
	status, _, _ := unstructured.NestedMap(tenant.Object, "status")
	if status != nil {
		namespaces, _, _ := unstructured.NestedInt64(status, "size")
		team.ProjectCount = int(namespaces)
		team.Status.Namespaces = int(namespaces)

		state, _, _ := unstructured.NestedString(status, "state")
		team.Status.State = state
		team.Status.Ready = state == "Active"
	}

	return team, nil
}

// teamToTenant converts a Team to a Capsule Tenant
func (s *TenantService) teamToTenant(team *Team) *unstructured.Unstructured {
	// Build owners list with kind
	owners := make([]interface{}, len(team.Owners))
	for i, owner := range team.Owners {
		kind := owner.Kind
		if kind == "" {
			kind = "Group" // Default to Group
		}
		owners[i] = map[string]interface{}{
			"kind": kind,
			"name": owner.Name,
		}
	}

	// Build resource quota (dynamic)
	hardQuota := map[string]interface{}{}
	for k, v := range team.Quota {
		if v != "" {
			// Convert simple names back to K8s resource names
			key := expandResourceName(k)
			hardQuota[key] = v
		}
	}

	spec := map[string]interface{}{
		"owners": owners,
	}

	// Set nodeSelector based on mode
	nodeSelector := map[string]interface{}{}
	if team.Mode == TeamModeExclusive {
		// Exclusive mode: use team- prefix to avoid conflicts with reserved names
		nodeSelector[LabelPoolKey] = GetExclusivePoolLabel(team.Name)
	} else {
		// Shared mode: use shared pool
		nodeSelector[LabelPoolKey] = LabelPoolShared
	}
	spec["nodeSelector"] = nodeSelector

	// Add resourceQuotas only for shared mode (exclusive mode uses node physical resources as limit)
	if team.Mode == TeamModeShared && len(hardQuota) > 0 {
		spec["resourceQuotas"] = map[string]interface{}{
			"scope": "Tenant",
			"items": []interface{}{
				map[string]interface{}{
					"hard": hardQuota,
				},
			},
		}
	}

	annotations := map[string]interface{}{
		"bison.io/display-name": team.DisplayName,
		"bison.io/description":  team.Description,
		"bison.io/mode":         string(team.Mode),
	}
	if team.Suspended {
		annotations["bison.io/suspended"] = "true"
	}
	if len(team.ExclusiveNodes) > 0 {
		annotations["bison.io/exclusive-nodes"] = joinNodes(team.ExclusiveNodes)
	}

	tenant := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "capsule.clastix.io/v1beta2",
			"kind":       "Tenant",
			"metadata": map[string]interface{}{
				"name":        team.Name,
				"annotations": annotations,
				"labels": map[string]interface{}{
					"bison.io/managed": "true",
				},
			},
			"spec": spec,
		},
	}

	return tenant
}

// simplifyResourceName converts K8s resource names to simpler names
// e.g., "requests.cpu" -> "cpu", "requests.nvidia.com/gpu" -> "nvidia.com/gpu"
func simplifyResourceName(name string) string {
	// Remove "requests." or "limits." prefix
	if len(name) > 9 && name[:9] == "requests." {
		return name[9:]
	}
	if len(name) > 7 && name[:7] == "limits." {
		return name[7:]
	}
	return name
}

// expandResourceName converts simple names back to K8s resource names
// e.g., "cpu" -> "requests.cpu", "nvidia.com/gpu" -> "requests.nvidia.com/gpu"
func expandResourceName(name string) string {
	// Standard resources that need "requests." prefix
	standardResources := map[string]bool{
		"cpu":    true,
		"memory": true,
	}

	if standardResources[name] {
		return "requests." + name
	}

	// For pods, use directly
	if name == "pods" {
		return "pods"
	}

	// For extended resources (like GPU), add "requests." prefix
	return "requests." + name
}

// getExclusiveNodeResources calculates total resources from exclusive nodes
func (s *TenantService) getExclusiveNodeResources(ctx context.Context, nodeNames []string) map[string]string {
	result := make(map[string]string)
	resourceTotals := make(map[string]float64)

	for _, nodeName := range nodeNames {
		node, err := s.k8sClient.GetNode(ctx, nodeName)
		if err != nil {
			logger.Warn("Failed to get node for resource calculation", "node", nodeName, "error", err)
			continue
		}

		// Use Allocatable (available for pods) instead of Capacity
		for resourceName, quantity := range node.Status.Allocatable {
			key := string(resourceName)
			resourceTotals[key] += quantity.AsApproximateFloat64()
		}
	}

	// Convert to string map with appropriate formatting
	for k, v := range resourceTotals {
		// Format based on resource type
		if k == "memory" || strings.HasSuffix(k, "-storage") || k == "ephemeral-storage" {
			// Memory/storage: convert to Gi for readability
			result[k] = fmt.Sprintf("%.0fGi", v/(1024*1024*1024))
		} else if k == "cpu" {
			// CPU: keep as float for millicores
			result[k] = fmt.Sprintf("%.0f", v)
		} else {
			// Other resources (GPU, etc.): integer
			result[k] = fmt.Sprintf("%.0f", v)
		}
	}

	return result
}

// getTeamResourceUsage aggregates resource usage from all pods under a team
func (s *TenantService) getTeamResourceUsage(ctx context.Context, teamName string) map[string]string {
	result := make(map[string]string)
	resourceUsed := make(map[string]float64)

	// List all namespaces for this team
	labelSelector := fmt.Sprintf("capsule.clastix.io/tenant=%s,bison.io/managed=true", teamName)
	namespaces, err := s.k8sClient.ListNamespaces(ctx, labelSelector)
	if err != nil {
		logger.Warn("Failed to list namespaces for resource usage", "team", teamName, "error", err)
		return result
	}

	// Aggregate resource requests from all running pods
	for _, ns := range namespaces.Items {
		pods, err := s.k8sClient.ListPods(ctx, ns.Name, "")
		if err != nil {
			logger.Warn("Failed to list pods", "namespace", ns.Name, "error", err)
			continue
		}

		for _, pod := range pods.Items {
			// Only count running pods
			if pod.Status.Phase != "Running" {
				continue
			}

			for _, container := range pod.Spec.Containers {
				for resourceName, quantity := range container.Resources.Requests {
					key := string(resourceName)
					resourceUsed[key] += quantity.AsApproximateFloat64()
				}
			}
		}
	}

	// Convert to string map with appropriate formatting
	for k, v := range resourceUsed {
		if k == "memory" || strings.HasSuffix(k, "-storage") || k == "ephemeral-storage" {
			// Memory/storage: convert to Gi for readability
			result[k] = fmt.Sprintf("%.0fGi", v/(1024*1024*1024))
		} else if k == "cpu" {
			// CPU: keep as float (cores)
			result[k] = fmt.Sprintf("%.1f", v)
		} else {
			// Other resources (GPU, etc.): integer
			result[k] = fmt.Sprintf("%.0f", v)
		}
	}

	return result
}

// splitNodes splits a comma-separated string of node names
func splitNodes(nodes string) []string {
	if nodes == "" {
		return nil
	}
	parts := strings.Split(nodes, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// joinNodes joins node names into a comma-separated string
func joinNodes(nodes []string) string {
	return strings.Join(nodes, ",")
}

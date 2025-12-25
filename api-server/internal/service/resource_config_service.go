package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

const (
	ResourceConfigNamespace = "bison-system"
	ResourceConfigName      = "bison-resource-config"
	ResourceConfigDataKey   = "resources"
)

// ResourceCategory represents resource categories
type ResourceCategory string

const (
	CategoryCompute     ResourceCategory = "compute"
	CategoryMemory      ResourceCategory = "memory"
	CategoryStorage     ResourceCategory = "storage"
	CategoryAccelerator ResourceCategory = "accelerator"
	CategoryOther       ResourceCategory = "other"
)

// ResourceDefinition represents a configured resource
type ResourceDefinition struct {
	Name        string           `json:"name"`        // K8s resource name: cpu, memory, nvidia.com/gpu
	DisplayName string           `json:"displayName"` // Display name: CPU, 内存, NVIDIA GPU
	Unit        string           `json:"unit"`        // Display unit: 核, GiB, 卡
	Divisor     float64          `json:"divisor"`     // Unit divisor: displayValue = rawValue / divisor
	Category    ResourceCategory `json:"category"`    // Category: compute, memory, storage, accelerator, other
	Enabled     bool             `json:"enabled"`     // Whether to show this resource
	SortOrder   int              `json:"sortOrder"`   // Sort order (lower = first)
	ShowInQuota bool             `json:"showInQuota"` // Whether to show in quota settings
	Price       float64          `json:"price"`       // Price per unit per hour
}

// DiscoveredResource represents a resource discovered from cluster
type DiscoveredResource struct {
	Name        string  `json:"name"`
	Capacity    float64 `json:"capacity"`
	Allocatable float64 `json:"allocatable"`
	Configured  bool    `json:"configured"` // Whether this resource has been configured
}

// ResourceConfigService manages resource configurations
type ResourceConfigService struct {
	k8sClient *k8s.Client
}

// NewResourceConfigService creates a new ResourceConfigService
func NewResourceConfigService(k8sClient *k8s.Client) *ResourceConfigService {
	return &ResourceConfigService{
		k8sClient: k8sClient,
	}
}

// DiscoverClusterResources discovers all resources available in the cluster
func (s *ResourceConfigService) DiscoverClusterResources(ctx context.Context) ([]DiscoveredResource, error) {
	logger.Debug("Discovering cluster resources")

	nodes, err := s.k8sClient.ListNodes(ctx)
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
		return nil, err
	}

	// Get existing configurations to mark configured resources
	configs, _ := s.GetResourceConfigs(ctx)
	configMap := make(map[string]bool)
	for _, cfg := range configs {
		configMap[cfg.Name] = true
	}

	// Aggregate resources from all nodes
	resourceMap := make(map[string]*DiscoveredResource)

	for _, node := range nodes.Items {
		// Process capacity
		for name, quantity := range node.Status.Capacity {
			resourceName := string(name)
			value := quantity.AsApproximateFloat64()

			if dr, exists := resourceMap[resourceName]; exists {
				dr.Capacity += value
			} else {
				resourceMap[resourceName] = &DiscoveredResource{
					Name:       resourceName,
					Capacity:   value,
					Configured: configMap[resourceName],
				}
			}
		}

		// Process allocatable
		for name, quantity := range node.Status.Allocatable {
			resourceName := string(name)
			value := quantity.AsApproximateFloat64()

			if dr, exists := resourceMap[resourceName]; exists {
				dr.Allocatable += value
			}
		}
	}

	// Convert to slice
	var resources []DiscoveredResource
	for _, dr := range resourceMap {
		resources = append(resources, *dr)
	}

	// Sort by name
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	return resources, nil
}

// GetResourceConfigs returns all configured resources
func (s *ResourceConfigService) GetResourceConfigs(ctx context.Context) ([]ResourceDefinition, error) {
	logger.Info("Getting resource configs from ConfigMap",
		"namespace", ResourceConfigNamespace,
		"name", ResourceConfigName)

	cm, err := s.k8sClient.GetConfigMap(ctx, ResourceConfigNamespace, ResourceConfigName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty list if not found - no default configs
			logger.Info("ConfigMap not found, returning empty list")
			return []ResourceDefinition{}, nil
		}
		logger.Error("Failed to get resource config", "error", err)
		return nil, err
	}

	logger.Info("ConfigMap found", "dataKeys", len(cm.Data))

	data, ok := cm.Data[ResourceConfigDataKey]
	if !ok {
		logger.Info("No resource data key in ConfigMap")
		return []ResourceDefinition{}, nil
	}

	logger.Debug("ConfigMap data", "data", data)

	var configs []ResourceDefinition
	if err := json.Unmarshal([]byte(data), &configs); err != nil {
		logger.Error("Failed to parse resource config", "error", err)
		return nil, err
	}

	logger.Info("Loaded resource configs", "count", len(configs))

	// Sort by sortOrder
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].SortOrder < configs[j].SortOrder
	})

	return configs, nil
}

// GetEnabledResourceConfigs returns only enabled resources
func (s *ResourceConfigService) GetEnabledResourceConfigs(ctx context.Context) ([]ResourceDefinition, error) {
	configs, err := s.GetResourceConfigs(ctx)
	if err != nil {
		return nil, err
	}

	var enabled []ResourceDefinition
	for _, cfg := range configs {
		if cfg.Enabled {
			enabled = append(enabled, cfg)
		}
	}

	return enabled, nil
}

// GetQuotaResourceConfigs returns resources that should be shown in quota settings
func (s *ResourceConfigService) GetQuotaResourceConfigs(ctx context.Context) ([]ResourceDefinition, error) {
	configs, err := s.GetResourceConfigs(ctx)
	if err != nil {
		return nil, err
	}

	var quotaResources []ResourceDefinition
	for _, cfg := range configs {
		if cfg.Enabled && cfg.ShowInQuota {
			quotaResources = append(quotaResources, cfg)
		}
	}

	return quotaResources, nil
}

// SaveResourceConfigs saves all resource configurations
func (s *ResourceConfigService) SaveResourceConfigs(ctx context.Context, configs []ResourceDefinition) error {
	logger.Info("Saving resource configs", "count", len(configs))

	// Ensure namespace exists
	if err := s.ensureNamespace(ctx); err != nil {
		logger.Error("Failed to ensure namespace", "namespace", ResourceConfigNamespace, "error", err)
		return fmt.Errorf("failed to ensure namespace %s: %w", ResourceConfigNamespace, err)
	}

	data, err := json.Marshal(configs)
	if err != nil {
		logger.Error("Failed to marshal configs", "error", err)
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	logger.Debug("Marshaled config data", "data", string(data))

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceConfigName,
			Namespace: ResourceConfigNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "bison",
				"app.kubernetes.io/component": "resource-config",
			},
		},
		Data: map[string]string{
			ResourceConfigDataKey: string(data),
		},
	}

	// Try to update, create if not exists
	existing, err := s.k8sClient.GetConfigMap(ctx, ResourceConfigNamespace, ResourceConfigName)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating new resource config ConfigMap")
			if createErr := s.k8sClient.CreateConfigMap(ctx, ResourceConfigNamespace, cm); createErr != nil {
				logger.Error("Failed to create ConfigMap", "error", createErr)
				return fmt.Errorf("failed to create ConfigMap: %w", createErr)
			}
			logger.Info("Resource config ConfigMap created successfully")
			return nil
		}
		logger.Error("Failed to get existing ConfigMap", "error", err)
		return fmt.Errorf("failed to get existing ConfigMap: %w", err)
	}

	existing.Data = cm.Data
	if updateErr := s.k8sClient.UpdateConfigMap(ctx, ResourceConfigNamespace, existing); updateErr != nil {
		logger.Error("Failed to update ConfigMap", "error", updateErr)
		return fmt.Errorf("failed to update ConfigMap: %w", updateErr)
	}
	logger.Info("Resource config ConfigMap updated successfully")

	// Verify the save was successful
	verifyConfigMap, verifyErr := s.k8sClient.GetConfigMap(ctx, ResourceConfigNamespace, ResourceConfigName)
	if verifyErr != nil {
		logger.Error("Failed to verify ConfigMap after save", "error", verifyErr)
	} else {
		logger.Info("Verified ConfigMap after save",
			"hasData", verifyConfigMap.Data != nil,
			"dataLength", len(verifyConfigMap.Data[ResourceConfigDataKey]))
	}

	return nil
}

// UpdateResourceConfig updates a single resource configuration
func (s *ResourceConfigService) UpdateResourceConfig(ctx context.Context, name string, updated ResourceDefinition) error {
	logger.Info("Updating resource config", "name", name)

	configs, err := s.GetResourceConfigs(ctx)
	if err != nil {
		return err
	}

	found := false
	for i, cfg := range configs {
		if cfg.Name == name {
			configs[i] = updated
			found = true
			break
		}
	}

	if !found {
		configs = append(configs, updated)
	}

	return s.SaveResourceConfigs(ctx, configs)
}

// GetResourceConfig returns a single resource configuration
func (s *ResourceConfigService) GetResourceConfig(ctx context.Context, name string) (*ResourceDefinition, error) {
	configs, err := s.GetResourceConfigs(ctx)
	if err != nil {
		return nil, err
	}

	for _, cfg := range configs {
		if cfg.Name == name {
			return &cfg, nil
		}
	}

	return nil, fmt.Errorf("resource config not found: %s", name)
}

// ensureNamespace ensures the bison-system namespace exists
func (s *ResourceConfigService) ensureNamespace(ctx context.Context) error {
	_, err := s.k8sClient.GetNamespace(ctx, ResourceConfigNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Creating namespace", "namespace", ResourceConfigNamespace)
			labels := map[string]string{
				"app.kubernetes.io/name": "bison",
			}
			if createErr := s.k8sClient.CreateNamespace(ctx, ResourceConfigNamespace, labels); createErr != nil {
				return createErr
			}
			logger.Info("Namespace created", "namespace", ResourceConfigNamespace)
			return nil
		}
		return err
	}
	return nil
}

// GetResourceDisplayName returns display name for a resource
func (s *ResourceConfigService) GetResourceDisplayName(ctx context.Context, name string) string {
	cfg, err := s.GetResourceConfig(ctx, name)
	if err == nil && cfg != nil {
		return cfg.DisplayName
	}
	// Return raw name if not configured
	return name
}

// GetResourceUnit returns unit for a resource
func (s *ResourceConfigService) GetResourceUnit(ctx context.Context, name string) string {
	cfg, err := s.GetResourceConfig(ctx, name)
	if err == nil && cfg != nil {
		return cfg.Unit
	}
	// Return empty if not configured
	return ""
}

// GetResourceDivisor returns divisor for a resource (for unit conversion)
func (s *ResourceConfigService) GetResourceDivisor(ctx context.Context, name string) float64 {
	cfg, err := s.GetResourceConfig(ctx, name)
	if err == nil && cfg != nil && cfg.Divisor > 0 {
		return cfg.Divisor
	}
	// Return 1 (no conversion) if not configured
	return 1
}

// GetResourcePrice returns price for a resource
func (s *ResourceConfigService) GetResourcePrice(ctx context.Context, name string) float64 {
	cfg, err := s.GetResourceConfig(ctx, name)
	if err == nil && cfg != nil {
		return cfg.Price
	}
	return 0
}

// ConvertValue applies divisor to convert raw value to display value
func (s *ResourceConfigService) ConvertValue(ctx context.Context, name string, rawValue float64) float64 {
	divisor := s.GetResourceDivisor(ctx, name)
	if divisor <= 0 {
		divisor = 1
	}
	return rawValue / divisor
}

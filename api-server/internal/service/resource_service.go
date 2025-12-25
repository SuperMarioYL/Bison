package service

import (
	"context"
	"sort"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

// ResourceType represents a cluster resource type
type ResourceType struct {
	Name        string  `json:"name"`        // K8s resource name
	DisplayName string  `json:"displayName"` // Configured display name
	Unit        string  `json:"unit"`        // Configured unit
	Capacity    float64 `json:"capacity"`    // Total cluster capacity (converted)
	Allocatable float64 `json:"allocatable"` // Total allocatable (converted)
}

// ResourceService handles cluster resource operations
type ResourceService struct {
	k8sClient         *k8s.Client
	resourceConfigSvc *ResourceConfigService
}

// NewResourceService creates a new ResourceService
func NewResourceService(k8sClient *k8s.Client, resourceConfigSvc *ResourceConfigService) *ResourceService {
	return &ResourceService{
		k8sClient:         k8sClient,
		resourceConfigSvc: resourceConfigSvc,
	}
}

// GetClusterResources returns only configured and enabled resources
func (s *ResourceService) GetClusterResources(ctx context.Context) ([]ResourceType, error) {
	logger.Debug("Getting cluster resources")

	// Get enabled resource configs
	configs, err := s.resourceConfigSvc.GetEnabledResourceConfigs(ctx)
	if err != nil {
		logger.Error("Failed to get resource configs", "error", err)
		return nil, err
	}

	// If no configs, return empty list
	if len(configs) == 0 {
		logger.Debug("No resource configs found, returning empty list")
		return []ResourceType{}, nil
	}

	// Build a set of configured resource names
	configMap := make(map[string]ResourceDefinition)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg
	}

	// Get nodes to aggregate resources
	nodes, err := s.k8sClient.ListNodes(ctx)
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
		return nil, err
	}

	// Aggregate resources from all nodes (raw values)
	rawCapacity := make(map[string]float64)
	rawAllocatable := make(map[string]float64)

	for _, node := range nodes.Items {
		// Process capacity
		for name, quantity := range node.Status.Capacity {
			resourceName := string(name)
			// Only process configured resources
			if _, ok := configMap[resourceName]; ok {
				rawCapacity[resourceName] += quantity.AsApproximateFloat64()
			}
		}

		// Process allocatable
		for name, quantity := range node.Status.Allocatable {
			resourceName := string(name)
			// Only process configured resources
			if _, ok := configMap[resourceName]; ok {
				rawAllocatable[resourceName] += quantity.AsApproximateFloat64()
			}
		}
	}

	// Build result with converted values
	var resources []ResourceType
	for _, cfg := range configs {
		divisor := cfg.Divisor
		if divisor <= 0 {
			divisor = 1
		}

		rt := ResourceType{
			Name:        cfg.Name,
			DisplayName: cfg.DisplayName,
			Unit:        cfg.Unit,
			Capacity:    rawCapacity[cfg.Name] / divisor,
			Allocatable: rawAllocatable[cfg.Name] / divisor,
		}
		resources = append(resources, rt)
	}

	// Sort by sortOrder from config
	sort.Slice(resources, func(i, j int) bool {
		cfgI := configMap[resources[i].Name]
		cfgJ := configMap[resources[j].Name]
		return cfgI.SortOrder < cfgJ.SortOrder
	})

	return resources, nil
}

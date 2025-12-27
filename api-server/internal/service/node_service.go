package service

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

const (
	// Label keys
	LabelPoolKey = "bison.io/pool"
	// Label values
	LabelPoolShared = "shared"
	// Taint keys
	TaintDisabledKey = "bison.io/disabled"
)

// NodeStatus represents the Bison management status of a node
type NodeStatus string

const (
	NodeStatusUnmanaged NodeStatus = "unmanaged" // Not managed by Bison
	NodeStatusDisabled  NodeStatus = "disabled"  // Managed but disabled (tainted)
	NodeStatusShared    NodeStatus = "shared"    // In shared pool
	NodeStatusExclusive NodeStatus = "exclusive" // Exclusively assigned to a team
)

// NodeInfo represents detailed node information with Bison status
type NodeInfo struct {
	Name           string            `json:"name"`
	Status         NodeStatus        `json:"status"`
	Team           string            `json:"team,omitempty"` // Team name if exclusive
	Labels         map[string]string `json:"labels"`
	Taints         []corev1.Taint    `json:"taints"`
	Conditions     []NodeCondition   `json:"conditions"`
	Capacity       map[string]string `json:"capacity"`
	Allocatable    map[string]string `json:"allocatable"`
	Architecture   string            `json:"architecture"`
	OS             string            `json:"os"`
	KernelVersion  string            `json:"kernelVersion"`
	Runtime        string            `json:"runtime"`
	KubeletVersion string            `json:"kubeletVersion"`
	InternalIP     string            `json:"internalIP"`
	Hostname       string            `json:"hostname"`
	PodCount       int               `json:"podCount"`
	CreationTime   string            `json:"creationTime"`
}

// NodeCondition represents a node condition
type NodeCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// NodeService handles node management operations
type NodeService struct {
	k8sClient *k8s.Client
}

// NewNodeService creates a new NodeService
func NewNodeService(k8sClient *k8s.Client) *NodeService {
	return &NodeService{
		k8sClient: k8sClient,
	}
}

// ListNodes returns all nodes with their Bison status
func (s *NodeService) ListNodes(ctx context.Context) ([]NodeInfo, error) {
	logger.Debug("Listing nodes with Bison status")

	nodes, err := s.k8sClient.ListNodes(ctx)
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
		return nil, err
	}

	var nodeInfos []NodeInfo
	for _, node := range nodes.Items {
		info := s.nodeToNodeInfo(&node)

		// Get pod count
		pods, err := s.k8sClient.ListPodsOnNode(ctx, node.Name)
		if err == nil {
			info.PodCount = len(pods.Items)
		}

		nodeInfos = append(nodeInfos, info)
	}

	return nodeInfos, nil
}

// GetNode returns detailed information about a node
func (s *NodeService) GetNode(ctx context.Context, name string) (*NodeInfo, error) {
	logger.Debug("Getting node info", "name", name)

	node, err := s.k8sClient.GetNode(ctx, name)
	if err != nil {
		logger.Error("Failed to get node", "name", name, "error", err)
		return nil, err
	}

	info := s.nodeToNodeInfo(node)

	// Get pod count
	pods, err := s.k8sClient.ListPodsOnNode(ctx, name)
	if err == nil {
		info.PodCount = len(pods.Items)
	}

	return &info, nil
}

// EnableNode enables a node for Bison management (adds to shared pool)
func (s *NodeService) EnableNode(ctx context.Context, name string) error {
	logger.Info("Enabling node", "name", name)

	// Remove disabled taint if exists
	if err := s.k8sClient.RemoveNodeTaintByKey(ctx, name, TaintDisabledKey); err != nil {
		logger.Error("Failed to remove disabled taint", "node", name, "error", err)
		return fmt.Errorf("failed to remove disabled taint: %w", err)
	}

	// Add shared pool label
	if err := s.k8sClient.AddNodeLabel(ctx, name, LabelPoolKey, LabelPoolShared); err != nil {
		logger.Error("Failed to add shared label", "node", name, "error", err)
		return fmt.Errorf("failed to add shared label: %w", err)
	}

	logger.Info("Node enabled successfully", "name", name)
	return nil
}

// DisableNode disables a node from Bison management (adds NoSchedule taint)
func (s *NodeService) DisableNode(ctx context.Context, name string) error {
	logger.Info("Disabling node", "name", name)

	// Check if node is exclusively assigned
	node, err := s.k8sClient.GetNode(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if pool, ok := node.Labels[LabelPoolKey]; ok && pool != LabelPoolShared && pool != "" {
		return fmt.Errorf("cannot disable node: node is exclusively assigned to team '%s'", pool)
	}

	// Remove pool label
	if err := s.k8sClient.RemoveNodeLabel(ctx, name, LabelPoolKey); err != nil {
		logger.Error("Failed to remove pool label", "node", name, "error", err)
		return fmt.Errorf("failed to remove pool label: %w", err)
	}

	// Add disabled taint
	taint := corev1.Taint{
		Key:    TaintDisabledKey,
		Value:  "true",
		Effect: corev1.TaintEffectNoSchedule,
	}
	if err := s.k8sClient.AddNodeTaint(ctx, name, taint); err != nil {
		logger.Error("Failed to add disabled taint", "node", name, "error", err)
		return fmt.Errorf("failed to add disabled taint: %w", err)
	}

	logger.Info("Node disabled successfully", "name", name)
	return nil
}

// AssignNodeToTeam exclusively assigns a node to a team
func (s *NodeService) AssignNodeToTeam(ctx context.Context, nodeName, teamName string) error {
	logger.Info("Assigning node to team", "node", nodeName, "team", teamName)

	// Check current status
	node, err := s.k8sClient.GetNode(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Check if node is disabled
	for _, taint := range node.Spec.Taints {
		if taint.Key == TaintDisabledKey {
			return fmt.Errorf("cannot assign disabled node to team")
		}
	}

	// Get the exclusive pool label for this team
	exclusiveLabel := GetExclusivePoolLabel(teamName)

	// Check if already assigned to another team
	if pool, ok := node.Labels[LabelPoolKey]; ok && pool != LabelPoolShared && pool != "" && pool != exclusiveLabel {
		existingTeam := ParseExclusivePoolLabel(pool)
		if existingTeam == "" {
			existingTeam = pool
		}
		return fmt.Errorf("node is already assigned to team '%s'", existingTeam)
	}

	// Update label to team-<teamName>
	if err := s.k8sClient.AddNodeLabel(ctx, nodeName, LabelPoolKey, exclusiveLabel); err != nil {
		logger.Error("Failed to assign node to team", "node", nodeName, "team", teamName, "error", err)
		return fmt.Errorf("failed to assign node: %w", err)
	}

	logger.Info("Node assigned to team successfully", "node", nodeName, "team", teamName)
	return nil
}

// ReleaseNodeFromTeam releases a node from exclusive assignment back to shared pool
func (s *NodeService) ReleaseNodeFromTeam(ctx context.Context, nodeName string) error {
	logger.Info("Releasing node from team", "node", nodeName)

	// Check current status
	node, err := s.k8sClient.GetNode(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	pool, ok := node.Labels[LabelPoolKey]
	if !ok || pool == "" || pool == LabelPoolShared {
		return fmt.Errorf("node is not exclusively assigned to any team")
	}

	// Verify it's an exclusive node (has team- prefix)
	teamName := ParseExclusivePoolLabel(pool)
	if teamName == "" {
		return fmt.Errorf("node has unknown pool label: %s", pool)
	}

	// Update label back to shared
	if err := s.k8sClient.AddNodeLabel(ctx, nodeName, LabelPoolKey, LabelPoolShared); err != nil {
		logger.Error("Failed to release node", "node", nodeName, "error", err)
		return fmt.Errorf("failed to release node: %w", err)
	}

	logger.Info("Node released successfully", "node", nodeName, "previousTeam", teamName)
	return nil
}

// GetSharedNodes returns all nodes in the shared pool
func (s *NodeService) GetSharedNodes(ctx context.Context) ([]NodeInfo, error) {
	logger.Debug("Getting shared nodes")

	nodes, err := s.k8sClient.ListNodesWithLabel(ctx, LabelPoolKey+"="+LabelPoolShared)
	if err != nil {
		return nil, err
	}

	var nodeInfos []NodeInfo
	for _, node := range nodes.Items {
		nodeInfos = append(nodeInfos, s.nodeToNodeInfo(&node))
	}

	return nodeInfos, nil
}

// GetTeamNodes returns all nodes exclusively assigned to a team
func (s *NodeService) GetTeamNodes(ctx context.Context, teamName string) ([]NodeInfo, error) {
	logger.Debug("Getting team nodes", "team", teamName)

	// Use team- prefix for label selector
	exclusiveLabel := GetExclusivePoolLabel(teamName)
	nodes, err := s.k8sClient.ListNodesWithLabel(ctx, LabelPoolKey+"="+exclusiveLabel)
	if err != nil {
		return nil, err
	}

	var nodeInfos []NodeInfo
	for _, node := range nodes.Items {
		nodeInfos = append(nodeInfos, s.nodeToNodeInfo(&node))
	}

	return nodeInfos, nil
}

// GetAvailableNodesForExclusive returns nodes that can be assigned to a team (shared nodes)
func (s *NodeService) GetAvailableNodesForExclusive(ctx context.Context) ([]NodeInfo, error) {
	return s.GetSharedNodes(ctx)
}

// nodeToNodeInfo converts a k8s Node to NodeInfo
func (s *NodeService) nodeToNodeInfo(node *corev1.Node) NodeInfo {
	info := NodeInfo{
		Name:         node.Name,
		Labels:       node.Labels,
		Taints:       node.Spec.Taints,
		Capacity:     make(map[string]string),
		Allocatable:  make(map[string]string),
		CreationTime: node.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}

	// Determine Bison status
	info.Status = s.getNodeStatus(node)
	if info.Status == NodeStatusExclusive {
		// Extract team name from pool label (remove "team-" prefix)
		poolValue := node.Labels[LabelPoolKey]
		info.Team = ParseExclusivePoolLabel(poolValue)
	}

	// Extract node info from status
	for _, addr := range node.Status.Addresses {
		switch addr.Type {
		case corev1.NodeInternalIP:
			info.InternalIP = addr.Address
		case corev1.NodeHostName:
			info.Hostname = addr.Address
		}
	}

	// Node info
	info.Architecture = node.Status.NodeInfo.Architecture
	info.OS = node.Status.NodeInfo.OSImage
	info.KernelVersion = node.Status.NodeInfo.KernelVersion
	info.Runtime = node.Status.NodeInfo.ContainerRuntimeVersion
	info.KubeletVersion = node.Status.NodeInfo.KubeletVersion

	// Capacity and allocatable
	for name, quantity := range node.Status.Capacity {
		info.Capacity[string(name)] = quantity.String()
	}
	for name, quantity := range node.Status.Allocatable {
		info.Allocatable[string(name)] = quantity.String()
	}

	// Conditions
	for _, cond := range node.Status.Conditions {
		info.Conditions = append(info.Conditions, NodeCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	return info
}

// getNodeStatus determines the Bison management status of a node
func (s *NodeService) getNodeStatus(node *corev1.Node) NodeStatus {
	// Check if disabled (has disabled taint)
	for _, taint := range node.Spec.Taints {
		if taint.Key == TaintDisabledKey {
			return NodeStatusDisabled
		}
	}

	// Check pool label
	pool, ok := node.Labels[LabelPoolKey]
	if !ok || pool == "" {
		return NodeStatusUnmanaged
	}

	if pool == LabelPoolShared {
		return NodeStatusShared
	}

	// Exclusive nodes have "team-" prefix
	if ParseExclusivePoolLabel(pool) != "" {
		return NodeStatusExclusive
	}

	// Unknown label value - treat as unmanaged
	return NodeStatusUnmanaged
}

// GetNodeStatusSummary returns a summary of node statuses
func (s *NodeService) GetNodeStatusSummary(ctx context.Context) (map[NodeStatus]int, error) {
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	summary := make(map[NodeStatus]int)
	for _, node := range nodes {
		summary[node.Status]++
	}

	return summary, nil
}

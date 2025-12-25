package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

// ClusterHandler handles cluster-related API requests
type ClusterHandler struct {
	k8sClient *k8s.Client
}

// NewClusterHandler creates a new ClusterHandler
func NewClusterHandler(k8sClient *k8s.Client) *ClusterHandler {
	return &ClusterHandler{
		k8sClient: k8sClient,
	}
}

// NodeResource represents a node's resource
type NodeResource struct {
	Name        string `json:"name"`
	Capacity    int64  `json:"capacity"`
	Allocatable int64  `json:"allocatable"`
}

// ClusterNode represents a node in the cluster
type ClusterNode struct {
	Name      string            `json:"name"`
	Arch      string            `json:"arch"`
	OS        string            `json:"os"`
	Ready     bool              `json:"ready"`
	Labels    map[string]string `json:"labels"`
	Resources []NodeResource    `json:"resources"`
}

// NodeDetail represents detailed node information
type NodeDetail struct {
	Name       string            `json:"name"`
	Arch       string            `json:"arch"`
	OS         string            `json:"os"`
	Ready      bool              `json:"ready"`
	Labels     map[string]string `json:"labels"`
	Taints     []NodeTaint       `json:"taints"`
	NodeInfo   NodeInfo          `json:"nodeInfo"`
	Addresses  []NodeAddress     `json:"addresses"`
	Resources  []NodeResource    `json:"resources"`
	Conditions []NodeCondition   `json:"conditions"`
}

// NodeTaint represents a node taint
type NodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

// NodeInfo represents node system information
type NodeInfo struct {
	KernelVersion           string `json:"kernelVersion"`
	OSImage                 string `json:"osImage"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	KubeletVersion          string `json:"kubeletVersion"`
	Architecture            string `json:"architecture"`
	OperatingSystem         string `json:"operatingSystem"`
}

// NodeAddress represents a node address
type NodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// NodeCondition represents a node condition
type NodeCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// NodePod represents a pod on a node
type NodePod struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Status        string `json:"status"`
	IP            string `json:"ip"`
	CPURequest    int64  `json:"cpuRequest"`
	MemoryRequest int64  `json:"memoryRequest"`
	Restarts      int32  `json:"restarts"`
}

// ListNodes returns all nodes in the cluster
func (h *ClusterHandler) ListNodes(c *gin.Context) {
	ctx := c.Request.Context()
	arch := c.Query("arch")

	nodes, err := h.k8sClient.ListNodes(ctx)
	if err != nil {
		logger.Error("Failed to list nodes", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []ClusterNode
	for _, node := range nodes.Items {
		nodeArch := node.Status.NodeInfo.Architecture
		if arch != "" && nodeArch != arch {
			continue
		}

		cn := ClusterNode{
			Name:      node.Name,
			Arch:      nodeArch,
			OS:        node.Status.NodeInfo.OperatingSystem,
			Ready:     isNodeReady(&node),
			Labels:    node.Labels,
			Resources: getNodeResources(&node),
		}
		result = append(result, cn)
	}

	c.JSON(http.StatusOK, gin.H{"items": result})
}

// GetNode returns detailed information about a specific node
func (h *ClusterHandler) GetNode(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	node, err := h.k8sClient.GetNode(ctx, name)
	if err != nil {
		logger.Error("Failed to get node", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	detail := NodeDetail{
		Name:       node.Name,
		Arch:       node.Status.NodeInfo.Architecture,
		OS:         node.Status.NodeInfo.OperatingSystem,
		Ready:      isNodeReady(node),
		Labels:     node.Labels,
		Taints:     getTaints(node),
		NodeInfo:   getNodeInfo(node),
		Addresses:  getAddresses(node),
		Resources:  getNodeResources(node),
		Conditions: getConditions(node),
	}

	c.JSON(http.StatusOK, detail)
}

// GetNodePods returns all pods running on a specific node
func (h *ClusterHandler) GetNodePods(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	pods, err := h.k8sClient.ListPodsOnNode(ctx, name)
	if err != nil {
		logger.Error("Failed to list pods on node", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []NodePod
	for _, pod := range pods.Items {
		np := NodePod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			IP:        pod.Status.PodIP,
		}

		// Calculate resource requests
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					np.CPURequest += cpu.MilliValue()
				}
				if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					np.MemoryRequest += mem.Value()
				}
			}
		}

		// Count restarts
		for _, cs := range pod.Status.ContainerStatuses {
			np.Restarts += cs.RestartCount
		}

		result = append(result, np)
	}

	c.JSON(http.StatusOK, gin.H{"items": result})
}

// UpdateNodeLabels updates labels on a node
func (h *ClusterHandler) UpdateNodeLabels(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	var req struct {
		Labels map[string]string `json:"labels" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.k8sClient.UpdateNodeLabels(ctx, name, req.Labels); err != nil {
		logger.Error("Failed to update node labels", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Labels updated successfully"})
}

// UpdateNodeTaints updates taints on a node
func (h *ClusterHandler) UpdateNodeTaints(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	var req struct {
		Taints []NodeTaint `json:"taints" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var taints []corev1.Taint
	for _, t := range req.Taints {
		taints = append(taints, corev1.Taint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: corev1.TaintEffect(t.Effect),
		})
	}

	if err := h.k8sClient.UpdateNodeTaints(ctx, name, taints); err != nil {
		logger.Error("Failed to update node taints", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Taints updated successfully"})
}

// Helper functions

func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getNodeResources(node *corev1.Node) []NodeResource {
	resources := []NodeResource{}
	
	for name, capacity := range node.Status.Capacity {
		allocatable := node.Status.Allocatable[name]
		resources = append(resources, NodeResource{
			Name:        string(name),
			Capacity:    capacity.Value(),
			Allocatable: allocatable.Value(),
		})
	}
	
	return resources
}

func getTaints(node *corev1.Node) []NodeTaint {
	var taints []NodeTaint
	for _, t := range node.Spec.Taints {
		taints = append(taints, NodeTaint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: string(t.Effect),
		})
	}
	return taints
}

func getNodeInfo(node *corev1.Node) NodeInfo {
	return NodeInfo{
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		OSImage:                 node.Status.NodeInfo.OSImage,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		Architecture:            node.Status.NodeInfo.Architecture,
		OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
	}
}

func getAddresses(node *corev1.Node) []NodeAddress {
	var addresses []NodeAddress
	for _, addr := range node.Status.Addresses {
		addresses = append(addresses, NodeAddress{
			Type:    string(addr.Type),
			Address: addr.Address,
		})
	}
	return addresses
}

func getConditions(node *corev1.Node) []NodeCondition {
	var conditions []NodeCondition
	for _, cond := range node.Status.Conditions {
		conditions = append(conditions, NodeCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}
	return conditions
}


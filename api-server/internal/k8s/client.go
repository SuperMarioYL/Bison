package k8s

import (
	"context"
	"io"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/bison/api-server/pkg/logger"
)

// Client wraps Kubernetes client operations
type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewClient creates a new Kubernetes client
func NewClient() (*Client, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first (for production)
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig (for local development)
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Error("Failed to get k8s config", "error", err)
			return nil, err
		}
		logger.Info("Using kubeconfig", "path", kubeconfig)
	} else {
		logger.Info("Using in-cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error("Failed to create clientset", "error", err)
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Error("Failed to create dynamic client", "error", err)
		return nil, err
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}, nil
}

// Namespace operations

func (c *Client) CreateNamespace(ctx context.Context, name string, labels map[string]string) error {
	logger.Debug("K8s: Creating namespace", "name", name)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to create namespace", "name", name, "error", err)
	}
	return err
}

func (c *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) ListNamespaces(ctx context.Context, labelSelector string) (*corev1.NamespaceList, error) {
	return c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	logger.Debug("K8s: Deleting namespace", "name", name)
	return c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) UpdateNamespaceLabels(ctx context.Context, name string, labels map[string]string) error {
	logger.Debug("K8s: Updating namespace labels", "name", name)

	ns, err := c.GetNamespace(ctx, name)
	if err != nil {
		return err
	}
	ns.Labels = labels
	_, err = c.clientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	return err
}

func (c *Client) UpdateNamespace(ctx context.Context, ns *corev1.Namespace) error {
	logger.Debug("K8s: Updating namespace", "name", ns.Name)
	_, err := c.clientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	return err
}

// Job operations

func (c *Client) ListJobs(ctx context.Context, namespace, labelSelector string) (*batchv1.JobList, error) {
	if namespace == "" {
		return c.clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	}
	return c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func (c *Client) GetJob(ctx context.Context, namespace, name string) (*batchv1.Job, error) {
	return c.clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) DeleteJob(ctx context.Context, namespace, name string) error {
	logger.Debug("K8s: Deleting job", "namespace", namespace, "name", name)
	propagationPolicy := metav1.DeletePropagationBackground
	return c.clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
}

// CronJob operations

func (c *Client) ListCronJobs(ctx context.Context, namespace string) (*batchv1.CronJobList, error) {
	return c.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
}

func (c *Client) GetCronJob(ctx context.Context, namespace, name string) (*batchv1.CronJob, error) {
	return c.clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

// Pod operations

func (c *Client) ListPods(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	if namespace == "" {
		return c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	}
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	logger.Debug("K8s: Deleting pod", "namespace", namespace, "name", name)
	return c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		logger.Debug("K8s: Failed to get pod logs stream", "namespace", namespace, "name", name, "error", err)
		return "", err
	}
	defer stream.Close()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

// Node operations

func (c *Client) ListNodes(ctx context.Context) (*corev1.NodeList, error) {
	return c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

func (c *Client) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	return c.clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) UpdateNodeLabels(ctx context.Context, name string, labels map[string]string) error {
	logger.Debug("K8s: Updating node labels", "node", name)

	node, err := c.GetNode(ctx, name)
	if err != nil {
		return err
	}
	node.Labels = labels
	_, err = c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to update node labels", "node", name, "error", err)
	}
	return err
}

func (c *Client) UpdateNodeTaints(ctx context.Context, name string, taints []corev1.Taint) error {
	logger.Debug("K8s: Updating node taints", "node", name)

	node, err := c.GetNode(ctx, name)
	if err != nil {
		return err
	}
	node.Spec.Taints = taints
	_, err = c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to update node taints", "node", name, "error", err)
	}
	return err
}

func (c *Client) ListPodsOnNode(ctx context.Context, nodeName string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
}

// UpdateNode updates the entire node object
func (c *Client) UpdateNode(ctx context.Context, node *corev1.Node) error {
	logger.Debug("K8s: Updating node", "node", node.Name)
	_, err := c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to update node", "node", node.Name, "error", err)
	}
	return err
}

// AddNodeLabel adds or updates a label on a node
func (c *Client) AddNodeLabel(ctx context.Context, nodeName, key, value string) error {
	logger.Debug("K8s: Adding node label", "node", nodeName, "key", key, "value", value)

	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	node.Labels[key] = value

	return c.UpdateNode(ctx, node)
}

// RemoveNodeLabel removes a label from a node
func (c *Client) RemoveNodeLabel(ctx context.Context, nodeName, key string) error {
	logger.Debug("K8s: Removing node label", "node", nodeName, "key", key)

	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	if node.Labels != nil {
		delete(node.Labels, key)
	}

	return c.UpdateNode(ctx, node)
}

// AddNodeTaint adds a taint to a node
func (c *Client) AddNodeTaint(ctx context.Context, nodeName string, taint corev1.Taint) error {
	logger.Debug("K8s: Adding node taint", "node", nodeName, "key", taint.Key, "effect", taint.Effect)

	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	// Check if taint already exists
	for i, t := range node.Spec.Taints {
		if t.Key == taint.Key && t.Effect == taint.Effect {
			// Update existing taint
			node.Spec.Taints[i] = taint
			return c.UpdateNode(ctx, node)
		}
	}

	// Add new taint
	node.Spec.Taints = append(node.Spec.Taints, taint)
	return c.UpdateNode(ctx, node)
}

// RemoveNodeTaint removes a taint from a node by key and effect
func (c *Client) RemoveNodeTaint(ctx context.Context, nodeName, key string, effect corev1.TaintEffect) error {
	logger.Debug("K8s: Removing node taint", "node", nodeName, "key", key, "effect", effect)

	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	var newTaints []corev1.Taint
	for _, t := range node.Spec.Taints {
		if t.Key != key || t.Effect != effect {
			newTaints = append(newTaints, t)
		}
	}
	node.Spec.Taints = newTaints

	return c.UpdateNode(ctx, node)
}

// RemoveNodeTaintByKey removes all taints with the given key from a node
func (c *Client) RemoveNodeTaintByKey(ctx context.Context, nodeName, key string) error {
	logger.Debug("K8s: Removing all node taints by key", "node", nodeName, "key", key)

	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	var newTaints []corev1.Taint
	for _, t := range node.Spec.Taints {
		if t.Key != key {
			newTaints = append(newTaints, t)
		}
	}
	node.Spec.Taints = newTaints

	return c.UpdateNode(ctx, node)
}

// ListNodesWithLabel returns nodes that have a specific label
func (c *Client) ListNodesWithLabel(ctx context.Context, labelSelector string) (*corev1.NodeList, error) {
	return c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// RBAC operations

func (c *Client) CreateRole(ctx context.Context, namespace, name string, rules []rbacv1.PolicyRule) error {
	logger.Debug("K8s: Creating role", "namespace", namespace, "name", name)

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}
	_, err := c.clientset.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to create role", "namespace", namespace, "name", name, "error", err)
	}
	return err
}

func (c *Client) CreateRoleBinding(ctx context.Context, namespace, name, roleName string, subjects []rbacv1.Subject) error {
	logger.Debug("K8s: Creating role binding", "namespace", namespace, "name", name, "role", roleName)

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: subjects,
	}
	_, err := c.clientset.RbacV1().RoleBindings(namespace).Create(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to create role binding", "namespace", namespace, "name", name, "error", err)
	}
	return err
}

func (c *Client) CreateClusterRoleBinding(ctx context.Context, name, clusterRoleName string, subjects []rbacv1.Subject) error {
	logger.Debug("K8s: Creating cluster role binding", "name", name)

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: subjects,
	}
	_, err := c.clientset.RbacV1().ClusterRoleBindings().Create(ctx, binding, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Debug("K8s: Failed to create cluster role binding", "name", name, "error", err)
	}
	return err
}

func (c *Client) DeleteRoleBinding(ctx context.Context, namespace, name string) error {
	logger.Debug("K8s: Deleting role binding", "namespace", namespace, "name", name)
	err := c.clientset.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *Client) GetRoleBinding(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error) {
	return c.clientset.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) ListRoleBindings(ctx context.Context, namespace string) (*rbacv1.RoleBindingList, error) {
	return c.clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
}

func (c *Client) UpdateRoleBinding(ctx context.Context, namespace string, binding *rbacv1.RoleBinding) error {
	logger.Debug("K8s: Updating role binding", "namespace", namespace, "name", binding.Name)
	_, err := c.clientset.RbacV1().RoleBindings(namespace).Update(ctx, binding, metav1.UpdateOptions{})
	return err
}

// CreateOrUpdateRoleBinding creates or updates a RoleBinding
func (c *Client) CreateOrUpdateRoleBinding(ctx context.Context, namespace, name, roleName string, subjects []rbacv1.Subject) error {
	existing, err := c.GetRoleBinding(ctx, namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreateRoleBinding(ctx, namespace, name, roleName, subjects)
		}
		return err
	}

	existing.Subjects = subjects
	existing.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     roleName,
	}
	return c.UpdateRoleBinding(ctx, namespace, existing)
}

// Capsule Tenant operations

var tenantGVR = schema.GroupVersionResource{
	Group:    "capsule.clastix.io",
	Version:  "v1beta2",
	Resource: "tenants",
}

func (c *Client) ListTenants(ctx context.Context) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(tenantGVR).List(ctx, metav1.ListOptions{})
}

func (c *Client) GetTenant(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(tenantGVR).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) CreateTenant(ctx context.Context, tenant *unstructured.Unstructured) error {
	logger.Debug("K8s: Creating Capsule Tenant", "name", tenant.GetName())
	_, err := c.dynamicClient.Resource(tenantGVR).Create(ctx, tenant, metav1.CreateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to create Tenant", "name", tenant.GetName(), "error", err)
	}
	return err
}

func (c *Client) UpdateTenant(ctx context.Context, tenant *unstructured.Unstructured) error {
	logger.Debug("K8s: Updating Capsule Tenant", "name", tenant.GetName())
	_, err := c.dynamicClient.Resource(tenantGVR).Update(ctx, tenant, metav1.UpdateOptions{})
	if err != nil {
		logger.Debug("K8s: Failed to update Tenant", "name", tenant.GetName(), "error", err)
	}
	return err
}

func (c *Client) DeleteTenant(ctx context.Context, name string) error {
	logger.Debug("K8s: Deleting Capsule Tenant", "name", name)
	return c.dynamicClient.Resource(tenantGVR).Delete(ctx, name, metav1.DeleteOptions{})
}

// ResourceQuota operations

func (c *Client) CreateResourceQuota(ctx context.Context, namespace string, quota *corev1.ResourceQuota) error {
	logger.Debug("K8s: Creating ResourceQuota", "namespace", namespace, "name", quota.Name)
	_, err := c.clientset.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{})
	return err
}

func (c *Client) GetResourceQuota(ctx context.Context, namespace, name string) (*corev1.ResourceQuota, error) {
	return c.clientset.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) UpdateResourceQuota(ctx context.Context, namespace string, quota *corev1.ResourceQuota) error {
	logger.Debug("K8s: Updating ResourceQuota", "namespace", namespace, "name", quota.Name)
	_, err := c.clientset.CoreV1().ResourceQuotas(namespace).Update(ctx, quota, metav1.UpdateOptions{})
	return err
}

func (c *Client) DeleteResourceQuota(ctx context.Context, namespace, name string) error {
	logger.Debug("K8s: Deleting ResourceQuota", "namespace", namespace, "name", name)
	return c.clientset.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) ListResourceQuotas(ctx context.Context, namespace string) (*corev1.ResourceQuotaList, error) {
	return c.clientset.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
}

// Helper function to check if resource exists
func (c *Client) NamespaceExists(ctx context.Context, name string) bool {
	_, err := c.GetNamespace(ctx, name)
	return err == nil || !errors.IsNotFound(err)
}

func (c *Client) TenantExists(ctx context.Context, name string) bool {
	_, err := c.GetTenant(ctx, name)
	return err == nil || !errors.IsNotFound(err)
}

// ConfigMap operations

func (c *Client) GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) CreateConfigMap(ctx context.Context, namespace string, cm *corev1.ConfigMap) error {
	logger.Debug("K8s: Creating ConfigMap", "namespace", namespace, "name", cm.Name)
	_, err := c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func (c *Client) UpdateConfigMap(ctx context.Context, namespace string, cm *corev1.ConfigMap) error {
	logger.Debug("K8s: Updating ConfigMap", "namespace", namespace, "name", cm.Name)
	_, err := c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func (c *Client) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	logger.Debug("K8s: Deleting ConfigMap", "namespace", namespace, "name", name)
	return c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Deployment operations (for suspend/resume)

func (c *Client) ListDeployments(ctx context.Context, namespace string) (*appsv1.DeploymentList, error) {
	return c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
}

func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) UpdateDeployment(ctx context.Context, namespace string, deployment *appsv1.Deployment) error {
	logger.Debug("K8s: Updating Deployment", "namespace", namespace, "name", deployment.Name)
	_, err := c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

// StatefulSet operations (for suspend/resume)

func (c *Client) ListStatefulSets(ctx context.Context, namespace string) (*appsv1.StatefulSetList, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
}

func (c *Client) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) UpdateStatefulSet(ctx context.Context, namespace string, statefulSet *appsv1.StatefulSet) error {
	logger.Debug("K8s: Updating StatefulSet", "namespace", namespace, "name", statefulSet.Name)
	_, err := c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

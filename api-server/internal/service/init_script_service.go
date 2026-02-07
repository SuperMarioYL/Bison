package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

const (
	InitScriptsConfigMap        = "bison-init-scripts"
	ControlPlaneConfigConfigMap = "bison-control-plane-config"
)

// ScriptPhase represents when a script should be executed
type ScriptPhase string

const (
	PhasePreJoin  ScriptPhase = "pre-join"
	PhasePostJoin ScriptPhase = "post-join"
)

// Script represents a platform-specific script implementation
type Script struct {
	ID      string `json:"id"`
	OS      string `json:"os"`      // "ubuntu", "centos", "debian", "*" (wildcard)
	Arch    string `json:"arch"`    // "amd64", "arm64", "*" (wildcard)
	Content string `json:"content"` // Shell script content
}

// ScriptGroup represents a group of scripts for a specific functionality
type ScriptGroup struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Phase       ScriptPhase `json:"phase"`
	Enabled     bool        `json:"enabled"`
	Order       int         `json:"order"`
	Builtin     bool        `json:"builtin"`
	Scripts     []Script    `json:"scripts"`
}

// InitScriptsConfig holds all script groups
type InitScriptsConfig struct {
	Groups []ScriptGroup `json:"groups"`
}

// NodePlatform represents the detected platform of a node
type NodePlatform struct {
	OS      string `json:"os"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

// ControlPlaneConfig holds the control plane SSH configuration
type ControlPlaneConfig struct {
	Host       string `json:"host"`
	SSHPort    int    `json:"sshPort"`
	SSHUser    string `json:"sshUser"`
	AuthMethod string `json:"authMethod"` // "password" or "privateKey"
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

// InitScriptService handles initialization script operations
type InitScriptService struct {
	k8sClient *k8s.Client
}

// NewInitScriptService creates a new InitScriptService
func NewInitScriptService(k8sClient *k8s.Client) *InitScriptService {
	return &InitScriptService{
		k8sClient: k8sClient,
	}
}

// GetAllScriptGroups returns all script groups
func (s *InitScriptService) GetAllScriptGroups(ctx context.Context) ([]ScriptGroup, error) {
	logger.Debug("Getting all script groups")

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Sort by order
	sort.Slice(config.Groups, func(i, j int) bool {
		return config.Groups[i].Order < config.Groups[j].Order
	})

	return config.Groups, nil
}

// GetScriptGroup returns a specific script group by ID
func (s *InitScriptService) GetScriptGroup(ctx context.Context, id string) (*ScriptGroup, error) {
	logger.Debug("Getting script group", "id", id)

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return nil, err
	}

	for _, group := range config.Groups {
		if group.ID == id {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("script group not found: %s", id)
}

// CreateScriptGroup creates a new script group
func (s *InitScriptService) CreateScriptGroup(ctx context.Context, group *ScriptGroup) error {
	logger.Info("Creating script group", "name", group.Name)

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return err
	}

	// Generate ID if not provided
	if group.ID == "" {
		group.ID = fmt.Sprintf("custom-%d", time.Now().UnixNano())
	}

	// Check for duplicate ID
	for _, existing := range config.Groups {
		if existing.ID == group.ID {
			return fmt.Errorf("script group with ID %s already exists", group.ID)
		}
	}

	// Set order to last
	if group.Order == 0 {
		maxOrder := 0
		for _, g := range config.Groups {
			if g.Order > maxOrder {
				maxOrder = g.Order
			}
		}
		group.Order = maxOrder + 1
	}

	// Custom scripts are not builtin
	group.Builtin = false

	config.Groups = append(config.Groups, *group)

	return s.saveInitScriptsConfig(ctx, config)
}

// UpdateScriptGroup updates an existing script group
func (s *InitScriptService) UpdateScriptGroup(ctx context.Context, id string, group *ScriptGroup) error {
	logger.Info("Updating script group", "id", id)

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return err
	}

	found := false
	for i, existing := range config.Groups {
		if existing.ID == id {
			// Preserve builtin status and ID
			group.ID = id
			group.Builtin = existing.Builtin
			config.Groups[i] = *group
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("script group not found: %s", id)
	}

	return s.saveInitScriptsConfig(ctx, config)
}

// DeleteScriptGroup deletes a script group (only custom scripts can be deleted)
func (s *InitScriptService) DeleteScriptGroup(ctx context.Context, id string) error {
	logger.Info("Deleting script group", "id", id)

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return err
	}

	newGroups := make([]ScriptGroup, 0, len(config.Groups))
	deleted := false

	for _, group := range config.Groups {
		if group.ID == id {
			if group.Builtin {
				return fmt.Errorf("cannot delete builtin script group: %s", id)
			}
			deleted = true
			continue
		}
		newGroups = append(newGroups, group)
	}

	if !deleted {
		return fmt.Errorf("script group not found: %s", id)
	}

	config.Groups = newGroups
	return s.saveInitScriptsConfig(ctx, config)
}

// ToggleScriptGroup enables or disables a script group
func (s *InitScriptService) ToggleScriptGroup(ctx context.Context, id string, enabled bool) error {
	logger.Info("Toggling script group", "id", id, "enabled", enabled)

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return err
	}

	found := false
	for i, group := range config.Groups {
		if group.ID == id {
			config.Groups[i].Enabled = enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("script group not found: %s", id)
	}

	return s.saveInitScriptsConfig(ctx, config)
}

// ReorderScriptGroups updates the order of script groups
func (s *InitScriptService) ReorderScriptGroups(ctx context.Context, ids []string) error {
	logger.Info("Reordering script groups", "ids", ids)

	config, err := s.getInitScriptsConfig(ctx)
	if err != nil {
		return err
	}

	// Create a map of current groups
	groupMap := make(map[string]*ScriptGroup)
	for i := range config.Groups {
		groupMap[config.Groups[i].ID] = &config.Groups[i]
	}

	// Update orders based on the provided order
	for i, id := range ids {
		if group, ok := groupMap[id]; ok {
			group.Order = i + 1
		}
	}

	return s.saveInitScriptsConfig(ctx, config)
}

// GetMatchingScript returns the best matching script for a given platform
func (s *InitScriptService) GetMatchingScript(group *ScriptGroup, platform NodePlatform) *Script {
	if len(group.Scripts) == 0 {
		return nil
	}

	// Priority: exact match > OS match with wildcard arch > wildcard OS with arch match > all wildcards
	var exactMatch, osMatch, archMatch, wildcardMatch *Script

	for i := range group.Scripts {
		script := &group.Scripts[i]
		osMatches := script.OS == platform.OS || script.OS == "*"
		archMatches := script.Arch == platform.Arch || script.Arch == "*"

		if !osMatches || !archMatches {
			continue
		}

		if script.OS == platform.OS && script.Arch == platform.Arch {
			exactMatch = script
			break // Best match found
		} else if script.OS == platform.OS && script.Arch == "*" {
			osMatch = script
		} else if script.OS == "*" && script.Arch == platform.Arch {
			archMatch = script
		} else if script.OS == "*" && script.Arch == "*" {
			wildcardMatch = script
		}
	}

	// Return by priority
	if exactMatch != nil {
		return exactMatch
	}
	if osMatch != nil {
		return osMatch
	}
	if archMatch != nil {
		return archMatch
	}
	return wildcardMatch
}

// GetScriptsForPhase returns all enabled scripts for a specific phase, matched to the platform
func (s *InitScriptService) GetScriptsForPhase(ctx context.Context, phase ScriptPhase, platform NodePlatform) ([]struct {
	Group  ScriptGroup
	Script Script
}, error) {
	groups, err := s.GetAllScriptGroups(ctx)
	if err != nil {
		return nil, err
	}

	var result []struct {
		Group  ScriptGroup
		Script Script
	}

	for _, group := range groups {
		if group.Phase != phase || !group.Enabled {
			continue
		}

		script := s.GetMatchingScript(&group, platform)
		if script != nil {
			result = append(result, struct {
				Group  ScriptGroup
				Script Script
			}{
				Group:  group,
				Script: *script,
			})
		}
	}

	return result, nil
}

// GetControlPlaneConfig returns the control plane SSH configuration
func (s *InitScriptService) GetControlPlaneConfig(ctx context.Context) (*ControlPlaneConfig, error) {
	logger.Debug("Getting control plane config")

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, ControlPlaneConfigConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return &ControlPlaneConfig{
				SSHPort: 22,
				SSHUser: "root",
			}, nil
		}
		return nil, fmt.Errorf("failed to get control plane config: %w", err)
	}

	data, ok := cm.Data["config"]
	if !ok {
		return &ControlPlaneConfig{
			SSHPort: 22,
			SSHUser: "root",
		}, nil
	}

	var config ControlPlaneConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse control plane config: %w", err)
	}

	return &config, nil
}

// SaveControlPlaneConfig saves the control plane SSH configuration
func (s *InitScriptService) SaveControlPlaneConfig(ctx context.Context, config *ControlPlaneConfig) error {
	logger.Info("Saving control plane config", "host", config.Host)

	// Set defaults
	if config.SSHPort == 0 {
		config.SSHPort = 22
	}
	if config.SSHUser == "" {
		config.SSHUser = "root"
	}

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal control plane config: %w", err)
	}

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, ControlPlaneConfigConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ControlPlaneConfigConfigMap,
					Namespace: BisonNamespace,
				},
				Data: map[string]string{
					"config": string(data),
				},
			}
			return s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm)
		}
		return fmt.Errorf("failed to get control plane config: %w", err)
	}

	// Update existing ConfigMap
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config"] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

// SaveAllScriptGroups replaces all script groups at once (used by config import)
func (s *InitScriptService) SaveAllScriptGroups(ctx context.Context, groups []ScriptGroup) error {
	logger.Info("Saving all script groups", "count", len(groups))
	config := &InitScriptsConfig{Groups: groups}
	return s.saveInitScriptsConfig(ctx, config)
}

// getInitScriptsConfig returns the init scripts configuration, initializing with defaults if not found
func (s *InitScriptService) getInitScriptsConfig(ctx context.Context) (*InitScriptsConfig, error) {
	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, InitScriptsConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			// Initialize with default builtin scripts
			config := s.getDefaultInitScriptsConfig()
			if err := s.saveInitScriptsConfig(ctx, config); err != nil {
				return nil, err
			}
			return config, nil
		}
		return nil, fmt.Errorf("failed to get init scripts config: %w", err)
	}

	data, ok := cm.Data["config"]
	if !ok {
		config := s.getDefaultInitScriptsConfig()
		if err := s.saveInitScriptsConfig(ctx, config); err != nil {
			return nil, err
		}
		return config, nil
	}

	var config InitScriptsConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse init scripts config: %w", err)
	}

	return &config, nil
}

// saveInitScriptsConfig saves the init scripts configuration
func (s *InitScriptService) saveInitScriptsConfig(ctx context.Context, config *InitScriptsConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal init scripts config: %w", err)
	}

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, InitScriptsConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      InitScriptsConfigMap,
					Namespace: BisonNamespace,
				},
				Data: map[string]string{
					"config": string(data),
				},
			}
			return s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm)
		}
		return fmt.Errorf("failed to get init scripts config: %w", err)
	}

	// Update existing ConfigMap
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data["config"] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

// getDefaultInitScriptsConfig returns the default builtin script groups
func (s *InitScriptService) getDefaultInitScriptsConfig() *InitScriptsConfig {
	return &InitScriptsConfig{
		Groups: []ScriptGroup{
			{
				ID:          "disable-swap",
				Name:        "禁用 Swap",
				Description: "禁用 Swap 分区（Kubernetes 要求）",
				Phase:       PhasePreJoin,
				Enabled:     true,
				Order:       1,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "disable-swap-universal",
						OS:   "*",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Disabling swap..."
swapoff -a || true
sed -i '/swap/d' /etc/fstab || true
echo "Swap disabled successfully"
`,
					},
				},
			},
			{
				ID:          "configure-kernel",
				Name:        "配置内核参数",
				Description: "配置 Kubernetes 所需的内核参数",
				Phase:       PhasePreJoin,
				Enabled:     true,
				Order:       2,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "configure-kernel-universal",
						OS:   "*",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Configuring kernel parameters..."

# Load required modules
modprobe br_netfilter || true
modprobe overlay || true

# Ensure modules load on boot
cat > /etc/modules-load.d/k8s.conf << EOF
br_netfilter
overlay
EOF

# Configure sysctl
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF

sysctl --system
echo "Kernel parameters configured successfully"
`,
					},
				},
			},
			{
				ID:          "disable-firewall",
				Name:        "禁用防火墙",
				Description: "禁用节点防火墙（firewalld/ufw）",
				Phase:       PhasePreJoin,
				Enabled:     false,
				Order:       3,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "disable-firewall-debian",
						OS:   "ubuntu",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Disabling firewall..."
if command -v ufw &> /dev/null; then
    ufw disable || true
fi
echo "Firewall disabled successfully"
`,
					},
					{
						ID:   "disable-firewall-debian2",
						OS:   "debian",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Disabling firewall..."
if command -v ufw &> /dev/null; then
    ufw disable || true
fi
echo "Firewall disabled successfully"
`,
					},
					{
						ID:   "disable-firewall-rhel",
						OS:   "centos",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Disabling firewall..."
if systemctl is-active --quiet firewalld 2>/dev/null; then
    systemctl stop firewalld
    systemctl disable firewalld
fi
echo "Firewall disabled successfully"
`,
					},
					{
						ID:   "disable-firewall-rhel2",
						OS:   "rhel",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Disabling firewall..."
if systemctl is-active --quiet firewalld 2>/dev/null; then
    systemctl stop firewalld
    systemctl disable firewalld
fi
echo "Firewall disabled successfully"
`,
					},
					{
						ID:   "disable-firewall-openeuler",
						OS:   "openEuler",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Disabling firewall..."
if systemctl is-active --quiet firewalld 2>/dev/null; then
    systemctl stop firewalld
    systemctl disable firewalld
fi
echo "Firewall disabled successfully"
`,
					},
				},
			},
			{
				ID:          "configure-selinux",
				Name:        "配置 SELinux",
				Description: "设置 SELinux 为 Permissive 模式（仅 RHEL/CentOS/openEuler）",
				Phase:       PhasePreJoin,
				Enabled:     false,
				Order:       4,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "configure-selinux-centos",
						OS:   "centos",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Configuring SELinux to permissive mode..."
if command -v setenforce &> /dev/null; then
    setenforce 0 || true
    if [ -f /etc/selinux/config ]; then
        sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
    fi
fi
echo "SELinux configured successfully"
`,
					},
					{
						ID:   "configure-selinux-rhel",
						OS:   "rhel",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Configuring SELinux to permissive mode..."
if command -v setenforce &> /dev/null; then
    setenforce 0 || true
    if [ -f /etc/selinux/config ]; then
        sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
    fi
fi
echo "SELinux configured successfully"
`,
					},
					{
						ID:   "configure-selinux-openeuler",
						OS:   "openEuler",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Configuring SELinux to permissive mode..."
if command -v setenforce &> /dev/null; then
    setenforce 0 || true
    if [ -f /etc/selinux/config ]; then
        sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
    fi
fi
echo "SELinux configured successfully"
`,
					},
				},
			},
			{
				ID:          "configure-timezone",
				Name:        "配置时区和 NTP",
				Description: "设置系统时区并启用 NTP 时间同步",
				Phase:       PhasePreJoin,
				Enabled:     false,
				Order:       5,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "configure-timezone-universal",
						OS:   "*",
						Arch: "*",
						Content: `#!/bin/bash
set -e
TIMEZONE="${TIMEZONE:-Asia/Shanghai}"

echo "Configuring timezone to $TIMEZONE..."
timedatectl set-timezone $TIMEZONE || true

echo "Enabling and starting NTP service..."
if systemctl list-unit-files | grep -q chronyd; then
    systemctl enable chronyd || true
    systemctl start chronyd || true
elif systemctl list-unit-files | grep -q ntpd; then
    systemctl enable ntpd || true
    systemctl start ntpd || true
elif systemctl list-unit-files | grep -q systemd-timesyncd; then
    systemctl enable systemd-timesyncd || true
    systemctl start systemd-timesyncd || true
fi

echo "Timezone and NTP configured successfully"
`,
					},
				},
			},
			{
				ID:          "configure-registry",
				Name:        "配置私有镜像仓库",
				Description: "配置 containerd 使用私有镜像仓库（支持 HTTP）",
				Phase:       PhasePreJoin,
				Enabled:     false,
				Order:       6,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "configure-registry-ubuntu",
						OS:   "ubuntu",
						Arch: "*",
						Content: `#!/bin/bash
set -e
REGISTRY_URL="${REGISTRY_URL:-registry.example.com:5000}"

echo "Configuring private registry: $REGISTRY_URL"

# Create registry config directory
mkdir -p /etc/containerd/certs.d/${REGISTRY_URL}

# Configure registry mirror
cat > /etc/containerd/certs.d/${REGISTRY_URL}/hosts.toml << EOF
server = "http://${REGISTRY_URL}"

[host."http://${REGISTRY_URL}"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# Restart containerd
systemctl restart containerd
echo "Private registry configured successfully"
`,
					},
					{
						ID:   "configure-registry-debian",
						OS:   "debian",
						Arch: "*",
						Content: `#!/bin/bash
set -e
REGISTRY_URL="${REGISTRY_URL:-registry.example.com:5000}"

echo "Configuring private registry: $REGISTRY_URL"

# Create registry config directory
mkdir -p /etc/containerd/certs.d/${REGISTRY_URL}

# Configure registry mirror
cat > /etc/containerd/certs.d/${REGISTRY_URL}/hosts.toml << EOF
server = "http://${REGISTRY_URL}"

[host."http://${REGISTRY_URL}"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# Restart containerd
systemctl restart containerd
echo "Private registry configured successfully"
`,
					},
					{
						ID:   "configure-registry-centos",
						OS:   "centos",
						Arch: "*",
						Content: `#!/bin/bash
set -e
REGISTRY_URL="${REGISTRY_URL:-registry.example.com:5000}"

echo "Configuring private registry: $REGISTRY_URL"

# Create registry config directory
mkdir -p /etc/containerd/certs.d/${REGISTRY_URL}

# Configure registry mirror
cat > /etc/containerd/certs.d/${REGISTRY_URL}/hosts.toml << EOF
server = "http://${REGISTRY_URL}"

[host."http://${REGISTRY_URL}"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# Restart containerd
systemctl restart containerd
echo "Private registry configured successfully"
`,
					},
					{
						ID:   "configure-registry-rhel",
						OS:   "rhel",
						Arch: "*",
						Content: `#!/bin/bash
set -e
REGISTRY_URL="${REGISTRY_URL:-registry.example.com:5000}"

echo "Configuring private registry: $REGISTRY_URL"

# Create registry config directory
mkdir -p /etc/containerd/certs.d/${REGISTRY_URL}

# Configure registry mirror
cat > /etc/containerd/certs.d/${REGISTRY_URL}/hosts.toml << EOF
server = "http://${REGISTRY_URL}"

[host."http://${REGISTRY_URL}"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# Restart containerd
systemctl restart containerd
echo "Private registry configured successfully"
`,
					},
					{
						ID:   "configure-registry-openeuler",
						OS:   "openEuler",
						Arch: "*",
						Content: `#!/bin/bash
set -e
REGISTRY_URL="${REGISTRY_URL:-registry.example.com:5000}"

echo "Configuring private registry: $REGISTRY_URL"

# Create registry config directory
mkdir -p /etc/containerd/certs.d/${REGISTRY_URL}

# Configure registry mirror
cat > /etc/containerd/certs.d/${REGISTRY_URL}/hosts.toml << EOF
server = "http://${REGISTRY_URL}"

[host."http://${REGISTRY_URL}"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# Restart containerd
systemctl restart containerd
echo "Private registry configured successfully"
`,
					},
				},
			},
			{
				ID:          "add-node-labels",
				Name:        "添加节点标签",
				Description: "为节点添加 Worker 角色标签",
				Phase:       PhasePostJoin,
				Enabled:     false,
				Order:       7,
				Builtin:     true,
				Scripts: []Script{
					{
						ID:   "add-node-labels-universal",
						OS:   "*",
						Arch: "*",
						Content: `#!/bin/bash
set -e
echo "Adding worker label to node ${NODE_NAME}..."

# Wait for node to be registered
sleep 5

# Add worker role label
kubectl label node ${NODE_NAME} node-role.kubernetes.io/worker= --overwrite || true

echo "Node label added successfully"
`,
					},
				},
			},
		},
	}
}

// ReplaceVariables replaces variables in the script content
func ReplaceVariables(content string, vars map[string]string) string {
	result := content
	for key, value := range vars {
		placeholder := "${" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

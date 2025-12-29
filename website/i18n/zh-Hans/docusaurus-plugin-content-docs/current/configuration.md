---
sidebar_position: 6
---

# 配置

本指南介绍如何根据您的特定环境和需求配置 Bison。

## Helm Chart 配置

Bison 主要通过 Helm values 进行配置。您可以通过提供 `values.yaml` 文件或使用 `--set` 参数来自定义安装。

### 关键配置参数

| 参数 | 描述 | 默认值 | 示例 |
|-----------|-------------|---------|---------|
| `auth.enabled` | 启用认证 | `false` | `true` |
| `auth.admin.username` | 管理员用户名 | `admin` | `admin` |
| `auth.admin.password` | 管理员密码 | `admin` | `changeme` |
| `apiServer.replicaCount` | API Server 副本数 | `2` | `3` |
| `apiServer.image.repository` | API Server 镜像 | `ghcr.io/supermarioyl/bison/api-server` | - |
| `apiServer.image.tag` | API Server 镜像标签 | `0.0.1` | `latest` |
| `webUI.replicaCount` | Web UI 副本数 | `2` | `3` |
| `webUI.image.repository` | Web UI 镜像 | `ghcr.io/supermarioyl/bison/web-ui` | - |
| `webUI.image.tag` | Web UI 镜像标签 | `0.0.1` | `latest` |
| `opencost.url` | OpenCost API 端点 | `http://opencost.opencost-system.svc:9003` | 自定义 URL |

### 自定义 Values 示例

创建一个 `custom-values.yaml` 文件：

```yaml
# 认证
auth:
  enabled: true
  admin:
    username: admin
    password: MySecurePassword123

# API Server
apiServer:
  replicaCount: 3
  image:
    tag: 0.0.1
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi

# Web UI
webUI:
  replicaCount: 3
  image:
    tag: 0.0.1
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi

# OpenCost 集成
opencost:
  url: http://opencost.opencost-system.svc:9003

# 节点选择（可选）
nodeSelector:
  node-role.kubernetes.io/control-plane: ""

# 容忍度（可选）
tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
```

使用自定义 values 安装：

```bash
helm install bison bison/bison \
  --namespace bison-system \
  --create-namespace \
  --values custom-values.yaml
```

## 计费配置

计费设置在安装后通过 Web UI 或 API 进行配置。

### 访问计费配置

1. **通过 Web UI：**
   - 导航到 **设置** > **计费配置**
   - 设置 CPU、内存、GPU 和其他资源的价格
   - 配置货币和计费周期

2. **通过 API：**
   ```bash
   curl -X POST http://localhost:8080/api/v1/billing/config \
     -H "Content-Type: application/json" \
     -d '{
       "enabled": true,
       "currency": "USD",
       "pricing": {
         "cpu": 0.05,
         "memory": 0.01,
         "nvidia.com/gpu": 2.50
       },
       "billingInterval": "hourly"
     }'
   ```

### 计费参数

| 参数 | 描述 | 示例 |
|-----------|-------------|---------|
| `enabled` | 启用/禁用计费 | `true` |
| `currency` | 计费货币 | `USD`, `CNY`, `EUR` |
| `pricing.cpu` | CPU 价格（每核心小时） | `0.05` |
| `pricing.memory` | 内存价格（每 GB 小时） | `0.01` |
| `pricing["nvidia.com/gpu"]` | GPU 价格（每 GPU 小时） | `2.50` |
| `billingInterval` | 计费聚合周期 | `hourly`, `daily` |
| `lowBalanceThreshold` | 警告阈值（%） | `20` |
| `suspendThreshold` | 自动暂停阈值（%） | `5` |

### 计费配置示例

```json
{
  "enabled": true,
  "currency": "USD",
  "pricing": {
    "cpu": 0.05,
    "memory": 0.01,
    "nvidia.com/gpu": 2.50,
    "nvidia.com/mig-1g.5gb": 0.50,
    "nvidia.com/mig-2g.10gb": 1.00
  },
  "billingInterval": "hourly",
  "lowBalanceThreshold": 20,
  "suspendThreshold": 5,
  "alertChannels": ["webhook", "dingtalk"]
}
```

## 团队配置

### 创建团队

团队可以通过 Web UI 或 API 创建：

**通过 Web UI：**
1. 导航到 **团队** 页面
2. 点击 **创建团队**
3. 设置团队名称、配额和初始余额

**通过 API：**
```bash
curl -X POST http://localhost:8080/api/v1/teams \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ml-team",
    "description": "Machine Learning Team",
    "quota": {
      "cpu": "20",
      "memory": "64Gi",
      "nvidia.com/gpu": "4"
    },
    "balance": 1000.00
  }'
```

### 团队配额

团队配额定义资源限制：

```yaml
quota:
  cpu: "20"              # 20 个 CPU 核心
  memory: "64Gi"         # 64 GB 内存
  nvidia.com/gpu: "4"    # 4 个 GPU
  storage: "500Gi"       # 500 GB 存储
```

### 团队余额管理

设置初始余额并配置自动充值：

```json
{
  "balance": 1000.00,
  "autoRecharge": {
    "enabled": true,
    "amount": 500.00,
    "schedule": "monthly",
    "threshold": 100.00
  }
}
```

## 告警配置

配置多渠道告警，用于低余额和配额警告。

### Webhook 告警

```json
{
  "type": "webhook",
  "enabled": true,
  "url": "https://your-webhook-endpoint.com/alerts",
  "headers": {
    "Authorization": "Bearer YOUR_TOKEN"
  },
  "template": {
    "title": "Bison Alert",
    "message": "Team {{.TeamName}} balance is {{.Balance}}"
  }
}
```

### 钉钉告警

```json
{
  "type": "dingtalk",
  "enabled": true,
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN",
  "secret": "YOUR_SECRET"
}
```

### 企业微信告警

```json
{
  "type": "wechat",
  "enabled": true,
  "corpid": "YOUR_CORP_ID",
  "corpsecret": "YOUR_CORP_SECRET",
  "agentid": 1000001
}
```

## OpenCost 集成

配置 OpenCost 连接：

### 检查 OpenCost 连通性

```bash
# 测试 OpenCost API
kubectl port-forward -n opencost-system svc/opencost 9003:9003
curl http://localhost:9003/healthz

# 测试 allocation API
curl http://localhost:9003/allocation/compute?window=1d
```

### 更新 OpenCost URL

如果 OpenCost 部署在不同的命名空间或使用不同的服务名称：

```bash
helm upgrade bison bison/bison \
  --set opencost.url=http://my-opencost.custom-namespace.svc:9003 \
  --namespace bison-system
```

## 认证与 OIDC

启用认证并与您的 SSO 提供商集成：

### 基本认证

```yaml
auth:
  enabled: true
  admin:
    username: admin
    password: SecurePassword123
```

### OIDC 集成

```yaml
auth:
  enabled: true
  oidc:
    enabled: true
    issuerURL: https://your-oidc-provider.com
    clientID: bison-client-id
    clientSecret: your-client-secret
    redirectURL: https://bison.example.com/callback
```

## 环境变量

可以通过环境变量提供其他配置：

| 变量 | 描述 | 默认值 |
|----------|-------------|---------|
| `KUBECONFIG` | kubeconfig 文件路径 | 集群内配置 |
| `OPENCOST_URL` | OpenCost API URL | `http://opencost.opencost-system.svc:9003` |
| `AUTH_ENABLED` | 启用认证 | `false` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `BILLING_INTERVAL` | 计费计算间隔 | `10m` |

在 Helm values 中设置环境变量：

```yaml
apiServer:
  env:
    - name: LOG_LEVEL
      value: debug
    - name: BILLING_INTERVAL
      value: 5m
```

## 高级配置

### 自定义资源定价

为任何 Kubernetes 资源定价：

```json
{
  "pricing": {
    "cpu": 0.05,
    "memory": 0.01,
    "nvidia.com/gpu": 2.50,
    "amd.com/gpu": 2.00,
    "ephemeral-storage": 0.001,
    "custom.io/fpga": 5.00
  }
}
```

### 多集群支持

在每个集群中部署 Bison，共享计费：

```yaml
# 集群 A
apiServer:
  clusterName: prod-us-west

# 集群 B
apiServer:
  clusterName: prod-us-east
```

## 下一步

- [用户指南](user-guides/admin.md) - 学习如何使用 Bison
- [架构](architecture.md) - 理解系统设计
- [功能特性](features.md) - 探索所有功能

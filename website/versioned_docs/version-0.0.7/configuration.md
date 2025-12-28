---
sidebar_position: 6
---

# Configuration

This guide covers how to configure Bison for your specific environment and requirements.

## Helm Chart Configuration

Bison is configured primarily through Helm values. You can customize the installation by providing a `values.yaml` file or using `--set` flags.

### Key Configuration Parameters

| Parameter | Description | Default | Example |
|-----------|-------------|---------|---------|
| `auth.enabled` | Enable authentication | `false` | `true` |
| `auth.admin.username` | Admin username | `admin` | `admin` |
| `auth.admin.password` | Admin password | `admin` | `changeme` |
| `apiServer.replicaCount` | API server replicas | `2` | `3` |
| `apiServer.image.repository` | API server image | `ghcr.io/supermarioyl/bison/api-server` | - |
| `apiServer.image.tag` | API server image tag | `0.0.1` | `latest` |
| `webUI.replicaCount` | Web UI replicas | `2` | `3` |
| `webUI.image.repository` | Web UI image | `ghcr.io/supermarioyl/bison/web-ui` | - |
| `webUI.image.tag` | Web UI image tag | `0.0.1` | `latest` |
| `opencost.url` | OpenCost API endpoint | `http://opencost.opencost-system.svc:9003` | Custom URL |

### Example Custom Values

Create a `custom-values.yaml` file:

```yaml
# Authentication
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

# OpenCost Integration
opencost:
  url: http://opencost.opencost-system.svc:9003

# Node Selection (optional)
nodeSelector:
  node-role.kubernetes.io/control-plane: ""

# Tolerations (optional)
tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
```

Install with custom values:

```bash
helm install bison bison/bison \
  --namespace bison-system \
  --create-namespace \
  --values custom-values.yaml
```

## Billing Configuration

Billing settings are configured through the Web UI or API after installation.

### Access Billing Configuration

1. **Via Web UI:**
   - Navigate to **Settings** > **Billing Configuration**
   - Set pricing for CPU, Memory, GPU, and other resources
   - Configure currency and billing intervals

2. **Via API:**
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

### Billing Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `enabled` | Enable/disable billing | `true` |
| `currency` | Currency for billing | `USD`, `CNY`, `EUR` |
| `pricing.cpu` | CPU price per core-hour | `0.05` |
| `pricing.memory` | Memory price per GB-hour | `0.01` |
| `pricing["nvidia.com/gpu"]` | GPU price per GPU-hour | `2.50` |
| `billingInterval` | Billing aggregation period | `hourly`, `daily` |
| `lowBalanceThreshold` | Warning threshold (%) | `20` |
| `suspendThreshold` | Auto-suspend threshold (%) | `5` |

### Example Billing Configuration

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

## Team Configuration

### Creating Teams

Teams can be created through the Web UI or API:

**Via Web UI:**
1. Navigate to **Teams** page
2. Click **Create Team**
3. Set team name, quota, and initial balance

**Via API:**
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

### Team Quotas

Team quotas define resource limits:

```yaml
quota:
  cpu: "20"              # 20 CPU cores
  memory: "64Gi"         # 64 GB RAM
  nvidia.com/gpu: "4"    # 4 GPUs
  storage: "500Gi"       # 500 GB storage
```

### Team Balance Management

Set initial balance and configure auto-recharge:

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

## Alert Configuration

Configure multi-channel alerts for low balance and quota warnings.

### Webhook Alerts

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

### DingTalk Alerts

```json
{
  "type": "dingtalk",
  "enabled": true,
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN",
  "secret": "YOUR_SECRET"
}
```

### WeChat Work Alerts

```json
{
  "type": "wechat",
  "enabled": true,
  "corpid": "YOUR_CORP_ID",
  "corpsecret": "YOUR_CORP_SECRET",
  "agentid": 1000001
}
```

## OpenCost Integration

Configure OpenCost connection:

### Check OpenCost Connectivity

```bash
# Test OpenCost API
kubectl port-forward -n opencost-system svc/opencost 9003:9003
curl http://localhost:9003/healthz

# Test allocation API
curl http://localhost:9003/allocation/compute?window=1d
```

### Update OpenCost URL

If OpenCost is deployed in a different namespace or with a different service name:

```bash
helm upgrade bison bison/bison \
  --set opencost.url=http://my-opencost.custom-namespace.svc:9003 \
  --namespace bison-system
```

## Authentication & OIDC

Enable authentication and integrate with your SSO provider:

### Basic Authentication

```yaml
auth:
  enabled: true
  admin:
    username: admin
    password: SecurePassword123
```

### OIDC Integration

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

## Environment Variables

Additional configuration can be provided via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Path to kubeconfig file | In-cluster config |
| `OPENCOST_URL` | OpenCost API URL | `http://opencost.opencost-system.svc:9003` |
| `AUTH_ENABLED` | Enable authentication | `false` |
| `LOG_LEVEL` | Logging level | `info` |
| `BILLING_INTERVAL` | Billing calculation interval | `10m` |

Set environment variables in Helm values:

```yaml
apiServer:
  env:
    - name: LOG_LEVEL
      value: debug
    - name: BILLING_INTERVAL
      value: 5m
```

## Advanced Configuration

### Custom Resource Pricing

Price any Kubernetes resource:

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

### Multi-Cluster Support

Deploy Bison in each cluster with shared billing:

```yaml
# Cluster A
apiServer:
  clusterName: prod-us-west

# Cluster B
apiServer:
  clusterName: prod-us-east
```

## Next Steps

- [User Guides](user-guides/admin.md) - Learn how to use Bison
- [Architecture](architecture.md) - Understand the system design
- [Features](features.md) - Explore all capabilities

---
sidebar_position: 1
---

# Administrator Guide

This guide is for platform administrators who deploy, configure, and manage the Bison platform.

## Responsibilities

As a platform administrator, you are responsible for:

- ✅ Deploying and configuring Bison
- ✅ Creating and managing teams
- ✅ Setting global billing configuration
- ✅ Monitoring cluster-wide metrics
- ✅ Responding to alerts and recharge requests

## Getting Started

### 1. Deploy Bison

Follow the [Installation Guide](../installation.md) to deploy Bison in your Kubernetes cluster.

### 2. Configure Billing

Set up billing rules and pricing:

1. Access the Web UI
2. Navigate to **Settings** > **Billing Configuration**
3. Configure:
   - **Currency**: USD, CNY, EUR, etc.
   - **CPU Price**: Cost per core-hour
   - **Memory Price**: Cost per GB-hour
   - **GPU Price**: Cost per GPU-hour
4. Click **Save**

### 3. Create First Team

Create a team for your users:

1. Navigate to **Teams** page
2. Click **Create Team**
3. Fill in:
   - **Team Name**: e.g., "ml-team"
   - **Description**: Team purpose
   - **Resource Quota**:
     - CPU: e.g., "20" cores
     - Memory: e.g., "64Gi"
     - GPU: e.g., "4"
   - **Initial Balance**: e.g., 1000.00
4. Click **Create**

## Common Tasks

### Managing Teams

#### View All Teams

```bash
# Via kubectl
kubectl get tenants

# Via API
curl http://localhost:8080/api/v1/teams
```

#### Update Team Quota

1. Navigate to **Teams** page
2. Click **Edit** on the team row
3. Modify quotas
4. Click **Save**

#### Recharge Team Balance

1. Navigate to **Teams** page
2. Click **Recharge** on the team row
3. Enter amount
4. Add notes (optional)
5. Click **Confirm**

### Monitoring

#### View Dashboard

Access real-time cluster metrics:
- Total teams and projects
- Resource utilization
- Cost trends
- Top consumers
- Balance status

#### Check Alerts

Monitor low-balance and quota alerts:
1. Navigate to **Alerts** page
2. Review active alerts
3. Take action as needed

### Billing Configuration

#### Update Pricing

```bash
curl -X PUT http://localhost:8080/api/v1/billing/config \
  -H "Content-Type: application/json" \
  -d '{
    "pricing": {
      "cpu": 0.06,
      "memory": 0.012,
      "nvidia.com/gpu": 3.00
    }
  }'
```

#### Configure Alert Thresholds

```json
{
  "lowBalanceThreshold": 20,
  "suspendThreshold": 5,
  "alertChannels": ["webhook", "dingtalk"]
}
```

## Best Practices

### Team Naming
- Use lowercase, alphanumeric characters and hyphens
- Example: `ml-team`, `data-science`, `dev-team`

### Quota Allocation
- Start with conservative quotas
- Monitor usage for 1-2 weeks
- Adjust based on actual needs

### Balance Management
- Set up auto-recharge for critical teams
- Monitor balance trends weekly
- Respond to low-balance alerts promptly

### Security
- Enable authentication in production
- Use OIDC/SSO for enterprise deployments
- Regularly audit user permissions

## Troubleshooting

### Team Creation Failed

Check Capsule operator logs:
```bash
kubectl logs -n capsule-system deployment/capsule-controller-manager
```

### Billing Not Working

Verify OpenCost connectivity:
```bash
kubectl port-forward -n opencost-system svc/opencost 9003:9003
curl http://localhost:9003/healthz
```

### High Resource Usage

Check resource consumption:
```bash
kubectl top pods -n bison-system
```

## Next Steps

- [Team Leader Guide](team-leader.md) - Guide for team leaders
- [Developer Guide](developer.md) - Guide for developers
- [Configuration](../configuration.md) - Advanced configuration

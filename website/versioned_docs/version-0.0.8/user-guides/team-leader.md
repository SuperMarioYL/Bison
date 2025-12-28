---
sidebar_position: 2
---

# Team Leader Guide

This guide is for team leaders who manage projects, monitor budgets, and allocate resources within their team.

## Responsibilities

As a team leader, you are responsible for:

- ✅ Creating and managing projects (namespaces)
- ✅ Allocating quotas to projects
- ✅ Monitoring team balance and consumption
- ✅ Requesting recharges when needed

## Getting Started

### 1. Access Bison

Log in to the Web UI with your credentials.

### 2. View Team Dashboard

Your dashboard shows:
- Team balance and status
- Resource utilization
- Active projects
- Cost trends

## Managing Projects

### Create a Project

1. Navigate to **Projects** page
2. Click **Create Project**
3. Fill in:
   - **Project Name**: e.g., "training-ml-models"
   - **Description**: Project purpose
   - **Quota** (optional):
     - CPU: e.g., "8" cores
     - Memory: e.g., "32Gi"
     - GPU: e.g., "2"
4. Click **Create**

### List Projects

```bash
# Via kubectl (if you have access)
kubectl get namespaces -l capsule.clastix.io/tenant=your-team

# Via API
curl http://localhost:8080/api/v1/teams/your-team/projects
```

### Delete a Project

1. Navigate to **Projects** page
2. Click **Delete** on the project row
3. Confirm deletion

**Warning**: This will delete all resources in the project!

## Monitoring Budget

### Check Balance

View your current balance:
1. Navigate to **Team** page
2. See balance in the status card

### View Usage Trends

Analyze spending patterns:
1. Navigate to **Reports** page
2. Select time range (7 days, 30 days, 90 days)
3. View:
   - Cost breakdown by resource type
   - Daily cost trends
   - Per-project consumption

### Request Recharge

When balance is low:
1. Click **Request Recharge** button
2. Enter requested amount
3. Add justification
4. Submit request to administrator

## Resource Management

### Monitor Quota Usage

Check how much of your quota is being used:
```bash
kubectl describe quota -n your-project
```

### Optimize Costs

Tips to reduce spending:
- **Right-size resources**: Don't over-provision CPU/Memory
- **Clean up idle pods**: Delete unused workloads
- **Use spot/preemptible instances**: Where applicable
- **Monitor GPU utilization**: Ensure GPUs are fully utilized

## Best Practices

### Project Organization
- Create separate projects for different workloads
- Example: `ml-training`, `ml-inference`, `data-processing`

### Quota Allocation
- Allocate quotas based on project priority
- Reserve buffer for urgent tasks

### Cost Awareness
- Review costs weekly
- Identify and eliminate waste
- Set up cost alerts

## Next Steps

- [Developer Guide](developer.md) - Guide for your team members
- [Features](../features.md) - Explore all Bison features

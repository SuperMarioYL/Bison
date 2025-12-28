---
sidebar_position: 3
---

# Developer Guide

This guide is for developers who deploy workloads and consume resources within team projects.

## Responsibilities

As a developer, you are responsible for:

- ✅ Deploying applications within your project
- ✅ Monitoring resource usage
- ✅ Staying within quota limits
- ✅ Optimizing resource consumption

## Getting Started

### 1. Get Kubeconfig

Request kubeconfig from your team leader or administrator.

### 2. Set Context

```bash
# Set context to your project namespace
kubectl config set-context --current --namespace=your-project

# Verify
kubectl config view --minify | grep namespace
```

### 3. Check Quota

See your available resources:
```bash
kubectl describe quota
```

## Deploying Workloads

### Basic Pod Deployment

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-training-job
  namespace: your-project
spec:
  containers:
  - name: trainer
    image: your-ml-image:latest
    resources:
      requests:
        cpu: "4"
        memory: "16Gi"
        nvidia.com/gpu: "1"
      limits:
        cpu: "4"
        memory: "16Gi"
        nvidia.com/gpu: "1"
```

### Using Deployments

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ml-inference
  namespace: your-project
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ml-inference
  template:
    metadata:
      labels:
        app: ml-inference
    spec:
      containers:
      - name: inference
        image: your-inference-image:latest
        resources:
          requests:
            cpu: "2"
            memory: "8Gi"
            nvidia.com/gpu: "1"
```

## Monitoring Usage

### Check Pod Resource Usage

```bash
# View resource consumption
kubectl top pods

# Detailed pod information
kubectl describe pod <pod-name>
```

### View Logs

```bash
# Stream logs
kubectl logs -f <pod-name>

# Previous logs (if pod restarted)
kubectl logs --previous <pod-name>
```

## Best Practices

### Resource Requests and Limits

Always specify both requests and limits:
```yaml
resources:
  requests:
    cpu: "2"
    memory: "8Gi"
  limits:
    cpu: "4"
    memory: "16Gi"
```

### GPU Usage

- Request GPUs only when needed
- Use GPU for compute-intensive tasks
- Monitor GPU utilization

### Clean Up

Delete resources when no longer needed:
```bash
# Delete pod
kubectl delete pod <pod-name>

# Delete deployment
kubectl delete deployment <deployment-name>

# Clean up completed jobs
kubectl delete job --field-selector status.successful=1
```

### Cost Optimization

- Right-size your resource requests
- Use horizontal pod autoscaling
- Clean up idle resources
- Share GPUs when possible (if supported)

## Troubleshooting

### Pod Pending (Insufficient Quota)

If your pod is stuck in `Pending` state:

```bash
kubectl describe pod <pod-name>
```

Look for quota-related errors and reduce resource requests or ask your team leader for more quota.

### Out of Memory (OOM)

If pods are killed due to OOM:
1. Check memory usage patterns
2. Increase memory limits
3. Optimize application memory usage

### GPU Not Available

Verify GPU requests:
```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU:.status.allocatable."nvidia\.com/gpu"
```

## Next Steps

- [Team Leader Guide](team-leader.md) - Understand team management
- [Architecture](../architecture.md) - Learn about the platform

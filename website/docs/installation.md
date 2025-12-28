---
sidebar_position: 3
---

# Installation Guide

This guide provides detailed instructions for installing Bison in your Kubernetes cluster.

## Prerequisites

Before installing Bison, ensure you have:

- **Kubernetes 1.22+** - A running Kubernetes cluster
- **kubectl** - Configured to access your cluster
- **Helm 3.0+** - Package manager for Kubernetes
- **Capsule Operator v0.1.0+** - For multi-tenant isolation
- **OpenCost** - Deployed with Prometheus for cost tracking

### Install Prerequisites

If you haven't installed the required components:

#### Install Capsule

```bash
# Using Helm
helm repo add projectcapsule https://projectcapsule.github.io/charts
helm install capsule projectcapsule/capsule \
  --namespace capsule-system \
  --create-namespace
```

#### Install OpenCost

```bash
# Using Helm
helm repo add opencost https://opencost.github.io/opencost-helm-chart
helm install opencost opencost/opencost \
  --namespace opencost-system \
  --create-namespace \
  --set prometheus.internal.serviceName=prometheus-server \
  --set prometheus.internal.namespaceName=prometheus-system
```

## Installation Methods

Bison Helm charts are distributed via **GitHub Container Registry (GHCR)** using the modern OCI format.

**Requirements:**
- Helm >= 3.8.0 (for OCI support)
- Kubernetes >= 1.22

### Option A: From GHCR (Recommended)

The simplest way to install Bison is directly from GitHub Container Registry:

```bash
# Install specific version from GHCR
helm install bison oci://ghcr.io/supermarioyl/bison/bison \
  --version 0.0.2 \
  --namespace bison-system \
  --create-namespace

# Or pull the chart first, then install
helm pull oci://ghcr.io/supermarioyl/bison/bison --version 0.0.2
helm install bison bison-0.0.2.tgz \
  --namespace bison-system \
  --create-namespace

# Customize installation
helm install bison oci://ghcr.io/supermarioyl/bison/bison \
  --version 0.0.2 \
  --namespace bison-system \
  --create-namespace \
  --set opencost.url=http://opencost.opencost-system.svc:9003 \
  --set auth.enabled=true \
  --set apiServer.image.tag=0.0.2 \
  --set webUI.image.tag=0.0.2
```

**Why GHCR OCI Format?**
- ✅ No separate Helm repository maintenance needed
- ✅ Unified with Docker images in GHCR
- ✅ Faster installation (direct registry pull)
- ✅ Modern Helm 3.8+ standard practice

### Option B: From GitHub Release

Download a specific version from GitHub Releases:

```bash
# Download Helm chart
VERSION=0.0.2
wget https://github.com/SuperMarioYL/Bison/releases/download/v${VERSION}/bison-${VERSION}.tgz

# Install the chart
helm install bison bison-${VERSION}.tgz \
  --namespace bison-system \
  --create-namespace
```

### Option C: From Source

Clone and build from source:

```bash
# Clone repository
git clone https://github.com/SuperMarioYL/Bison.git
cd Bison

# Install dependencies and build
make install-deps
make build

# Deploy using Helm
helm install bison ./deploy/charts/bison \
  --namespace bison-system \
  --create-namespace
```

## Configuration Options

Bison can be configured using Helm values. Here are the key configuration options:

### Basic Configuration

```yaml
# values.yaml
apiServer:
  image:
    repository: ghcr.io/supermarioyl/bison/api-server
    tag: 0.0.1
  replicas: 2

webUI:
  image:
    repository: ghcr.io/supermarioyl/bison/web-ui
    tag: 0.0.1
  replicas: 2

# OpenCost URL
opencost:
  url: http://opencost.opencost-system.svc:9003

# Authentication
auth:
  enabled: false
```

### Custom Configuration Example

```bash
helm install bison bison/bison \
  --namespace bison-system \
  --create-namespace \
  --set apiServer.replicas=3 \
  --set webUI.replicas=3 \
  --set opencost.url=http://opencost.opencost-system.svc:9003 \
  --set auth.enabled=true
```

## Verify Installation

After installation, verify that all components are running:

```bash
# Check pod status
kubectl get pods -n bison-system

# Expected output:
# NAME                              READY   STATUS    RESTARTS   AGE
# bison-api-server-xxxxxxxxx-xxxxx  1/1     Running   0          2m
# bison-webui-xxxxxxxxx-xxxxx       1/1     Running   0          2m

# Check services
kubectl get svc -n bison-system

# Check logs
kubectl logs -n bison-system deployment/bison-api-server
kubectl logs -n bison-system deployment/bison-webui
```

## Access the Platform

### Port Forward (Development)

```bash
# Port-forward the Web UI
kubectl port-forward -n bison-system svc/bison-webui 3000:80

# Access at http://localhost:3000
```

### Ingress (Production)

For production deployments, configure an Ingress:

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bison-ingress
  namespace: bison-system
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: bison.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: bison-webui
            port:
              number: 80
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: bison-api-server
            port:
              number: 8080
```

Apply the Ingress:

```bash
kubectl apply -f ingress.yaml
```

## Docker Images

Bison images are available on GitHub Container Registry:

```bash
# Pull images
docker pull ghcr.io/supermarioyl/bison/api-server:0.0.1
docker pull ghcr.io/supermarioyl/bison/web-ui:0.0.1

# Or use latest
docker pull ghcr.io/supermarioyl/bison/api-server:latest
docker pull ghcr.io/supermarioyl/bison/web-ui:latest
```

**Supported Platforms:**
- `linux/amd64`
- `linux/arm64`

## Upgrading

To upgrade Bison to a new version:

```bash
# Update Helm repository
helm repo update

# Upgrade to latest version
helm upgrade bison bison/bison --namespace bison-system

# Or upgrade to specific version
helm upgrade bison bison/bison --version 0.0.2 --namespace bison-system
```

## Uninstalling

To completely remove Bison:

```bash
# Uninstall Helm release
helm uninstall bison --namespace bison-system

# Remove namespace (optional)
kubectl delete namespace bison-system
```

## Troubleshooting

### Pod Not Starting

Check pod logs for errors:

```bash
kubectl logs -n bison-system deployment/bison-api-server
kubectl describe pod -n bison-system <pod-name>
```

### Cannot Connect to OpenCost

Verify OpenCost is running and accessible:

```bash
kubectl get svc -n opencost-system
kubectl port-forward -n opencost-system svc/opencost 9003:9003

# Test endpoint
curl http://localhost:9003/healthz
```

### Authentication Issues

If authentication is enabled, ensure you have the correct credentials:

```bash
# Default credentials (change in production!)
Username: admin
Password: admin
```

## Next Steps

- [Configuration Guide](configuration.md) - Configure billing and settings
- [User Guides](user-guides/admin.md) - Learn how to use Bison
- [Architecture](architecture.md) - Understand the system design

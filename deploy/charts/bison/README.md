# Bison Helm Chart

Kubernetes-based GPU Resource Billing and Scheduling Platform

## Installation

**Requirements:**
- Helm >= 3.8.0 (for OCI support)
- Kubernetes >= 1.22

### Method 1: From GHCR (Recommended)

Install directly from GitHub Container Registry using OCI format:

```bash
# Install specific version
helm install my-bison oci://ghcr.io/supermarioyl/bison/bison --version 0.0.2

# Or pull first, then install
helm pull oci://ghcr.io/supermarioyl/bison/bison --version 0.0.2
helm install my-bison bison-0.0.2.tgz

# With custom configuration
helm install my-bison oci://ghcr.io/supermarioyl/bison/bison \
  --version 0.0.2 \
  --namespace bison-system \
  --create-namespace \
  --set opencost.url=http://opencost.opencost-system.svc:9003 \
  --set auth.enabled=true
```

**Why GHCR OCI Format?**
- ✅ No separate Helm repository needed
- ✅ Unified with Docker images in GHCR
- ✅ Faster installation
- ✅ Modern Helm 3.8+ standard

### Method 2: From GitHub Releases

Download the chart from [GitHub Releases](https://github.com/SuperMarioYL/Bison/releases) and install locally:

```bash
# Download from release page
wget https://github.com/SuperMarioYL/Bison/releases/download/v0.0.2/bison-0.0.2.tgz

# Install
helm install my-bison bison-0.0.2.tgz \
  --namespace bison-system \
  --create-namespace
```

## Prerequisites

Before installing Bison, ensure the following dependencies are installed:

1. **Capsule** - Multi-tenant management
   ```bash
   helm install capsule projectcapsule/capsule -n capsule-system --create-namespace
   ```

2. **OpenCost** - Cost tracking
   ```bash
   helm install opencost opencost/opencost -n opencost --create-namespace
   ```

3. **Prometheus** - Metrics collection
   ```bash
   helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
   ```

## Configuration

See [values.yaml](./values.yaml) for all configuration options.

### Basic Configuration

```bash
helm install my-bison oci://ghcr.io/supermarioyl/bison/bison \
  --set apiServer.replicas=2 \
  --set webUI.replicas=2
```

## Uninstall

```bash
helm uninstall my-bison
```

## More Information

- [Project Homepage](https://supermarioyl.github.io/Bison/)
- [Documentation](https://supermarioyl.github.io/Bison/docs/)
- [GitHub Repository](https://github.com/SuperMarioYL/Bison)

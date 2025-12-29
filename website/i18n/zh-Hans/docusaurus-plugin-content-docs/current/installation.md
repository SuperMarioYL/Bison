---
sidebar_position: 3
---

# 安装指南

本指南提供在 Kubernetes 集群中安装 Bison 的详细说明。

## 前置要求

在安装 Bison 之前，请确保您具备：

- **Kubernetes 1.22+** - 正在运行的 Kubernetes 集群
- **kubectl** - 已配置为访问您的集群
- **Helm 3.0+** - Kubernetes 包管理器
- **Capsule Operator v0.1.0+** - 用于多租户隔离
- **OpenCost** - 已与 Prometheus 一起部署用于成本追踪

### 安装前置组件

如果您还没有安装所需的组件：

#### 安装 Capsule

```bash
# 使用 Helm
helm repo add projectcapsule https://projectcapsule.github.io/charts
helm install capsule projectcapsule/capsule \
  --namespace capsule-system \
  --create-namespace
```

#### 安装 OpenCost

```bash
# 使用 Helm
helm repo add opencost https://opencost.github.io/opencost-helm-chart
helm install opencost opencost/opencost \
  --namespace opencost-system \
  --create-namespace \
  --set prometheus.internal.serviceName=prometheus-server \
  --set prometheus.internal.namespaceName=prometheus-system
```

## 安装方法

Bison Helm charts 通过 **GitHub Container Registry (GHCR)** 使用现代 OCI 格式分发。

**要求：**
- Helm >= 3.8.0（用于 OCI 支持）
- Kubernetes >= 1.22

### 方式 A：从 GHCR 安装（推荐）

从 GitHub Container Registry 直接安装 Bison 是最简单的方法：

```bash
# 从 GHCR 安装特定版本
helm install bison oci://ghcr.io/supermarioyl/bison/bison \
  --version 0.0.2 \
  --namespace bison-system \
  --create-namespace

# 或先拉取 chart，然后安装
helm pull oci://ghcr.io/supermarioyl/bison/bison --version 0.0.2
helm install bison bison-0.0.2.tgz \
  --namespace bison-system \
  --create-namespace

# 自定义安装
helm install bison oci://ghcr.io/supermarioyl/bison/bison \
  --version 0.0.2 \
  --namespace bison-system \
  --create-namespace \
  --set opencost.url=http://opencost.opencost-system.svc:9003 \
  --set auth.enabled=true \
  --set apiServer.image.tag=0.0.2 \
  --set webUI.image.tag=0.0.2
```

**为什么使用 GHCR OCI 格式？**
- ✅ 无需维护单独的 Helm 仓库
- ✅ 在 GHCR 中与 Docker 镜像统一
- ✅ 更快的安装速度（直接从注册表拉取）
- ✅ 现代 Helm 3.8+ 标准实践

### 方式 B：从 GitHub Release 安装

从 GitHub Releases 下载特定版本：

```bash
# 下载 Helm chart
VERSION=0.0.2
wget https://github.com/SuperMarioYL/Bison/releases/download/v${VERSION}/bison-${VERSION}.tgz

# 安装 chart
helm install bison bison-${VERSION}.tgz \
  --namespace bison-system \
  --create-namespace
```

### 方式 C：从源码安装

克隆并从源码构建：

```bash
# 克隆仓库
git clone https://github.com/SuperMarioYL/Bison.git
cd Bison

# 安装依赖并构建
make install-deps
make build

# 使用 Helm 部署
helm install bison ./deploy/charts/bison \
  --namespace bison-system \
  --create-namespace
```

## 配置选项

Bison 可以使用 Helm values 进行配置。以下是关键配置选项：

### 基本配置

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

# 认证
auth:
  enabled: false
```

### 自定义配置示例

```bash
helm install bison bison/bison \
  --namespace bison-system \
  --create-namespace \
  --set apiServer.replicas=3 \
  --set webUI.replicas=3 \
  --set opencost.url=http://opencost.opencost-system.svc:9003 \
  --set auth.enabled=true
```

## 验证安装

安装后，验证所有组件是否正在运行：

```bash
# 检查 pod 状态
kubectl get pods -n bison-system

# 预期输出：
# NAME                              READY   STATUS    RESTARTS   AGE
# bison-api-server-xxxxxxxxx-xxxxx  1/1     Running   0          2m
# bison-webui-xxxxxxxxx-xxxxx       1/1     Running   0          2m

# 检查服务
kubectl get svc -n bison-system

# 检查日志
kubectl logs -n bison-system deployment/bison-api-server
kubectl logs -n bison-system deployment/bison-webui
```

## 访问平台

### 端口转发（开发环境）

```bash
# 端口转发 Web UI
kubectl port-forward -n bison-system svc/bison-webui 3000:80

# 访问 http://localhost:3000
```

### Ingress（生产环境）

对于生产部署，配置 Ingress：

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

应用 Ingress：

```bash
kubectl apply -f ingress.yaml
```

## Docker 镜像

Bison 镜像可在 GitHub Container Registry 上获取：

```bash
# 拉取镜像
docker pull ghcr.io/supermarioyl/bison/api-server:0.0.1
docker pull ghcr.io/supermarioyl/bison/web-ui:0.0.1

# 或使用 latest
docker pull ghcr.io/supermarioyl/bison/api-server:latest
docker pull ghcr.io/supermarioyl/bison/web-ui:latest
```

**支持的平台：**
- `linux/amd64`
- `linux/arm64`

## 升级

将 Bison 升级到新版本：

```bash
# 更新 Helm 仓库
helm repo update

# 升级到最新版本
helm upgrade bison bison/bison --namespace bison-system

# 或升级到特定版本
helm upgrade bison bison/bison --version 0.0.2 --namespace bison-system
```

## 卸载

完全删除 Bison：

```bash
# 卸载 Helm release
helm uninstall bison --namespace bison-system

# 删除命名空间（可选）
kubectl delete namespace bison-system
```

## 故障排查

### Pod 无法启动

检查 pod 日志以查找错误：

```bash
kubectl logs -n bison-system deployment/bison-api-server
kubectl describe pod -n bison-system <pod-name>
```

### 无法连接到 OpenCost

验证 OpenCost 是否正在运行且可访问：

```bash
kubectl get svc -n opencost-system
kubectl port-forward -n opencost-system svc/opencost 9003:9003

# 测试端点
curl http://localhost:9003/healthz
```

### 认证问题

如果启用了认证，请确保您有正确的凭据：

```bash
# 默认凭据（生产环境请更改！）
用户名: admin
密码: admin
```

## 下一步

- [配置指南](configuration.md) - 配置计费和设置
- [用户指南](user-guides/admin.md) - 学习如何使用 Bison
- [架构](architecture.md) - 理解系统设计

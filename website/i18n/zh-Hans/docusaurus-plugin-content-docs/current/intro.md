---
sidebar_position: 1
---

# 简介

让我们**在不到 5 分钟内**了解 **Bison**。

## 开始使用

通过**创建新集群**或**添加 Bison 到现有 Kubernetes 集群**开始使用。

### 您需要什么

- [Kubernetes](https://kubernetes.io/) 版本 1.22 或更高:
  - 运行中的 Kubernetes 集群
  - 已配置 kubectl 访问
- [Helm](https://helm.sh/) 版本 3.x 或更高
- [Capsule](https://capsule.clastix.io/) 用于多租户管理
- [OpenCost](https://www.opencost.io/) 用于成本追踪
- [Prometheus](https://prometheus.io/) 用于指标收集

## 安装 Bison

使用 Helm 在您的 Kubernetes 集群中安装 Bison:

```bash
# 添加 Bison Helm 仓库
helm repo add bison https://supermarioyl.github.io/Bison/
helm repo update

# 安装 Bison
helm install bison bison/bison \
  --namespace bison-system \
  --create-namespace \
  --set opencost.url=http://opencost.opencost-system:9003
```

## 配置您的第一个租户

安装完成后,创建您的第一个租户(团队):

```bash
kubectl apply -f - <<EOF
apiVersion: capsule.clastix.io/v1beta2
kind: Tenant
metadata:
  name: team-ai
spec:
  owners:
  - name: admin@team-ai.com
    kind: User
EOF
```

恭喜!您已经在 Kubernetes 集群上成功安装并配置了 **Bison**!🎉

## 下一步

- 了解 Bison 的[核心功能](./features.md)
- 探索[架构设计](./architecture.md)
- 查看[用户指南](./category/user-guides)了解不同角色的使用方法

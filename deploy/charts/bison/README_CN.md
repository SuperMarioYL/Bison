# Bison Helm Chart

基于 Kubernetes 的 GPU 资源计费与调度平台

## 安装

### 从 GHCR 安装（推荐）

直接从 GitHub Container Registry 使用 OCI 格式安装：

```bash
# 安装指定版本
helm install my-bison oci://ghcr.io/supermarioyl/bison/bison --version 0.0.2

# 或者先拉取，再安装
helm pull oci://ghcr.io/supermarioyl/bison/bison --version 0.0.2
helm install my-bison bison-0.0.2.tgz
```

**要求：**
- Helm >= 3.8.0（支持 OCI）

### 从 GitHub Releases 安装

从 [GitHub Releases](https://github.com/SuperMarioYL/Bison/releases) 下载 chart 并本地安装：

```bash
# 从 release 页面下载
wget https://github.com/SuperMarioYL/Bison/releases/download/v0.0.2/bison-0.0.2.tgz

# 安装
helm install my-bison bison-0.0.2.tgz
```

## 前置条件

安装 Bison 前，请确保已安装以下依赖：

1. **Capsule** - 多租户管理
   ```bash
   helm install capsule projectcapsule/capsule -n capsule-system --create-namespace
   ```

2. **OpenCost** - 成本追踪
   ```bash
   helm install opencost opencost/opencost -n opencost --create-namespace
   ```

3. **Prometheus** - 指标收集
   ```bash
   helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
   ```

## 配置

所有配置选项请查看 [values.yaml](./values.yaml)。

### 基础配置

```bash
helm install my-bison oci://ghcr.io/supermarioyl/bison/bison \
  --set apiServer.replicas=2 \
  --set webUI.replicas=2
```

### 常用配置项

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `apiServer.replicas` | API 服务器副本数 | `1` |
| `webUI.replicas` | Web UI 副本数 | `1` |
| `auth.enabled` | 启用认证 | `false` |
| `opencost.url` | OpenCost API 地址 | `http://opencost.opencost:9003` |

## 卸载

```bash
helm uninstall my-bison
```

## 更多信息

- [项目主页](https://supermarioyl.github.io/Bison/)
- [文档](https://supermarioyl.github.io/Bison/docs/)
- [GitHub 仓库](https://github.com/SuperMarioYL/Bison)
- [English README](./README.md)

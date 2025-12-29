---
sidebar_position: 3
---

# 开发者指南

本指南面向在团队项目中部署工作负载和消耗资源的开发者。

## 职责

作为开发者，您负责：

- ✅ 在您的项目中部署应用程序
- ✅ 监控资源使用情况
- ✅ 保持在配额限制内
- ✅ 优化资源消耗

## 入门

### 1. 获取 Kubeconfig

向您的团队负责人或管理员请求 kubeconfig。

### 2. 设置上下文

```bash
# 将上下文设置为您的项目命名空间
kubectl config set-context --current --namespace=your-project

# 验证
kubectl config view --minify | grep namespace
```

### 3. 检查配额

查看您的可用资源：
```bash
kubectl describe quota
```

## 部署工作负载

### 基本 Pod 部署

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

### 使用 Deployments

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

## 监控使用情况

### 检查 Pod 资源使用情况

```bash
# 查看资源消耗
kubectl top pods

# 详细的 pod 信息
kubectl describe pod <pod-name>
```

### 查看日志

```bash
# 流式查看日志
kubectl logs -f <pod-name>

# 查看之前的日志（如果 pod 重启了）
kubectl logs --previous <pod-name>
```

## 最佳实践

### 资源请求和限制

始终指定请求和限制：
```yaml
resources:
  requests:
    cpu: "2"
    memory: "8Gi"
  limits:
    cpu: "4"
    memory: "16Gi"
```

### GPU 使用

- 仅在需要时请求 GPU
- 将 GPU 用于计算密集型任务
- 监控 GPU 利用率

### 清理

不再需要时删除资源：
```bash
# 删除 pod
kubectl delete pod <pod-name>

# 删除 deployment
kubectl delete deployment <deployment-name>

# 清理已完成的 job
kubectl delete job --field-selector status.successful=1
```

### 成本优化

- 正确调整资源请求的大小
- 使用水平 pod 自动扩展
- 清理空闲资源
- 在可能的情况下共享 GPU（如果支持）

## 故障排查

### Pod 处于 Pending 状态（配额不足）

如果您的 pod 卡在 `Pending` 状态：

```bash
kubectl describe pod <pod-name>
```

查找与配额相关的错误，并减少资源请求或向团队负责人申请更多配额。

### 内存不足 (OOM)

如果 pod 因 OOM 被杀死：
1. 检查内存使用模式
2. 增加内存限制
3. 优化应用程序内存使用

### GPU 不可用

验证 GPU 请求：
```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU:.status.allocatable."nvidia\.com/gpu"
```

## 下一步

- [团队负责人指南](team-leader.md) - 了解团队管理
- [架构](../architecture.md) - 了解平台

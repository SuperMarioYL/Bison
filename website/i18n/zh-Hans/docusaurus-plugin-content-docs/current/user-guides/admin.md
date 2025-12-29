---
sidebar_position: 1
---

# 管理员指南

本指南面向部署、配置和管理 Bison 平台的平台管理员。

## 职责

作为平台管理员，您负责：

- ✅ 部署和配置 Bison
- ✅ 创建和管理团队
- ✅ 设置全局计费配置
- ✅ 监控集群范围的指标
- ✅ 响应告警和充值请求

## 入门

### 1. 部署 Bison

按照[安装指南](../installation.md)在您的 Kubernetes 集群中部署 Bison。

### 2. 配置计费

设置计费规则和定价：

1. 访问 Web UI
2. 导航到 **设置** > **计费配置**
3. 配置：
   - **货币**: USD、CNY、EUR 等
   - **CPU 价格**: 每核心小时的成本
   - **内存价格**: 每 GB 小时的成本
   - **GPU 价格**: 每 GPU 小时的成本
4. 点击 **保存**

### 3. 创建第一个团队

为您的用户创建团队：

1. 导航到 **团队** 页面
2. 点击 **创建团队**
3. 填写：
   - **团队名称**: 例如 "ml-team"
   - **描述**: 团队用途
   - **资源配额**:
     - CPU: 例如 "20" 核心
     - 内存: 例如 "64Gi"
     - GPU: 例如 "4"
   - **初始余额**: 例如 1000.00
4. 点击 **创建**

## 常见任务

### 管理团队

#### 查看所有团队

```bash
# 通过 kubectl
kubectl get tenants

# 通过 API
curl http://localhost:8080/api/v1/teams
```

#### 更新团队配额

1. 导航到 **团队** 页面
2. 点击团队行上的 **编辑**
3. 修改配额
4. 点击 **保存**

#### 充值团队余额

1. 导航到 **团队** 页面
2. 点击团队行上的 **充值**
3. 输入金额
4. 添加备注（可选）
5. 点击 **确认**

### 监控

#### 查看仪表板

访问实时集群指标：
- 总团队数和项目数
- 资源利用率
- 成本趋势
- 热门消费者
- 余额状态

#### 检查告警

监控低余额和配额告警：
1. 导航到 **告警** 页面
2. 查看活动告警
3. 根据需要采取行动

### 计费配置

#### 更新定价

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

#### 配置告警阈值

```json
{
  "lowBalanceThreshold": 20,
  "suspendThreshold": 5,
  "alertChannels": ["webhook", "dingtalk"]
}
```

## 最佳实践

### 团队命名
- 使用小写字母、数字和连字符
- 示例：`ml-team`、`data-science`、`dev-team`

### 配额分配
- 从保守的配额开始
- 监控 1-2 周的使用情况
- 根据实际需求调整

### 余额管理
- 为关键团队设置自动充值
- 每周监控余额趋势
- 及时响应低余额告警

### 安全
- 在生产环境中启用认证
- 使用 OIDC/SSO 进行企业部署
- 定期审计用户权限

## 故障排查

### 团队创建失败

检查 Capsule operator 日志：
```bash
kubectl logs -n capsule-system deployment/capsule-controller-manager
```

### 计费无法工作

验证 OpenCost 连接性：
```bash
kubectl port-forward -n opencost-system svc/opencost 9003:9003
curl http://localhost:9003/healthz
```

### 高资源使用率

检查资源消耗：
```bash
kubectl top pods -n bison-system
```

## 下一步

- [团队负责人指南](team-leader.md) - 团队负责人指南
- [开发者指南](developer.md) - 开发者指南
- [配置](../configuration.md) - 高级配置

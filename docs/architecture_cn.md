# Bison 技术架构

<p align="center">
  <a href="./architecture.md">English Version</a>
</p>

本文档详细介绍 Bison 的技术架构，采用**高内聚、低耦合**的设计原则，确保系统的可维护性和可扩展性。

---

## 目录

- [系统概览](#系统概览)
- [架构分层](#架构分层)
- [核心组件](#核心组件)
- [数据流转](#数据流转)
- [集成接口](#集成接口)
- [部署架构](#部署架构)
- [安全模型](#安全模型)

---

## 系统概览

### 整体架构

```mermaid
graph TB
    subgraph "表现层"
        WEB[Web UI<br/>React 18 + Ant Design 5]
        CLI[kubectl / API 客户端]
    end

    subgraph "网关层"
        GW[API Server<br/>Go + Gin 框架]
        AUTH[认证中间件<br/>JWT + OIDC]
    end

    subgraph "业务逻辑层"
        TS[租户服务<br/>团队与项目管理]
        BS[计费服务<br/>成本计算]
        BLS[余额服务<br/>钱包管理]
        QS[配额服务<br/>资源限制]
        AS[告警服务<br/>通知推送]
        RS[报表服务<br/>数据分析]
    end

    subgraph "集成层"
        K8S[Kubernetes 客户端<br/>client-go]
        OCC[OpenCost 客户端<br/>REST API]
        PC[Prometheus 客户端<br/>PromQL]
    end

    subgraph "外部系统"
        KAPI[Kubernetes API]
        CAP[Capsule 控制器]
        OC[OpenCost]
        PROM[Prometheus]
    end

    subgraph "数据层"
        CM[ConfigMaps<br/>持久化存储]
    end

    WEB --> GW
    CLI --> GW
    GW --> AUTH
    AUTH --> TS & BS & BLS & QS & AS & RS

    TS --> K8S
    BS --> OCC
    BLS --> K8S
    QS --> K8S
    RS --> OCC & PC

    K8S --> KAPI
    K8S --> CAP
    OCC --> OC
    PC --> PROM

    TS & BLS --> CM
    KAPI --> CM
```

### 设计原则

| 原则 | 实现方式 |
|------|----------|
| **高内聚** | 每个服务只处理单一领域（计费、配额、告警） |
| **低耦合** | 服务间通过明确定义的接口通信 |
| **无状态 API** | 所有状态持久化在 Kubernetes ConfigMaps |
| **云原生** | 利用 Kubernetes 原生能力实现高可用和弹性伸缩 |
| **零数据库** | ConfigMaps 消除外部数据库依赖 |

---

## 架构分层

### 分层图

```mermaid
graph LR
    subgraph "第1层: 表现层"
        direction TB
        A1[React SPA]
        A2[REST API]
    end

    subgraph "第2层: 应用层"
        direction TB
        B1[处理器 Handlers]
        B2[服务 Services]
        B3[调度器 Scheduler]
    end

    subgraph "第3层: 领域层"
        direction TB
        C1[团队领域]
        C2[计费领域]
        C3[告警领域]
    end

    subgraph "第4层: 基础设施层"
        direction TB
        D1[K8s 客户端]
        D2[OpenCost 客户端]
        D3[ConfigMap 存储]
    end

    A1 --> A2
    A2 --> B1
    B1 --> B2
    B2 --> C1 & C2 & C3
    C1 & C2 & C3 --> D1 & D2 & D3
```

### 各层职责

#### 表现层
- **Web UI**: React 单页应用，使用 Ant Design Pro 组件库
- **REST API**: 遵循 OpenAPI 3.0 规范的 RESTful 接口

#### 应用层
- **处理器**: HTTP 请求/响应处理、参数校验
- **服务**: 业务逻辑编排
- **调度器**: 后台定时任务（计费、告警、自动充值）

#### 领域层
- **团队领域**: Capsule Tenant 生命周期管理
- **计费领域**: 成本计算、余额管理
- **告警领域**: 阈值监控、通知推送

#### 基础设施层
- **Kubernetes 客户端**: Tenant、Namespace、ConfigMap 的 CRUD
- **OpenCost 客户端**: 查询成本分配 API
- **ConfigMap 存储**: 数据持久化抽象

---

## 核心组件

### 后端服务架构

```mermaid
classDiagram
    class Handler {
        +TeamHandler
        +ProjectHandler
        +BillingHandler
        +AlertHandler
        +StatsHandler
    }

    class Service {
        <<interface>>
        +Create()
        +Get()
        +Update()
        +Delete()
        +List()
    }

    class TenantService {
        -k8sClient
        +CreateTeam() 创建团队
        +UpdateQuota() 更新配额
        +BindNodes() 绑定节点
        +SuspendTeam() 暂停团队
    }

    class BillingService {
        -opencostClient
        -balanceService
        +CalculateCost() 计算成本
        +ProcessBilling() 处理计费
        +GetUsageReport() 获取报表
    }

    class BalanceService {
        -k8sClient
        +GetBalance() 获取余额
        +Recharge() 充值
        +Deduct() 扣费
        +SetAutoRecharge() 设置自动充值
    }

    class AlertService {
        -notifier
        +CheckThresholds() 检查阈值
        +SendAlert() 发送告警
        +GetAlertHistory() 告警历史
    }

    Handler --> Service
    Service <|-- TenantService
    Service <|-- BillingService
    Service <|-- BalanceService
    Service <|-- AlertService

    BillingService --> BalanceService
    BillingService --> AlertService
```

### 服务依赖关系

```mermaid
graph TD
    subgraph "独立服务"
        TS[TenantService<br/>租户服务]
        AS[AlertService<br/>告警服务]
    end

    subgraph "依赖服务"
        BLS[BalanceService<br/>余额服务]
        BS[BillingService<br/>计费服务]
        RS[ReportService<br/>报表服务]
    end

    BS --> BLS
    BS --> AS
    RS --> BS
    RS --> TS
```

### 前端架构

```mermaid
graph TB
    subgraph "React 应用"
        subgraph "状态管理"
            CTX[React Context<br/>全局状态]
            RQ[React Query<br/>服务端状态]
        end

        subgraph "页面组件"
            DASH[Dashboard<br/>仪表盘]
            TEAM[TeamManagement<br/>团队管理]
            PROJ[ProjectManagement<br/>项目管理]
            BILL[Billing<br/>计费中心]
            REPORT[Reports<br/>报表中心]
            SETTINGS[Settings<br/>系统设置]
        end

        subgraph "公共组件"
            LAYOUT[Layout 布局]
            TABLE[ProTable 表格]
            FORM[ProForm 表单]
            CHART[ECharts 图表]
        end

        subgraph "API 服务"
            API[API Service<br/>Axios 封装]
        end
    end

    CTX --> DASH & TEAM & PROJ & BILL
    RQ --> API
    DASH & TEAM & PROJ --> TABLE & FORM & CHART
    API --> |HTTP| BE[后端 API]
```

---

## 数据流转

### 计费周期

```mermaid
sequenceDiagram
    autonumber
    participant SCHED as 调度器
    participant OC as OpenCost
    participant BILL as 计费服务
    participant BAL as 余额服务
    participant ALERT as 告警服务
    participant CM as ConfigMaps
    participant NOTIFY as 通知器

    loop 每小时执行
        SCHED->>SCHED: 触发计费任务

        par 查询所有团队
            SCHED->>OC: GET /allocation?window=1h
            OC-->>SCHED: 命名空间成本数据
        end

        loop 遍历每个团队
            SCHED->>BILL: 计算团队成本
            BILL->>BAL: 获取当前余额
            BAL->>CM: 读取 team-balances
            CM-->>BAL: 余额数据
            BAL-->>BILL: 当前余额

            BILL->>BILL: 计算扣费金额
            BILL->>BAL: 扣除费用
            BAL->>CM: 更新 team-balances
            BILL->>CM: 写入审计日志

            alt 余额 < 告警阈值
                BILL->>ALERT: 触发低余额告警
                ALERT->>CM: 记录告警
                ALERT->>NOTIFY: 发送通知
            end

            alt 余额 <= 0
                BILL->>BILL: 标记团队为暂停状态
            end
        end
    end
```

### 团队创建流程

```mermaid
sequenceDiagram
    autonumber
    participant UI as Web UI
    participant API as API Server
    participant TS as 租户服务
    participant K8S as Kubernetes
    participant CAP as Capsule

    UI->>API: POST /api/v1/teams
    API->>API: 参数校验
    API->>TS: CreateTeam(teamData)

    TS->>K8S: 创建 Capsule Tenant
    K8S->>CAP: 协调 Tenant 资源
    CAP-->>K8S: Tenant 就绪

    TS->>K8S: 创建 ConfigMap 条目
    K8S-->>TS: ConfigMap 已更新

    TS-->>API: 团队创建成功
    API-->>UI: 201 Created
```

### 项目命名空间生命周期

```mermaid
sequenceDiagram
    autonumber
    participant UI as Web UI
    participant API as API Server
    participant PS as 项目服务
    participant K8S as Kubernetes
    participant CAP as Capsule

    UI->>API: POST /api/v1/projects
    API->>PS: CreateProject(projectData)

    PS->>K8S: 创建 Namespace（带标签）
    Note over K8S: capsule.clastix.io/tenant: team-name

    K8S->>CAP: 验证租户所有权
    CAP-->>K8S: 验证通过

    PS->>K8S: 应用 ResourceQuota
    PS->>K8S: 应用 NetworkPolicy

    PS-->>API: 项目创建成功
    API-->>UI: 201 Created
```

---

## 集成接口

### Capsule 集成

```mermaid
graph LR
    subgraph "Bison"
        TS[租户服务]
    end

    subgraph "Capsule"
        CTRL[Capsule Controller]
        TEN[Tenant CRD]
    end

    subgraph "Kubernetes"
        NS[Namespaces]
        RQ[ResourceQuotas]
        LR[LimitRanges]
    end

    TS -->|创建/更新| TEN
    CTRL -->|监听| TEN
    CTRL -->|协调| NS
    CTRL -->|应用| RQ & LR
```

**Tenant CRD 映射关系：**

| Bison 概念 | Capsule 资源 |
|------------|--------------|
| 团队 | Tenant |
| 项目 | Namespace（属于 Tenant） |
| 团队管理员 | Tenant Owners（OIDC 组） |
| 资源配额 | Tenant ResourceQuota |
| 节点绑定 | Tenant NodeSelector |

### OpenCost 集成

```mermaid
graph LR
    subgraph "Bison"
        CS[成本服务]
        RS[报表服务]
    end

    subgraph "OpenCost"
        API[Allocation API<br/>:9003/allocation]
        UI[OpenCost UI<br/>:9090]
    end

    subgraph "Prometheus"
        PROM[Prometheus Server]
        METRICS[容器指标]
    end

    CS -->|GET /allocation| API
    RS -->|GET /allocation| API
    API -->|查询| PROM
    PROM -->|采集| METRICS
```

**OpenCost API 使用示例：**

```bash
# 按命名空间查询每小时成本
GET /allocation?window=1h&aggregate=namespace

# 响应结构
{
  "namespace-name": {
    "cpuCost": 0.05,
    "memoryCost": 0.02,
    "gpuCost": 2.50,
    "totalCost": 2.57
  }
}
```

---

## 部署架构

### Kubernetes 资源

```mermaid
graph TB
    subgraph "bison-system 命名空间"
        subgraph "API Server"
            DEP1[Deployment<br/>副本数: 2]
            SVC1[Service<br/>ClusterIP]
            ING1[Ingress]
        end

        subgraph "Web UI"
            DEP2[Deployment<br/>副本数: 2]
            SVC2[Service<br/>ClusterIP]
            ING2[Ingress]
        end

        subgraph "数据存储"
            CM1[ConfigMap<br/>bison-billing-config]
            CM2[ConfigMap<br/>bison-team-balances]
            CM3[ConfigMap<br/>bison-auto-recharge]
            CM4[ConfigMap<br/>bison-audit-logs]
            SEC[Secret<br/>bison-auth]
        end

        subgraph "权限控制"
            SA[ServiceAccount]
            CR[ClusterRole]
            CRB[ClusterRoleBinding]
        end
    end

    DEP1 --> SVC1 --> ING1
    DEP2 --> SVC2 --> ING2
    DEP1 --> CM1 & CM2 & CM3 & CM4
    DEP1 --> SEC
    DEP1 --> SA
    SA --> CRB --> CR
```

### 高可用架构

```mermaid
graph TB
    subgraph "负载均衡"
        LB[Ingress Controller]
    end

    subgraph "API Server 池"
        API1[API Pod 1]
        API2[API Pod 2]
        API3[API Pod N]
    end

    subgraph "Web UI 池"
        WEB1[Web Pod 1]
        WEB2[Web Pod 2]
    end

    subgraph "共享状态"
        CM[ConfigMaps<br/>etcd 存储]
    end

    LB --> API1 & API2 & API3
    LB --> WEB1 & WEB2
    API1 & API2 & API3 --> CM
```

---

## 安全模型

### 认证与授权

```mermaid
sequenceDiagram
    participant USER as 用户
    participant UI as Web UI
    participant API as API Server
    participant AUTH as 认证中间件

    USER->>UI: 登录请求
    UI->>API: POST /api/v1/auth/login
    API->>AUTH: 验证凭据
    AUTH-->>API: 生成 JWT
    API-->>UI: JWT Token

    USER->>UI: 访问资源
    UI->>API: GET /api/v1/teams<br/>Authorization: Bearer JWT
    API->>AUTH: 验证 JWT
    AUTH-->>API: 提取用户信息
    API-->>UI: 返回资源数据
```

### RBAC 权限配置

```mermaid
graph TD
    subgraph "ClusterRole: bison-api"
        P1[configmaps: 增删改查]
        P2[namespaces: 增删改查]
        P3[resourcequotas: 增删改查]
        P4[pods: 查看、列表、删除]
        P5[tenants.capsule: 增删改查]
        P6[nodes: 查看、列表、更新]
    end

    subgraph "作用范围"
        S1[集群级别权限]
    end

    P1 & P2 & P3 & P4 & P5 & P6 --> S1
```

---

## 界面设计

### 仪表盘

<p align="center">
  <img src="./images/ui-dashboard.png" alt="Dashboard" width="90%" />
</p>

**设计要点：**
- 资源使用率实时展示（CPU/内存/GPU）
- 成本趋势图表（日/周/月）
- 团队 Top N 排行
- 告警概览

### 团队管理

<p align="center">
  <img src="./images/ui-team.png" alt="Team Management" width="90%" />
</p>

**功能模块：**
- 团队列表与搜索
- 配额配置（动态资源类型）
- 余额管理（充值/扣费历史）
- 成员与权限设置

### 计费配置

<p align="center">
  <img src="./images/ui-billing.png" alt="Billing Config" width="90%" />
</p>

**配置项：**
- 启用/禁用计费
- 币种设置
- 资源单价（CPU/内存/GPU）
- 计费周期
- 告警阈值

---

## 技术栈总结

| 层级 | 技术 | 用途 |
|------|------|------|
| 前端框架 | React 18 + TypeScript | 单页应用开发 |
| UI 组件库 | Ant Design Pro 5 | 企业级组件 |
| 图表库 | ECharts | 数据可视化 |
| 后端框架 | Go 1.21 + Gin | REST API 开发 |
| K8s 客户端 | client-go | Kubernetes 集成 |
| 多租户 | Capsule | 命名空间隔离 |
| 成本追踪 | OpenCost | 资源计费 |
| 指标采集 | Prometheus | 时序数据 |
| 数据存储 | ConfigMaps | 状态持久化 |
| 部署工具 | Helm 3 | 应用打包 |

---

<p align="center">
  <em>专为企业级 GPU 资源管理而设计</em>
</p>

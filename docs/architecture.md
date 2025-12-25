# Bison Technical Architecture

<p align="center">
  <a href="./architecture_cn.md">中文版</a>
</p>

This document provides a comprehensive technical overview of Bison's architecture, designed with **high cohesion and low coupling** principles for maintainability and scalability.

---

## Table of Contents

- [System Overview](#system-overview)
- [Architecture Layers](#architecture-layers)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [Integration Points](#integration-points)
- [Deployment Architecture](#deployment-architecture)
- [Security Model](#security-model)

---

## System Overview

### High-Level Architecture

```mermaid
graph TB
    subgraph "Presentation Layer"
        WEB[Web UI<br/>React 18 + Ant Design 5]
        CLI[kubectl / API Client]
    end

    subgraph "API Gateway Layer"
        GW[API Server<br/>Go + Gin Framework]
        AUTH[Auth Middleware<br/>JWT + OIDC]
    end

    subgraph "Business Logic Layer"
        TS[Tenant Service<br/>Team & Project CRUD]
        BS[Billing Service<br/>Cost Calculation]
        BLS[Balance Service<br/>Wallet Management]
        QS[Quota Service<br/>Resource Limits]
        AS[Alert Service<br/>Notifications]
        RS[Report Service<br/>Analytics]
    end

    subgraph "Integration Layer"
        K8S[Kubernetes Client<br/>client-go]
        OCC[OpenCost Client<br/>REST API]
        PC[Prometheus Client<br/>PromQL]
    end

    subgraph "External Systems"
        KAPI[Kubernetes API]
        CAP[Capsule Controller]
        OC[OpenCost]
        PROM[Prometheus]
    end

    subgraph "Data Layer"
        CM[ConfigMaps<br/>Persistent Storage]
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

### Design Principles

| Principle | Implementation |
|-----------|----------------|
| **High Cohesion** | Each service handles a single domain (billing, quota, alerts) |
| **Low Coupling** | Services communicate via well-defined interfaces |
| **Stateless API** | All state persisted in Kubernetes ConfigMaps |
| **Cloud Native** | Leverages Kubernetes primitives for HA and scaling |
| **Zero Database** | ConfigMaps eliminate external database dependencies |

---

## Architecture Layers

### Layer Diagram

```mermaid
graph LR
    subgraph "Layer 1: Presentation"
        direction TB
        A1[React SPA]
        A2[REST API]
    end

    subgraph "Layer 2: Application"
        direction TB
        B1[Handlers]
        B2[Services]
        B3[Scheduler]
    end

    subgraph "Layer 3: Domain"
        direction TB
        C1[Team Domain]
        C2[Billing Domain]
        C3[Alert Domain]
    end

    subgraph "Layer 4: Infrastructure"
        direction TB
        D1[K8s Client]
        D2[OpenCost Client]
        D3[ConfigMap Store]
    end

    A1 --> A2
    A2 --> B1
    B1 --> B2
    B2 --> C1 & C2 & C3
    C1 & C2 & C3 --> D1 & D2 & D3
```

### Layer Responsibilities

#### Presentation Layer
- **Web UI**: React SPA with Ant Design Pro components
- **REST API**: RESTful endpoints following OpenAPI 3.0

#### Application Layer
- **Handlers**: HTTP request/response handling, validation
- **Services**: Business logic orchestration
- **Scheduler**: Background jobs (billing, alerts, auto-recharge)

#### Domain Layer
- **Team Domain**: Capsule Tenant lifecycle
- **Billing Domain**: Cost calculation, balance management
- **Alert Domain**: Threshold monitoring, notifications

#### Infrastructure Layer
- **Kubernetes Client**: CRUD for Tenants, Namespaces, ConfigMaps
- **OpenCost Client**: Query cost allocation API
- **ConfigMap Store**: Data persistence abstraction

---

## Core Components

### Backend Services

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
        +CreateTeam()
        +UpdateQuota()
        +BindNodes()
        +SuspendTeam()
    }

    class BillingService {
        -opencostClient
        -balanceService
        +CalculateCost()
        +ProcessBilling()
        +GetUsageReport()
    }

    class BalanceService {
        -k8sClient
        +GetBalance()
        +Recharge()
        +Deduct()
        +SetAutoRecharge()
    }

    class AlertService {
        -notifier
        +CheckThresholds()
        +SendAlert()
        +GetAlertHistory()
    }

    Handler --> Service
    Service <|-- TenantService
    Service <|-- BillingService
    Service <|-- BalanceService
    Service <|-- AlertService

    BillingService --> BalanceService
    BillingService --> AlertService
```

### Service Dependencies

```mermaid
graph TD
    subgraph "Independent Services"
        TS[TenantService]
        AS[AlertService]
    end

    subgraph "Dependent Services"
        BLS[BalanceService]
        BS[BillingService]
        RS[ReportService]
    end

    BS --> BLS
    BS --> AS
    RS --> BS
    RS --> TS
```

### Frontend Architecture

```mermaid
graph TB
    subgraph "React Application"
        subgraph "State Management"
            CTX[React Context]
            RQ[React Query<br/>TanStack]
        end

        subgraph "Pages"
            DASH[Dashboard]
            TEAM[Team Management]
            PROJ[Project Management]
            BILL[Billing]
            REPORT[Reports]
            SETTINGS[Settings]
        end

        subgraph "Shared Components"
            LAYOUT[Layout]
            TABLE[ProTable]
            FORM[ProForm]
            CHART[ECharts]
        end

        subgraph "Services"
            API[API Service<br/>Axios]
        end
    end

    CTX --> DASH & TEAM & PROJ & BILL
    RQ --> API
    DASH & TEAM & PROJ --> TABLE & FORM & CHART
    API --> |HTTP| BE[Backend API]
```

---

## Data Flow

### Billing Cycle

```mermaid
sequenceDiagram
    autonumber
    participant SCHED as Scheduler
    participant OC as OpenCost
    participant BILL as BillingService
    participant BAL as BalanceService
    participant ALERT as AlertService
    participant CM as ConfigMaps
    participant NOTIFY as Notifier

    loop Every Hour
        SCHED->>SCHED: Trigger billing job

        par Query all teams
            SCHED->>OC: GET /allocation?window=1h
            OC-->>SCHED: Namespace costs
        end

        loop For each team
            SCHED->>BILL: Calculate team cost
            BILL->>BAL: Get current balance
            BAL->>CM: Read team-balances
            CM-->>BAL: Balance data
            BAL-->>BILL: Current balance

            BILL->>BILL: Compute charges
            BILL->>BAL: Deduct amount
            BAL->>CM: Update team-balances
            BILL->>CM: Log to audit

            alt Balance < Threshold
                BILL->>ALERT: Trigger alert
                ALERT->>CM: Log alert
                ALERT->>NOTIFY: Send notification
            end

            alt Balance <= 0
                BILL->>BILL: Mark team suspended
            end
        end
    end
```

### Team Creation Flow

```mermaid
sequenceDiagram
    autonumber
    participant UI as Web UI
    participant API as API Server
    participant TS as TenantService
    participant K8S as Kubernetes
    participant CAP as Capsule

    UI->>API: POST /api/v1/teams
    API->>API: Validate request
    API->>TS: CreateTeam(teamData)

    TS->>K8S: Create Capsule Tenant
    K8S->>CAP: Reconcile Tenant
    CAP-->>K8S: Tenant Ready

    TS->>K8S: Create ConfigMap entry
    K8S-->>TS: ConfigMap updated

    TS-->>API: Team created
    API-->>UI: 201 Created
```

### Project Namespace Lifecycle

```mermaid
sequenceDiagram
    autonumber
    participant UI as Web UI
    participant API as API Server
    participant PS as ProjectService
    participant K8S as Kubernetes
    participant CAP as Capsule

    UI->>API: POST /api/v1/projects
    API->>PS: CreateProject(projectData)

    PS->>K8S: Create Namespace with labels
    Note over K8S: capsule.clastix.io/tenant: team-name

    K8S->>CAP: Validate tenant ownership
    CAP-->>K8S: Approved

    PS->>K8S: Apply ResourceQuota
    PS->>K8S: Apply NetworkPolicy

    PS-->>API: Project created
    API-->>UI: 201 Created
```

---

## Integration Points

### Capsule Integration

```mermaid
graph LR
    subgraph "Bison"
        TS[TenantService]
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

    TS -->|Create/Update| TEN
    CTRL -->|Watch| TEN
    CTRL -->|Reconcile| NS
    CTRL -->|Apply| RQ & LR
```

**Tenant CRD Mapping:**

| Bison Concept | Capsule Resource |
|---------------|-----------------|
| Team | Tenant |
| Project | Namespace (within Tenant) |
| Team Owners | Tenant Owners (OIDC groups) |
| Resource Quota | Tenant ResourceQuota |
| Node Binding | Tenant NodeSelector |

### OpenCost Integration

```mermaid
graph LR
    subgraph "Bison"
        CS[CostService]
        RS[ReportService]
    end

    subgraph "OpenCost"
        API[Allocation API<br/>:9003/allocation]
        UI[OpenCost UI<br/>:9090]
    end

    subgraph "Prometheus"
        PROM[Prometheus Server]
        METRICS[Container Metrics]
    end

    CS -->|GET /allocation| API
    RS -->|GET /allocation| API
    API -->|Query| PROM
    PROM -->|Scrape| METRICS
```

**OpenCost API Usage:**

```bash
# Query hourly costs by namespace
GET /allocation?window=1h&aggregate=namespace

# Response structure
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

## Deployment Architecture

### Kubernetes Resources

```mermaid
graph TB
    subgraph "bison-system namespace"
        subgraph "API Server"
            DEP1[Deployment<br/>replicas: 2]
            SVC1[Service<br/>ClusterIP]
            ING1[Ingress]
        end

        subgraph "Web UI"
            DEP2[Deployment<br/>replicas: 2]
            SVC2[Service<br/>ClusterIP]
            ING2[Ingress]
        end

        subgraph "Data Storage"
            CM1[ConfigMap<br/>bison-billing-config]
            CM2[ConfigMap<br/>bison-team-balances]
            CM3[ConfigMap<br/>bison-auto-recharge]
            CM4[ConfigMap<br/>bison-audit-logs]
            SEC[Secret<br/>bison-auth]
        end

        subgraph "RBAC"
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

### High Availability

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Ingress Controller]
    end

    subgraph "API Server Pool"
        API1[API Pod 1]
        API2[API Pod 2]
        API3[API Pod N]
    end

    subgraph "Web UI Pool"
        WEB1[Web Pod 1]
        WEB2[Web Pod 2]
    end

    subgraph "Shared State"
        CM[ConfigMaps<br/>etcd backed]
    end

    LB --> API1 & API2 & API3
    LB --> WEB1 & WEB2
    API1 & API2 & API3 --> CM
```

---

## Security Model

### Authentication & Authorization

```mermaid
sequenceDiagram
    participant USER as User
    participant UI as Web UI
    participant API as API Server
    participant AUTH as Auth Middleware

    USER->>UI: Login request
    UI->>API: POST /api/v1/auth/login
    API->>AUTH: Validate credentials
    AUTH-->>API: Generate JWT
    API-->>UI: JWT Token

    USER->>UI: Access resource
    UI->>API: GET /api/v1/teams<br/>Authorization: Bearer JWT
    API->>AUTH: Validate JWT
    AUTH-->>API: Claims extracted
    API-->>UI: Resource data
```

### RBAC Permissions

```mermaid
graph TD
    subgraph "ClusterRole: bison-api"
        P1[configmaps: CRUD]
        P2[namespaces: CRUD]
        P3[resourcequotas: CRUD]
        P4[pods: get, list, delete]
        P5[tenants.capsule: CRUD]
        P6[nodes: get, list, patch]
    end

    subgraph "Scope"
        S1[Cluster-wide access]
    end

    P1 & P2 & P3 & P4 & P5 & P6 --> S1
```

---

## Technology Stack Summary

| Layer | Technology | Purpose |
|-------|------------|---------|
| Frontend | React 18 + TypeScript | SPA framework |
| UI Components | Ant Design Pro 5 | Enterprise UI |
| Charts | ECharts | Data visualization |
| Backend | Go 1.21 + Gin | REST API |
| K8s Client | client-go | Kubernetes integration |
| Multi-Tenancy | Capsule | Namespace isolation |
| Cost Tracking | OpenCost | Resource billing |
| Metrics | Prometheus | Time-series data |
| Data Storage | ConfigMaps | Persistent state |
| Deployment | Helm 3 | Package management |

---

<p align="center">
  <em>Designed for enterprise-grade GPU resource management</em>
</p>

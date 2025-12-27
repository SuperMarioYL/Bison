# Changelog

All notable changes to the Bison project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1] - 2025-12-27

### 🎉 Initial Release

First official stable release of Bison - GPU Resource Billing & Multi-Tenant Management Platform.

### Added

#### Core Platform
- **Multi-Tenant Management System**
  - Capsule integration for Kubernetes-native multi-tenancy
  - Support for shared and exclusive GPU node pools
  - Dynamic resource quotas (CPU, Memory, GPU)
  - Team-based namespace isolation
  - RBAC integration for access control

- **Real-Time Billing Engine**
  - OpenCost integration for per-pod cost tracking
  - Hourly billing with configurable pricing
  - Support for multiple currencies (USD, CNY, EUR, etc.)
  - Prepaid team balance with auto-deduction
  - Billing configuration API

- **Web Dashboard (React 18)**
  - Cluster overview with real-time metrics
  - Team management interface
  - Project (namespace) management
  - Billing configuration UI
  - Usage reports with CSV export
  - Apple-style design (Big Sur blue #0A84FF)
  - Frosted glass navbar effects
  - Dark mode support
  - Responsive layout

- **REST API (Go 1.24)**
  - Complete CRUD operations for teams/projects
  - Billing and balance management endpoints
  - Usage statistics and reports API
  - Health check and readiness probes
  - Swagger/OpenAPI documentation

- **Alert System**
  - Low balance alerts (warning/critical thresholds)
  - Multi-channel notifications:
    - Generic Webhook
    - DingTalk (钉钉)
    - WeChat Work (企业微信)
  - Auto-suspension when balance ≤ 0

- **Data Persistence**
  - Zero-database architecture
  - All data stored in Kubernetes ConfigMaps
  - Team balances tracking
  - Billing configuration storage
  - Audit logging

#### Deployment & Infrastructure
- **Helm Chart (v0.0.1)**
  - One-command deployment
  - Configurable values for all components
  - High availability support (2+ replicas)
  - Resource requests/limits pre-configured
  - RBAC manifests included

- **Docker Images**
  - Multi-platform support (linux/amd64, linux/arm64)
  - Published to GitHub Container Registry
  - Optimized layer caching
  - Non-root user execution
  - Security scanning with Trivy

- **CI/CD Pipeline (GitHub Actions)**
  - Automated Docker builds on tag push
  - Multi-platform image builds with buildx
  - Helm chart packaging and publishing
  - GitHub Release creation
  - Helm repository updates (GitHub Pages)
  - Documentation deployment

- **Documentation Site (Docusaurus 3.9.2)**
  - Modern, searchable documentation
  - Multi-language support (English, 简体中文)
  - API reference
  - Architecture diagrams (Mermaid)
  - Getting started guides
  - Deployment hosted on GitHub Pages

#### Developer Experience
- **VSCode Integration**
  - Debug configurations for API, Web UI, and Docs
  - Background tasks for development servers
  - Port management to avoid conflicts
  - Comprehensive README in .vscode/

- **Makefile Automation**
  - `make dev` - Run full stack locally
  - `make dev-api` - API server only
  - `make dev-web` - Web UI only
  - `make dev-docs` - Documentation site
  - `make test` - Run all tests
  - `make build` - Build Docker images
  - `make deploy` - Deploy to Kubernetes

- **Development Tools**
  - Hot reload for API (Air)
  - Hot reload for Web UI (Vite HMR)
  - Live documentation preview
  - Linting for Go and TypeScript
  - Test coverage reports

### Documentation

- Comprehensive README with:
  - Architecture diagrams
  - User journey flows
  - Quick start guide
  - Feature highlights
  - UI screenshots
- API documentation (Swagger)
- Deployment guides
- Development setup instructions
- Multi-language support (EN/ZH)

### Technical Specifications

- **Backend:** Go 1.24, Gin framework, client-go
- **Frontend:** React 18, TypeScript, Ant Design 5, Vite
- **Kubernetes:** 1.22+ required
- **Dependencies:** Capsule, OpenCost, Prometheus
- **Storage:** ConfigMaps (etcd-backed)
- **Container Registry:** GitHub Container Registry (ghcr.io)

### Known Limitations

- Basic authentication only (OIDC/SSO support planned for v0.1.0)
- No cost forecasting (historical data only)
- ConfigMap storage (no external database option)
- English UI only (Chinese localization in progress)

### Contributors

- Initial development by Bison Team
- Special thanks to the Capsule and OpenCost communities

---

## [Unreleased]

### Planned for v0.1.0
- OIDC/SSO integration (Keycloak, Okta, Azure AD)
- Chinese UI localization (简体中文)
- Cost forecasting and budget alerts
- Enhanced RBAC with custom roles

### Planned for v0.2.0
- Grafana dashboard templates
- Webhook event streaming
- API rate limiting
- Multi-cluster federation

---

[0.0.1]: https://github.com/SuperMarioYL/Bison/releases/tag/v0.0.1
[Unreleased]: https://github.com/SuperMarioYL/Bison/compare/v0.0.1...HEAD

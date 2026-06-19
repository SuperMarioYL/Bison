# Changelog

All notable changes to the Bison project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.22] - 2026-06-19

### Security — Refuse insecure defaults at startup

- When `AUTH_ENABLED=true`, the server now refuses to start if `JWT_SECRET` is empty or still the built-in public default, or if `ADMIN_PASSWORD` is empty or `admin`. This prevents a production deployment from silently running with a forgeable token-signing key or the well-known default password. Auth-disabled and local development are unaffected; the Helm chart already injects randomly generated, persisted secrets. Added table-driven config validation tests.

## [0.0.21] - 2026-06-19

### Added — Release test/lint gate

- The release workflow now runs a **Test & Lint Gate** before anything is built or published: `go vet`, `gofmt` check, `go build`, `go test -race` (with coverage in the job summary), plus web `npm ci` / lint / `vitest run` / build. `prepare` (and the whole publish chain) `needs` this gate, so broken code can no longer be tagged into a public release.

### Fixed — Reproducible web build

- Declared `tslib` as an explicit dependency: `echarts-for-react` imports it but doesn't declare it, so it was a phantom dependency previously satisfied only by the removed `@ant-design/pro-components`. Clean installs (`npm ci`) now build reliably.
- Synced `package-lock.json` with `package.json` (removed stale `pro-components`, added `tslib`) so `npm ci` works.
- Applied `gofmt` across the api-server (formatting only).

## [0.0.20] - 2026-06-19

### Changed — OpenCost query caching

- The OpenCost client now wraps allocation queries in a 30s TTL cache that also **coalesces concurrent identical queries** (same window/aggregate/filter), so a burst of dashboard/billing requests hits OpenCost once instead of once per caller. Errors are not cached (next caller retries). Self-contained implementation — no new dependency; covered by race-tested unit tests.

## [0.0.19] - 2026-06-19

### Changed — Backend performance

- `GET /teams` no longer issues one (discarded) OpenCost usage query per team; per-team usage is fetched on demand by the detail/dashboard endpoints. This removes an O(teams) OpenCost call storm from the team list.
- Billing/report cost computation now resolves the resource price table **once per operation** (`loadPrices` + `costFromPrices`) instead of reading the resource-config ConfigMap for every allocation row, cutting ConfigMap reads from O(allocations) to O(1) in `ProcessBilling`, `GetTeamBill`, and `GetProjectBill`.

## [0.0.18] - 2026-06-19

### Fixed — Daily-consumption (burn-rate) estimate

- `CalculateDailyConsumption` now divides total in-window deductions by the **actual span of deduction activity** (capped at 7 days, floored at 0.5 day) instead of a fixed 7-day denominator, which previously underestimated the burn rate and overestimated the time-to-overdue.
- Fetches up to 400 history records (was 100) so a full week of hourly deductions isn't truncated and undercounted. Recharges and out-of-window records are correctly excluded. Added unit tests for span, floor, and exclusion behavior.

## [0.0.17] - 2026-06-19

### Fixed — Billing interval correctness & restart safety

- Billing now gates on a persisted `lastBilledAt` timestamp (stored in the billing ConfigMap): a cycle only runs once ~the configured interval has actually elapsed. This stops two failure modes — the hourly scheduler tick over-billing when `interval > 1h`, and a process restart re-billing a window that was already charged.
- The first run on a fresh deployment establishes a baseline instead of billing an unknown historical window.
- The timestamp write uses optimistic-concurrency retry; added a round-trip unit test.

## [0.0.16] - 2026-06-19

### Security — Configurable CORS

- CORS is now configurable via `CORS_ALLOWED_ORIGINS` (comma-separated allowlist). When set, only listed origins are echoed back (with `Vary: Origin` and `Access-Control-Allow-Credentials`); other origins get no `Access-Control-Allow-Origin` and are blocked by the browser. Default (unset) preserves the previous `*` behavior, so existing deployments are unaffected until they opt in to tightening.

## [0.0.15] - 2026-06-19

### Security — Login hardening

- **Per-IP login rate limiting**: after 5 failed attempts within 5 minutes an IP is locked out for 15 minutes (HTTP 429 + `Retry-After`), stopping unthrottled brute-force of the admin password.
- **Constant-time credential comparison** (`crypto/subtle.ConstantTimeCompare`) for both username and password, removing the early-exit timing side channel; both comparisons always run so username validity isn't leaked.
- Added unit tests for the limiter (block threshold, success reset, window reset).

## [0.0.14] - 2026-06-19

### Fixed — Helm secret persistence

- The auth `Secret` now reuses the existing JWT signing key and admin password on `helm upgrade` via `lookup`, instead of regenerating them with `randAlphaNum` on every render. Previously each upgrade rotated the JWT key (invalidating all sessions) and silently changed the admin password. Fresh installs still auto-generate; explicit `auth.admin.password` / `auth.jwt.secret` and `existingSecret` continue to take precedence.

## [0.0.13] - 2026-06-19

### Added — Scheduler leader election

- **Lease-based leader election** (`internal/leader`) guards the singleton billing/auto-recharge/alert scheduler so it runs on exactly one api-server replica at a time. This is the root fix for the duplicate-billing risk; `apiServer.replicaCount` is restored to `2` for HA.
- The scheduler is now **re-startable** (clean `Start`/`Stop` on leadership changes), with new tests covering restart and stop-before-start safety.
- Toggle via `LEADER_ELECTION_ENABLED` (default on); disable for single-replica / local dev.
- Added `coordination.k8s.io/leases` (get/create/update) to the api-server RBAC.

## [0.0.12] - 2026-06-19

### Fixed — Billing correctness & concurrency

- **Atomic balance updates**: `Recharge`, `Deduct`, auto-recharge, and overdue-marking now perform their ConfigMap read-modify-write under `retry.RetryOnConflict`, eliminating silent lost updates / balance corruption under concurrent operations.
- **`Deduct` returns the post-write balance**, so `ProcessBilling` no longer makes a racy second read to decide suspension.
- **Overdue marker preserved across deductions** — a deduction no longer wipes `OverdueAt`, so the grace-period clock is measured from when the balance first went negative (teams now actually suspend after the grace window instead of never).
- **Scheduler hardening**: per-task `panic` recovery (one failing task can no longer crash the api-server) plus startup jitter to avoid multi-replica stampede.
- **Stopgap against double-billing**: `apiServer.replicaCount` defaults to `1` until scheduler leader election lands (the scheduler runs in the api-server; 2 replicas billed tenants twice).

### Changed — Performance

- **K8s client** QPS/Burst raised to 50/100 (from the 5/10 default) so dashboard/billing list bursts are not client-side throttled.
- **Web UI** route-level code splitting + Vite `manualChunks`: echarts (~1 MB) and per-page bundles now load on demand instead of shipping with every Login/Dashboard session.

### Added

- First backend unit tests (`calculateCost`, `Recharge`/`Deduct`, grace-period logic, concurrent-recharge race test) with a fake-clientset optimistic-concurrency harness.
- Top-level React `ErrorBoundary` and a shared `getApiErrorMessage` utility.
- `docs/optimization-roadmap.md` — prioritized continuous-optimization roadmap from a full-codebase audit.

### Changed — Website & docs

- Replaced emoji icons with inline Tabler-style SVG icons; added an interactive vector `ProductShowcase` (Dashboard / Cluster / Reports / Billing).
- Fixed the displayed UI version (was hardcoded `v3.0.0`; now injected from `package.json`).
- Corrected install docs: OCI path `charts/bison`, `/healthz` health check, OpenCost namespace/value keys, `replicaCount`, object names `bison-api`/`bison-web`.
- New 1200×630 social/OG card and SEO metadata; reduced-motion + offscreen-pause for the particle background.
- Removed the unused `@ant-design/pro-components` dependency and stray `console.log`s.

> Note: versions 0.0.2–0.0.11 were release-automation version bumps without dedicated changelog entries.

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

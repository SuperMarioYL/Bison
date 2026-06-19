# Bison 持续优化路线图

> 基于一次覆盖后端、前端、官网、文档、DevOps、测试的综合优化审计整理（多 agent 并行审计 + 对抗式验证，共 89 条发现）。
> 生成时间：2026-06-19 · 审计基线版本 **0.0.11**（`deploy/charts/bison/Chart.yaml`、`web-ui/package.json`、`website/versions.json` 一致；唯一例外是 UI 页脚仍硬编码 `v3.0.0`，本次迭代已修复）。

## 概览

本路线图把 89 条已验证发现归并为 **9 个主题**，按「影响 / 工作量」排序为「快赢 → 战略」两档。整体判断：

- **最危险的是一簇资金正确性 + 并发缺陷**：余额/计费状态存在共享 ConfigMap，read-modify-write 无 ResourceVersion 冲突重试（`api-server/internal/k8s/client.go` 的 `UpdateConfigMap` 为裸 `Update`，全仓无 `RetryOnConflict`），而调度器在 `replicaCount: 2`（`deploy/charts/bison/values.yaml`）下**无 leader election**，每个副本独立跑小时计费/自动充值/告警 —— 客户被按副本数重复扣费、并发充值丢更新。这两条必须最先处理。
- **一组纯文档错误以极低成本拦住所有新用户**：OCI chart 路径写成 `oci://ghcr.io/supermarioyl/bison/bison`，而 CI 实际推送到 `oci://ghcr.io/supermarioyl/charts/bison`；健康检查文档指向不存在的 `/api/v1/health`（实际 `/healthz`）；版本停留 0.0.2/0.0.1；OpenCost 命名空间在 chart 默认（`opencost`）与官网（`opencost-system`）间不一致。
- **官网正在迭代**：emoji 图标、缺失能力展示、缺失截图、版本错印、homepage i18n、社交卡尺寸、SEO 收尾，已抽成独立的「本次迭代清单」。
- **测试是 0**：后端 0 个 `*_test.go`，前端仅 `1+1==2`，CI 测试任务空跑报绿，所有金额逻辑无回归安全网。

---

## 已落地版本（v0.0.12 → v0.0.26，按「每个功能一个小版本」迭代）

| 版本 | 主题 | 内容 |
|---|---|---|
| 0.0.12 | 资金/前端/官网/文档 | ConfigMap 余额 RMW 加 `RetryOnConflict`；`Deduct` 返回写后余额；前端路由 lazy + echarts 分包 + ErrorBoundary；官网去 emoji 换 SVG + ProductShowcase；安装文档纠错；本路线图 |
| 0.0.13 | 资金/并发 | 调度器 **leader election**（Lease），恢复 `replicaCount: 2`，scheduler 可重启 + 测试 |
| 0.0.14 | 安全 | Helm Secret `lookup` 持久化，升级不再轮换 JWT/密码 |
| 0.0.15 | 安全 | 登录 per-IP 限流 + `crypto/subtle` 常量时间比较 + 测试 |
| 0.0.16 | 安全 | `CORS_ALLOWED_ORIGINS` 可配置 allowlist |
| 0.0.17 | 资金 | `lastBilledAt` 时间戳门控，修计费窗口/Interval 不一致 + 防重启重扣 + 测试 |
| 0.0.18 | 资金 | `CalculateDailyConsumption` 分母改实际跨度，修 ~6x 烧钱速率低估 + 测试 |
| 0.0.19 | 性能 | 删 `ListTeams` 丢弃用量循环；`loadPrices` 每次计费读一次配置 |
| 0.0.20 | 性能 | OpenCost 查询 30s TTL + 并发合并缓存 + race 测试 |
| 0.0.21 | DevOps | release 加 **Test & Lint Gate**（vet/fmt/build/test -race + web）；修 `tslib` 幽灵依赖 |
| 0.0.22 | 安全 | auth 开启时启动拒绝默认 JWT/密码 + 测试 |
| 0.0.23 | 前端质量 | 13 处错误提取统一为 `getApiErrorMessage` |
| 0.0.24 | 前端性能 | Auth/Theme Context value `useMemo` |
| 0.0.25 | DevOps | `values.schema.json` 类型校验 + `kubeVersion >=1.22` |
| 0.0.26 | DevOps | 可选 PDB / HPA / NetworkPolicy 模板 |

> 已覆盖 P0 资金正确性与并发、安全基线、开箱即用/CI 门禁、热点性能、chart 健壮性与可用性等全部近期项与多数中期项。

### 仍待办（多需活集群验证或产品决策，建议后续单独排期）

- **后端规模化**：SharedInformer/lister 缓存热点 List、列表分页、per-request 与调度器超时、报表单次聚合查询、`GetCostTrend` 按桶日期映射、suspend/resume 二次缩容修复。
- **前端**：NodeDetail echarts option `useMemo`、逐行 N+1 query 分页门、硬编码色→主题 token、`formatCurrency`/`currencySymbol`、dayjs 统一 bootstrap、i18n 层。
- **安全/供应链**：RBAC 去 `clusterrolebindings` 写、onboarding SSH 入参校验、镜像 Trivy 扫描/SBOM/cosign 签名、基础镜像 digest 固定、Dependabot。
- **架构/平台化（远期）**：余额持久化模型升级（ConfigMap → per-key patch / CRD）、money 整数最小单位、能力补全或下线（OIDC/Email/Excel-PDF）。

---

## 优化主题

### 主题 1 · 资金正确性与并发安全

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| 调度器在每个副本无 leader election（`scheduler.go` 裸 ticker，`replicaCount:2`） | 按副本数重复扣费/重复自动充值/重复告警 —— 资金错误 | 立即将 `values.yaml` replicaCount 设 1 止血；用 `client-go` leaderelection Lease 根治，或拆成单副本 Deployment/CronJob | P0 | M（止血 S） |
| 余额 ConfigMap read-modify-write 无冲突重试（`balance_service.go`、`k8s/client.go`） | 并发充值与计费扣费丢更新，余额静默腐蚀，无审计差额 | `Recharge/Deduct/addRechargeRecord/SetOverdueAt` 包入 `retry.RetryOnConflict`，循环内重读重算重写 | P0 | M |
| ProcessBilling 扣费后独立再读余额做停机判断，且丢弃错误（`billing_service.go` `balance, _ := ...GetBalance`） | 基于陈旧/竞态值错误停机：deployment/statefulset 缩 0、删孤儿 pod | 让 `Deduct` 返回写后余额供判断；停止吞错；与上条乐观锁配合 | P0 | M |
| 计费窗口由 `config.Interval` 决定但 ticker 硬编码 1h（`scheduler.go` vs `billing_service.go`） | Interval≠1 时每小时按 Interval 小时扣费，严重超收 | ticker 周期由 Interval 驱动，或固定查 1h 窗口；加校验；记录 last-billed 时间戳防重启重扣 | P1 | S |
| `CalculateDailyConsumption` 分母取最近 100 条任意类型记录的最旧时间（`balance_service.go`） | 烧钱速率被低估约 6x，欠费预计时间高估 | 分母改为窗口内 deduction 记录的实际跨度（≤7 天 + 下限） | P1 | S |
| suspend/resume 依赖 `original-replicas` 注解，二次缩容丢副本数（`billing_service.go`） | 恢复后工作负载回到 0 副本 | suspend 前查 `team.Suspended` 只缩一次；per-object Update 包 RetryOnConflict | P2 | M |
| `GetCostTrend` 按位置索引映射 OpenCost 日桶（`opencost/client.go`） | 成本错位到错误日期或被静默置 0 | 按每桶 `Window.Start` 解析日期建趋势 | P2 | M |

### 主题 2 · 安全加固

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| 鉴权默认关闭 + admin/admin + 硬编码 JWS 密钥（`config.go`） | 财务+集群控制面开箱即用零鉴权；token 可伪造 | 默认开启鉴权；启动时拒绝内置默认密钥/密码；无密钥时随机生成 | P0 | S |
| `helm upgrade` 每次 `randAlphaNum` 重生成密码与 JWT（`secret.yaml`） | 每次升级踢出所有用户、admin 密码静默改变 | 用 `lookup` 复用已存在 Secret，仅首装生成；可加 `resource-policy: keep` | P0 | S |
| 登录无限流、明文非常量时间比较（`auth.go`） | admin 密码可全速暴力破解 | per-IP 限流 + 退避；`crypto/subtle.ConstantTimeCompare` | P1 | S |
| CORS `Allow-Origin: *` 且允许 Authorization 头（`main.go`） | 任意站点可发起带凭证跨域请求 | 收紧为可配置 allowlist（Bison UI origin） | P2 | S |
| Onboarding/控制面 SSH 配置零校验（`onboarding.go`） | 配合鉴权关闭可在节点/控制面跑攻击者命令 | 校验 host/user 非空、host allowlist/CIDR、端口范围、脚本约束；限管理员可达 | P1 | M |
| API ClusterRole 过权：clusterrolebindings 写、node patch、namespace 增删（`rbac.yaml`） | api-server 被攻破≈集群管理员提权 | 去掉 clusterrolebindings 写（除非必需），node 写与最宽 verb 用 values 开关收口，拆 ClusterRole + per-ns Role | P2 | M |

### 主题 3 · 后端性能与 K8s/OpenCost 访问模式

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| client-go 默认 5 QPS 无 informer 缓存（`k8s/client.go`） | 请求在客户端限流后串行，延迟随集群规模线性增长 | 设 `config.QPS=50`/`Burst=100`；为 namespaces/pods/nodes/tenants 引入 SharedInformerFactory + lister | P1 | L |
| OpenCost 查询无缓存，仪表盘多端点重复查同窗口（`opencost/client.go`、`stats.go`） | 并发用户成倍放大 OpenCost 负载，仪表盘慢 | 加 30-60s TTL 缓存（按 window+aggregate+filter）+ singleflight 合并并发 | P1 | M |
| `TenantService.List` 每团队按 namespace 逐个列 pod（`tenant_service.go`） | `/teams`、`/stats/overview`、`/stats/quota-alerts` 每次 O(团队×ns×pod) | 单次集群级 ListPods 后内存分桶；或 `?usage=true` 让用量按需 | P1 | M |
| `ListTeams` 逐团队算 OpenCost 用量后丢弃（`team.go` `_ = usage`） | 纯浪费的 N 次 OpenCost+tenant 扫描 | 删除该循环，或单次 `GetTeamUsage` 合并入响应 | P1 | S |
| 计费 nsToTeam 逐团队 `ListByTeam` 且吞错（`billing_service.go`） | 列举失败的团队被静默不计费 | 单次 `ListTenants` 用 status.namespaces 构图；记录而非吞错 | P1 | S |
| `calculateCost` 每行重读资源配置（`billing_service.go`） | 每个 allocation 一次 ConfigMap 读 | 每次计费操作读一次配置，预建价格表传入 | P1 | S |
| Summary/团队账单每 namespace 一次 OpenCost 查询（`report_service.go`、`billing_service.go`） | T×P 次串行 HTTP，报表生成随项目数变慢 | 一次 `GetAllocationByNamespace(window)` 建 ns→allocation 映射后内存聚合 | P2 | M |
| 列表无分页（`team.go`/`project.go`/`user.go`） | 大部署返回大而慢的载荷且无法分页 | 仿 `audit.go` 加 page/pageSize + total | P2 | M |
| 无 per-request/调度器超时（`main.go`、`k8s/client.go`） | K8s/OpenCost 卡住时 goroutine 与连接泄漏 | 中间件包 `context.WithTimeout`(20-25s)；设 `rest.Config.Timeout`；调度每轮加超时 | P2 | M |
| 调度器 goroutine 无 panic 恢复/无首跑/无 jitter（`scheduler.go`） | 一次 panic 拖垮整进程；多副本同刻齐发踩踏 | 每任务 defer/recover + 随机 jitter + 启动后首跑 | P1 | S |

### 主题 4 · 前端性能与健壮性

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| 无路由级代码分割，echarts 全量进主包（`App.tsx`、`NodeDetail.tsx`、`vite.config.ts`） | 所有会话首屏都下载只 NodeDetail 用的 ~1MB echarts | 路由 `React.lazy`+`Suspense`；`build.rollupOptions.output.manualChunks` 拆 react/antd/echarts | P1 | M |
| 无 ErrorBoundary，query 读错误未处理 | 渲染异常白屏；读失败页面静默空白/无重试 | 顶层 ErrorBoundary 包 `<Routes>`；共享 `isError/error`→Alert/Result+refetch | P1 | M |
| `@ant-design/pro-components` 声明却零引用（2.7MB） | 拖慢安装、有误入包风险 | 从 `package.json` 移除并更新 CLAUDE.md 表述 | P1 | S |
| ProjectList/TeamList 逐行 N+1 query 无分页门（`ProjectList.tsx`、`TeamList.tsx`） | 50 项目=100 次后端调用/页 | 用分页切片驱动 useQueries，或批量后端端点，至少加 `enabled` | P2 | M |
| Auth/Theme context value 每次渲染新对象（`AuthContext.tsx`、`ThemeContext.tsx`、`main.tsx`） | 所有消费者及整套 antd 主题级联重渲染 | provider value 用 useMemo；`main.tsx` theme token useMemo([isDark]) | P2 | S |
| NodeDetail 每次渲染重建 8+ echarts option（`NodeDetail.tsx`，无 useMemo） | 隐藏 tab 也计算时序 map | 各 option useMemo 按 metric 切片；可 `destroyInactiveTabPane` | P2 | M |
| Dashboard 6 路轮询冗余、全局 staleTime 30s 偏激进 | 后台稳定网络负载 | `refetchIntervalInBackground:false`；合并低频概览轮询；分级 staleTime | P3 | S |
| Dashboard 列定义/渲染闭包每次（含轮询）重建（`Dashboard/index.tsx`） | 每次轮询全表重渲染 | 列数组 useMemo、helper useCallback | P3 | S |

### 主题 5 · 前端质量、可访问性与一致性

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| 错误提取串复制 13 处、无 utils 目录 | 后端错误信封一变需改 13 处 | 加 `src/utils/error.ts` 的 `getApiErrorMessage(err, fallback)` | P2 | S |
| `error:any` 12 处破坏 strict 模式 | 在处理不可信响应的路径上失去类型安全 | 改 `unknown`/`AxiosError<{error?:string}>`；eslint 禁 explicit-any | P2 | S |
| 硬编码十六进制色不随暗色主题（`Dashboard`、`ResourceQuotaInput`、`NodeDetail` echarts） | 暗色下 `#999` 文本/弱阴影/图表轴对比度差 | 用 `theme.useToken()` 或 `var(--*)`；echarts 读 `isDark` 派生轴/文本色 | P2 | M |
| `<a onClick>` 无 href 共 10 处 | 非 tab 可达、无法 Enter/Space、不能新开 | 用 react-router `<Link>` 或 `Typography.Link` | P2 | S |
| Dashboard 硬编码 `$` 而非 `currencySymbol` | CNY/¥ 部署仪表盘显示 $ | 取 billingConfig + 抽 `formatCurrency(value,symbol)` | P2 | S |
| dayjs 插件/locale 每文件重复 bootstrap（3 文件） | 第四个页面易漏配 | 在 `main.tsx`/`lib/dayjs.ts` 统一一次 | P3 | S |
| 残留 `console.log`（`ResourceConfig.tsx`） | 生产控制台泄漏内部数据形状 | 删除；加 eslint `no-console`（留 warn/error） | P3 | S |
| 无 i18n 层，UI 文案全内联中文 | 无法本地化/集中措辞 | 若需多语引 react-i18next；否则至少集中重复标签/toast 前缀 | P3 | L |

### 主题 6 · 文档与安装链路正确性

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| OCI chart 路径错误 `bison/bison`（应 `charts/bison`） | 主推荐安装命令 not found 全失败 | 全量替换为 `oci://ghcr.io/supermarioyl/charts/bison`；加 CI grep 校验与 release.yml 一致 | P0 | S |
| 健康检查文档 `/api/v1/health` 不存在（实际 `/healthz`/`/readyz`） | 首装验证步骤返回 404，误判安装失败 | 改 `curl .../healthz` | P1 | S |
| 版本停留 0.0.2/镜像 0.0.1（实际 0.0.11） | 装到陈旧版本、示例内部不一致 | 改用 VERSION 变量/最新版；release 工作流模板化注入防漂移 | P1 | S |
| intro.md Option A 用 github.io helm repo（CI 从不发布该 index） | 第一个安装方法即失败 | 重写为 OCI；清理别名命令 | P0 | S |
| values 键名虚构：`opencost.url`/`apiServer.replicas`/`auth.oidc`/`clusterName` | `--set` 静默 no-op：OpenCost 错指、计费无数据 | 改正键名；移除/标 roadmap 的 oidc 与 clusterName；加 helm template 校验 CI | P0 | M |
| OpenCost 命名空间不一致：chart 默认 `opencost` vs 官网 `opencost-system` | 默认 URL 触不到文档安装位置，计费显示 0 | 统一为 chart 默认 `opencost` | P1 | S |
| features.md 夸大未实现能力（OIDC/Email/Excel-PDF/插件） | 卖点接触即失败、生支持工单 | 标 planned 或移除，对齐 README 真实清单 | P1 | M |
| 配置默认虚构（`BILLING_INTERVAL=10m`、`auth.admin.password=admin`） | 运维误以为 10m 节奏与默认密码 | 删/实现该 env；密码默认改空/自动生成；逐列对照 values.yaml | P2 | S |
| CHANGELOG 停在 0.0.1、installation.md 对象名错（`bison-api-server`/`bison-webui`，实际 `bison-api`/`bison-web`）、孤儿 `docs/architecture.html` | 误导与维护腐烂 | 回填 0.0.6-0.0.11；改正对象名；删 html 或入构建 | P2 | S-M |

### 主题 7 · 官网本次迭代

详见下方独立「网站本次迭代清单」章节。

### 主题 8 · DevOps / 发布管线 / 部署可用性

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| release 工作流无测试/lint 门（`release.yml`，仅 helm lint） | 破损代码可被打成正式 Release | 加 gating job 跑 `go test -race`/`vet`/`npm lint`/`npm test`（`workflow_call` 复用），build-and-push `needs:` 它 | P0 | S |
| web-ui nginx envsubst 失效 + 只读根 FS 会 CrashLoop（`nginx.conf`、`values.yaml`、`web-deployment.yaml`） | 默认生产配置下 Web UI 无法启动/连后端 | 改 `nginx-unprivileged`/listen 8080 + `pid /tmp` + emptyDir 挂载；`${API_BASE_URL}` 走 templates | P0 | M |
| helm upgrade 轮换密钥（见主题 2，列此联动） | 升级即登出全员 | `lookup` 复用 Secret | P0 | S |
| Dockerfile/CI 删 package-lock 再 install（`web-ui/Dockerfile`、`build-test.yml`） | 构建不可复现、缓存失效 | 修根因后全用 `npm ci` | P2 | S |
| 镜像无扫描/SBOM/签名（`release.yml` 已声明 id-token 却未接） | 漏洞基础层流向用户、无法验真 | build-push 加 `provenance/sbom`，cosign 无密钥签名，PR 跑非阻断 Trivy | P2 | M |
| 缺 PDB/HPA/NetworkPolicy（replicaCount:2） | 同时驱逐两副本致停机 | 加可选 PDB(minAvailable:1)/HPA/NetworkPolicy，values 开关 | P2 | M |
| 基础镜像未按 digest 固定，alpine:3.19 近 EOL | OS 层不可复现、积累 CVE | 按 `@sha256:` 固定并升 alpine；Renovate/Dependabot 维护；api-server 考虑 distroless | P2 | S |
| GitHub Actions 用 `latest`/浮动 major | CI 行为非确定、供应链风险 | helm 固定具体版本、actions 钉 SHA；启 github-actions Dependabot | P3 | S |
| Chart 无 kubeVersion 与 values.schema.json | 不支持集群晚失败、values 拼写静默忽略 | 加 `kubeVersion:'>=1.22.0-0'` 与 values.schema.json；`helm lint --strict` | P3 | S |
| api-server Dockerfile `COPY . .` 无 .dockerignore | 构建上下文大、禁用 Go 缓存 | 加 .dockerignore；去 `-a -installsuffix cgo`；BuildKit cache mount | P3 | S |

### 主题 9 · 测试覆盖

| 问题 | 影响 | 建议 | 优先级 | 工作量 |
|---|---|---|---|---|
| 后端 0 个 `*_test.go`，所有金额逻辑无测试 | 符号/边界/重构错误可静默错账上线 | 表驱动单测 + fake clientset，优先 `calculateCost`/`Recharge`/`Deduct`/`isGracePeriodExpired`/`CalculateDailyConsumption` | P0 | L（首批 M） |
| CI 测试任务 vacuous 通过报绿 0% 覆盖（`build-test.yml`） | 对评审制造虚假信心 | 加最低覆盖门；检测关键包零测试文件即失败 | P0 | S |
| 并发扣费/充值无 `-race` 测试 | 丢更新竞态不可捕获 | N 并发 Recharge/Deduct 对 fake clientset 断言终值==操作和 | P1 | M |
| 计费 deduct+复读 分支、grace/suspend 单位逻辑无测试 | 错阈值/单位错停付费租户工作负载 | 覆盖正余额/grace 内/过 grace(hours&days)/恢复 四分支 | P1 | M |
| recharge 校验仅在 handler binding，`SetAutoRechargeConfig` 无正值检查 | 负自动充值额每 tick 静默扣费 | 加正值检查 + 边界/NaN 测试 | P1 | S |
| 前端仅占位测试，BillingConfig/api.ts/recharge 0% | money-facing UI 无输入校验/格式/错误保护 | 删 example.test 换真测试 + vitest 覆盖门 | P2 | L |

---

## Top 优先级

> 排序原则：先「低工作量、止血资金/安全/可用性」的快赢，再「打基础」的战略项。

**快赢档（P0，可立即排期）**

1. **调度器 leader election（先 replicaCount=1 止血）** —— 消除按副本数重复扣费。止血 S，根治 M。
2. **ConfigMap 余额包 `RetryOnConflict` + 乐观并发** —— 杜绝余额丢更新/腐蚀，是金额准确性的地基。M。
3. **修正安装文档致命错误**（OCI `charts/bison`、`/healthz`、版本 0.0.11、OpenCost 命名空间/键名）。S。
4. **鉴权安全基线**（默认开启 + 拒绝默认密钥/密码 + helm `lookup` 持久化 Secret + 登录限流 + CORS 收紧）。S。
5. **修复 web-ui 部署可用性**（nginx envsubst + 只读根 FS → unprivileged/8080 + emptyDir）。M。
6. **release 工作流加测试/lint 门 + CI 覆盖门**。S。
7. **官网本次迭代快赢三连**（修 `v3.0.0` 版本号 → emoji 改 Tabler SVG → 社交卡 1200x630）。S-M。

**战略档（P1，打基础）**

8. **首批后端单元测试**（`calculateCost` 最高价值，叠加 `Recharge`/`Deduct`/计费分支）。M。
9. **前端 `React.lazy` + Vite `manualChunks` 拆 echarts，并加顶层 ErrorBoundary**。M。
10. **K8s client QPS/Burst + 热点 List informer 缓存 + OpenCost 短 TTL+singleflight 缓存**。L。

---

## 网站本次迭代清单

> 官网（Docusaurus，`website/`，线上 `bison.lei6393.com`）。第 1 项是第 7 项截图的前置条件。✅ = 本次迭代已落地。

1. ✅ **修正版本号（前置项）**：`web-ui/src/layouts/BasicLayout.tsx` 硬编码的 `v3.0.0` 改为从 `package.json` 经 Vite `define`（`__APP_VERSION__`）注入。
2. ✅ **emoji 图标改 Tabler 内联 SVG**：`HomepageFeatures` 的 🔐💰📊🚀⚡🎯 改为 `Icons/` 内联 SVG（shield-lock / currency-dollar / dashboard / rocket / bolt / shield-check），`stroke=currentColor` 适配暗色。
3. ✅ **UseCases / hero emoji 替换**：🤖🏢💵 与 ❌/✅/→ 改内联 SVG；hero 按钮去 🚀/⭐。
4. ✅ **Homepage i18n**：6 张 feature 卡 + showcase 标签包 `<Translate>` 并补 `i18n/zh-Hans/code.json` 中文。
5. ✅ **补内联 SVG 图表/示意图**：新增 `ProductShowcase` 组件（资源总览/集群节点/报表中心/计费配置 四屏全矢量渲染），呼应「多用 SVG 图栏展示功能」。
6. **刷新陈旧内容**：在 `features.md` 与 `HomepageFeatures` 增补已发布但缺失的能力——「集群与节点管理」「自动化节点 Onboarding」；校正 features.md 夸大项。
7. **补缺失截图**：本次以矢量 `ProductShowcase` 替代（无可用集群拍摄真实截图）；待有环境后补 Reports/Audit/Cluster/Project/Settings/Login 真实截图。
8. **修正官网安装文档**：`installation.md`/`intro.md` 的 OCI 路径、OpenCost 命名空间、values 键名、对象名、版本号。
9. **社交 / OG 卡**：换 1200×630，`docusaurus.config.ts` 补 `twitter:card=summary_large_image` 与 `og:image:width/height`。
10. **SEO 收尾**：确认 google/baidu 验证标签、robots.txt、sitemap 完整；清理默认脚手架（blog 样例、plushie banner、boilerplate README）。
11. **ParticleBackground 可访问性**：加 `prefers-reduced-motion` 短路与离屏暂停 rAF。

---

## 后续持续优化方向

### 近期（本季度，止血与开箱即用）

- **资金正确性收口**：完成 leader election + ConfigMap 乐观并发 + `Deduct` 返回写后余额 + 计费窗口/Interval 校验 + `CalculateDailyConsumption` 分母修复，让一笔钱在任何并发与重启下都不丢、不重复、不基于陈旧值停机。
- **安全基线**：鉴权默认开启 + 启动拒绝默认密钥 + helm Secret 持久化 + 登录限流 + CORS allowlist。
- **开箱即用**：修复 web-ui nginx/只读根 FS CrashLoop；全量修正安装文档（OCI 路径、healthz、版本、values 键名、OpenCost 命名空间）。
- **CI 门禁**：release 工作流接测试/lint 门 + 最低覆盖门，并补首批后端金额单测（`calculateCost`/`Recharge`/`Deduct`/计费分支 + `-race` 并发测试）。
- **官网本次迭代**：执行上方 11 项清单。
- **快赢清理**：删 `@ant-design/pro-components`、去 `ListTeams` 丢弃用量循环、计费 `calculateCost` 配置改读一次。

### 中期（规模化与体验）

- **后端规模化**：K8s client QPS/Burst + SharedInformer/lister 缓存热点 List；OpenCost 短 TTL + singleflight 缓存；计费/Summary 改单次 `GetAllocationByNamespace` 聚合；列表加分页；per-request 与调度器超时；调度器 panic 恢复 + jitter。
- **前端体验**：路由 lazy + manualChunks；顶层 ErrorBoundary + 共享 query 错误 UI；逐行 N+1 query 分页门/批量端点；Context value 与 Dashboard 列/闭包 memo 化；NodeDetail echarts option useMemo；轮询节流。
- **前端一致性**：`getApiErrorMessage` 工具 + `error: unknown` + eslint 禁 any；硬编码色改主题 token（含 echarts 暗色）；`<a onClick>` 改 `<Link>`；`formatCurrency` + 统一 `currencySymbol`；dayjs 统一 bootstrap；去 console.log。
- **DevOps 供应链**：镜像扫描(Trivy)/SBOM/cosign 签名；npm ci 复现构建；基础镜像 digest 固定 + alpine 升级；PDB/HPA/NetworkPolicy；Chart kubeVersion + values.schema.json。
- **文档治理**：CHANGELOG 回填并在 release 强制每 tag 有条目；features.md 与代码对齐；删孤儿 architecture.html；配置默认逐列对照 values.yaml。
- **测试纵深**：补 `ProcessBilling` 分支、suspend/resume、grace 单位、自动充值正值校验测试；前端 api 拦截器/BillingConfig/recharge 表单测试 + vitest 覆盖门。

### 远期（架构与平台化）

- **持久化模型升级**：评估将余额/计费历史从单一共享 ConfigMap 迁移到 per-team key + patch 语义，或引入轻量嵌入式存储/CRD，从根上消除 read-modify-write 竞态与单对象写放大；money 字段考虑整数最小单位避免 float64 精度问题。
- **RBAC 最小权限**：拆分 ClusterRole + per-namespace Role，去除 clusterrolebindings 写，node 写按 values 开关。
- **可观测性与多副本水平扩展**：在 leader election + informer 缓存到位后，让 API 真正多副本水平扩展，配 HPA/PDB/NetworkPolicy 与请求级 metrics/trace。
- **i18n 平台化**：若多语言成为目标，引入 react-i18next 与 zh-CN 目录，逐步替换内联文案。
- **能力补全或下线**：对 features.md 中标为 planned 的 OIDC/SSO、Email/SMTP 告警、Excel/PDF 导出、插件化计费规则，按路线图实现或正式从对外材料移除，保持「文档=能力」。

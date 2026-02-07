import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
});

// Request interceptor
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('username');
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

// Auth APIs
export interface AuthStatus {
  authEnabled: boolean;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  expiresAt: number;
  username: string;
}

export const getAuthStatus = () => api.get<AuthStatus>('/auth/status');
export const login = (data: LoginRequest) => api.post<LoginResponse>('/auth/login', data);

// Feature flags
export interface Features {
  costEnabled: boolean;
  capsuleEnabled: boolean;
  prometheusEnabled: boolean;
}

export const getFeatures = () => api.get<Features>('/features');

// Cluster Resources (dynamic)
export interface ResourceType {
  name: string;
  displayName: string;
  unit: string;
  capacity: number;
  allocatable: number;
}

export const getClusterResources = () => 
  api.get<{ items: ResourceType[] }>('/cluster/resources');

// Team APIs (Capsule Tenants) - Dynamic Quota
export interface TeamStatus {
  ready: boolean;
  namespaces: number;
  state: string;
}

export interface OwnerRef {
  kind: 'User' | 'Group';
  name: string;
}

export type TeamMode = 'shared' | 'exclusive';

export interface Team {
  name: string;
  displayName: string;
  description?: string;
  owners: OwnerRef[];
  mode: TeamMode;
  exclusiveNodes?: string[];
  nodeSelector?: Record<string, string>;
  quota: Record<string, string>; // Dynamic quota
  quotaUsed?: Record<string, string>; // Quota usage
  projectCount: number;
  status?: TeamStatus;
  suspended?: boolean;
}

export interface TeamWithUsage {
  team: Team;
  usage?: UsageData;
}

export const getTeams = () => 
  api.get<{ items: Team[] }>('/teams');
export const getTeam = (name: string, window = '7d') => 
  api.get<TeamWithUsage>(`/teams/${name}`, { params: { window } });
export const createTeam = (team: Omit<Team, 'projectCount' | 'status' | 'suspended'>) => 
  api.post('/teams', team);
export const updateTeam = (name: string, team: Partial<Team>) => 
  api.put(`/teams/${name}`, team);
export const deleteTeam = (name: string) => 
  api.delete(`/teams/${name}`);

// Team Balance APIs
export interface Balance {
  teamName: string;
  amount: number;
  lastUpdated: string;
  overdueAt?: string;           // When balance first went negative
  estimatedOverdueAt?: string;  // Predicted time when balance will go negative
  dailyConsumption?: number;    // Average daily consumption
  graceRemaining?: string;      // Remaining grace period (e.g., "2天 3小时")
}

export interface RechargeRecord {
  id: string;
  timestamp: string;
  type: 'recharge' | 'deduction' | 'auto_recharge';
  amount: number;
  operator: string;
  reason?: string;
  balance: number;
}

export interface AutoRechargeConfig {
  enabled: boolean;
  amount: number;
  schedule: 'weekly' | 'monthly';
  dayOfWeek?: number;
  dayOfMonth?: number;
  nextExecution: string;
  lastExecuted?: string;
}

export const getTeamBalance = (name: string) =>
  api.get<Balance>(`/teams/${name}/balance`);
export const rechargeTeam = (name: string, data: { amount: number; remark?: string; operator?: string }) =>
  api.post(`/teams/${name}/recharge`, data);
export const getRechargeHistory = (name: string) =>
  api.get<{ items: RechargeRecord[] }>(`/teams/${name}/balance/history`);
export const getTeamBill = (name: string, window = '7d') =>
  api.get(`/teams/${name}/bill`, { params: { window } });
export const getAutoRechargeConfig = (name: string) =>
  api.get<AutoRechargeConfig>(`/teams/${name}/auto-recharge`);
export const updateAutoRechargeConfig = (name: string, config: AutoRechargeConfig) =>
  api.put(`/teams/${name}/auto-recharge`, config);
export const suspendTeam = (name: string) =>
  api.post(`/teams/${name}/suspend`);
export const resumeTeam = (name: string) =>
  api.post(`/teams/${name}/resume`);

// Project APIs (Namespaces)
export interface ProjectMember {
  user: string;
  role: 'admin' | 'edit' | 'view';
}

export interface Project {
  name: string;
  team: string;
  displayName: string;
  description?: string;
  members?: ProjectMember[];
  status: string;
}

export interface ProjectWithUsage {
  project: Project;
  usage?: UsageData;
}

// Dynamic resource usage (from system resource config)
export interface ResourceUsage {
  name: string;        // K8s resource name (cpu, memory, nvidia.com/gpu)
  displayName: string; // Display name from config
  unit: string;        // Display unit from config
  used: number;        // Current usage (after divisor applied)
  rawUsed: number;     // Raw usage value
}

export interface ProjectUsage {
  projectName: string;
  resources: ResourceUsage[];
}

export const getProjects = (team?: string) => 
  api.get<{ items: Project[] }>('/projects', { params: { team } });
export const getProject = (name: string, window = '7d') => 
  api.get<ProjectWithUsage>(`/projects/${name}`, { params: { window } });
export const createProject = (project: Omit<Project, 'status'>) => 
  api.post<Project>('/projects', project);
export const updateProject = (name: string, project: Partial<Project>) => 
  api.put<Project>(`/projects/${name}`, project);
export const getProjectUsage = (name: string) =>
  api.get<ProjectUsage>(`/projects/${name}/usage`);
export const deleteProject = (name: string) => 
  api.delete(`/projects/${name}`);

// Project Workloads APIs
export interface WorkloadSummary {
  deployments: number;
  statefulSets: number;
  pods: number;       // Orphan pods
  jobs: number;
  cronJobs: number;
  totalPods: number;  // Total pods including controller-managed
}

export interface Workload {
  kind: string;       // Deployment, StatefulSet, Pod, Job, CronJob
  name: string;
  namespace: string;
  replicas: number;   // Desired replicas
  ready: number;      // Ready replicas
  status: string;     // Running, Pending, Failed, etc.
  image?: string;     // Main container image
  createdAt: string;  // ISO 8601 timestamp
}

export const getProjectWorkloads = (name: string) =>
  api.get<{ items: Workload[] }>(`/projects/${name}/workloads`);
export const getProjectWorkloadSummary = (name: string) =>
  api.get<WorkloadSummary>(`/projects/${name}/workloads/summary`);

// User APIs
export interface User {
  email: string;           // Unique identifier
  displayName: string;     // Display name
  source: 'manual' | 'oidc'; // Source
  status: 'active' | 'disabled'; // Status
  createdAt: string;       // ISO 8601 timestamp
  lastLogin?: string;      // ISO 8601 timestamp
}

export interface UserTeamDetail {
  teamName: string;
  displayName: string;
  role: string;
  joinedAt?: string;
}

export interface UserProjectDetail {
  projectName: string;
  displayName: string;
  teamName: string;
  role: 'admin' | 'edit' | 'view';
}

export interface UserDetail extends User {
  teams: UserTeamDetail[];
  projects: UserProjectDetail[];
  usage?: UsageData;
}

// User CRUD APIs
export const getUsers = (query?: string, status?: string, source?: string) =>
  api.get<{ items: User[] }>('/users', { params: { q: query, status, source } });
export const getUser = (email: string) =>
  api.get<UserDetail>(`/users/${encodeURIComponent(email)}`);
export const createUser = (data: { email: string; displayName?: string; status?: string; initialTeam?: string }) =>
  api.post<User>('/users', data);
export const updateUser = (email: string, data: { displayName?: string; status?: string }) =>
  api.put<User>(`/users/${encodeURIComponent(email)}`, data);
export const deleteUser = (email: string) =>
  api.delete(`/users/${encodeURIComponent(email)}`);
export const setUserStatus = (email: string, status: string) =>
  api.put(`/users/${encodeURIComponent(email)}/status`, { status });
export const getUserUsage = (email: string, window = '7d') =>
  api.get<UsageData>(`/users/${encodeURIComponent(email)}/usage`, { params: { window } });

// User-Team Association APIs
export const addUserToTeam = (email: string, teamName: string) =>
  api.post(`/users/${encodeURIComponent(email)}/teams`, { teamName });
export const removeUserFromTeam = (email: string, teamName: string) =>
  api.delete(`/users/${encodeURIComponent(email)}/teams/${teamName}`);

// User-Project Association APIs
export const addUserToProject = (email: string, projectName: string, role: string) =>
  api.post(`/users/${encodeURIComponent(email)}/projects`, { projectName, role });
export const removeUserFromProject = (email: string, projectName: string) =>
  api.delete(`/users/${encodeURIComponent(email)}/projects/${projectName}`);
export const updateUserProjectRole = (email: string, projectName: string, role: string) =>
  api.put(`/users/${encodeURIComponent(email)}/projects/${projectName}/role`, { role });

// Usage/Cost APIs (OpenCost)
export interface UsageData {
  name: string;
  cpuCoreHours: number;
  ramGBHours: number;
  gpuHours: number;
  totalCost: number;
  cpuCost: number;
  ramCost: number;
  gpuCost: number;
  minutes: number;
}

export interface UsageReport {
  window: string;
  aggregateBy: string;
  data: UsageData[];
  totalCost: number;
}

export const getTeamUsage = (window = '7d') => 
  api.get<UsageReport>('/stats/usage/teams', { params: { window } });
export const getProjectsUsageReport = (window = '7d') => 
  api.get<UsageReport>('/stats/usage/projects', { params: { window } });
export const getUserUsageReport = (window = '7d') => 
  api.get<UsageReport>('/stats/usage/users', { params: { window } });

// Quota alert types
export interface QuotaAlert {
  type: 'team' | 'project';
  name: string;
  displayName?: string;
  resource: string;
  used: string;
  limit: string;
  usagePercent: number;
}

// Cost trend types
export interface CostTrendPoint {
  date: string;
  totalCost: number;
}

// Top consumer types
export interface TopConsumer {
  type: 'team' | 'project';
  name: string;
  displayName?: string;
  totalCost: number;
  cpuHours: number;
  memoryGBH: number;
  gpuHours: number;
}

export const getQuotaAlerts = (threshold = 80) =>
  api.get<{ items: QuotaAlert[] }>('/stats/quota-alerts', { params: { threshold } });
export const getCostTrend = (window = '7d') =>
  api.get<{ items: CostTrendPoint[] }>('/stats/cost-trend', { params: { window } });
export const getTopConsumers = (window = '7d', limit = 5) =>
  api.get<{ items: TopConsumer[] }>('/stats/top-consumers', { params: { window, limit } });
export const getCostStatus = () => 
  api.get<{ enabled: boolean }>('/stats/cost-status');

// Stats APIs
export interface ResourceSummary {
  name: string;
  capacity: number;
  allocatable: number;
}

export interface ArchSummary {
  arch: string;
  count: number;
}

export interface Overview {
  totalNodes: number;
  totalTeams: number;
  totalProjects: number;
  resources: ResourceSummary[];
  nodesByArch: ArchSummary[];
  nodesByStatus: Record<string, number>;
  costEnabled: boolean;
}

export const getOverview = () => api.get<Overview>('/stats/overview');

// Cluster APIs
export interface NodeResource {
  name: string;
  capacity: number;
  allocatable: number;
}

export interface ClusterNode {
  name: string;
  arch: string;
  os: string;
  ready: boolean;
  labels: Record<string, string>;
  resources: NodeResource[];
}

export interface NodeAddress {
  type: string;
  address: string;
}

export interface NodeCondition {
  type: string;
  status: string;
  reason: string;
  message: string;
}

export interface NodeTaint {
  key: string;
  value?: string;
  effect: string;
}

export interface NodeInfo {
  kernelVersion: string;
  osImage: string;
  containerRuntimeVersion: string;
  kubeletVersion: string;
  architecture: string;
  operatingSystem: string;
}

export interface NodeDetail {
  name: string;
  arch: string;
  os: string;
  ready: boolean;
  labels: Record<string, string>;
  taints: NodeTaint[];
  nodeInfo: NodeInfo;
  addresses: NodeAddress[];
  resources: NodeResource[];
  conditions: NodeCondition[];
}

export interface NodePod {
  name: string;
  namespace: string;
  status: string;
  ip: string;
  cpuRequest: number;
  memoryRequest: number;
  restarts: number;
}

export const getClusterNodes = (arch?: string) => 
  api.get<{ items: ClusterNode[] }>('/cluster/nodes', { params: { arch } });
export const getClusterNode = (name: string) => 
  api.get<NodeDetail>(`/cluster/nodes/${name}`);
export const getNodePods = (name: string) => 
  api.get<{ items: NodePod[] }>(`/cluster/nodes/${name}/pods`);
export const updateNodeLabels = (name: string, labels: Record<string, string>) =>
  api.put(`/cluster/nodes/${name}/labels`, { labels });
export const updateNodeTaints = (name: string, taints: NodeTaint[]) =>
  api.put(`/cluster/nodes/${name}/taints`, { taints });

// Billing Config APIs
export interface ResourcePrice {
  price: number;
  unit: string;
}

export interface BillingConfig {
  enabled: boolean;
  interval: number;
  currency: string;
  currencySymbol: string;
  pricing: Record<string, ResourcePrice>;
  gracePeriodValue?: number;  // Grace period value (e.g., 3)
  gracePeriodUnit?: string;   // Grace period unit: "hours" or "days"
}

export const getBillingConfig = () => 
  api.get<BillingConfig>('/settings/billing');
export const updateBillingConfig = (config: BillingConfig) =>
  api.put('/settings/billing', config);

// Alert Config APIs
export interface NotifyChannel {
  id: string;
  type: 'email' | 'webhook' | 'dingtalk' | 'wechat';
  name: string;
  config: Record<string, string>;
  enabled: boolean;
}

export interface AlertConfig {
  balanceThreshold: number;
  channels: NotifyChannel[];
}

export interface Alert {
  id: string;
  timestamp: string;
  type: string;
  severity: 'info' | 'warning' | 'critical';
  target: string;
  message: string;
  sent: boolean;
  sentAt?: string;
  channels?: string[];
}

export const getAlertConfig = () =>
  api.get<AlertConfig>('/settings/alerts');
export const updateAlertConfig = (config: AlertConfig) =>
  api.put('/settings/alerts', config);
export const testAlertChannel = (channel: NotifyChannel) =>
  api.post('/settings/alerts/test', channel);
export const getAlertHistory = (limit = 50) =>
  api.get<{ items: Alert[] }>('/alerts/history', { params: { limit } });

// Audit APIs
export interface AuditLog {
  id: string;
  timestamp: string;
  operator: string;
  action: string;
  resource: string;
  target: string;
  detail?: Record<string, unknown>;
  ip?: string;
  userAgent?: string;
}

export interface AuditPage {
  items: AuditLog[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface AuditFilter {
  action?: string;
  resource?: string;
  operator?: string;
  target?: string;
  from?: string;
  to?: string;
}

export const getAuditLogs = (filter: AuditFilter = {}, page = 1, pageSize = 20) =>
  api.get<AuditPage>('/audit/logs', { params: { ...filter, page, pageSize } });
export const getRecentAuditLogs = (limit = 50) =>
  api.get<{ items: AuditLog[] }>('/audit/recent', { params: { limit } });

// Report APIs
export interface Report {
  type: string;
  name: string;
  window: string;
  generatedAt: string;
  totalCost: number;
  costByResource: Record<string, number>;
  usageSummary?: UsageData;
}

export interface TeamCostRank {
  rank: number;
  teamName: string;
  cost: number;
  percentage: number;
}

export interface SummaryReport {
  window: string;
  generatedAt: string;
  totalCost: number;
  totalTeams: number;
  totalProjects: number;
  topTeams: TeamCostRank[];
}

export const getTeamReport = (name: string, window = '30d') =>
  api.get<Report>(`/reports/team/${name}`, { params: { window } });
export const exportTeamReport = (name: string, window = '30d') =>
  api.get(`/reports/team/${name}/export`, { params: { window, format: 'csv' }, responseType: 'blob' });
export const getProjectReport = (name: string, window = '30d') =>
  api.get<Report>(`/reports/project/${name}`, { params: { window } });
export const exportProjectReport = (name: string, window = '30d') =>
  api.get(`/reports/project/${name}/export`, { params: { window, format: 'csv' }, responseType: 'blob' });
export const getSummaryReport = (window = '30d') =>
  api.get<SummaryReport>('/reports/summary', { params: { window } });
export const exportSummaryReport = (window = '30d') =>
  api.get('/reports/summary/export', { params: { window, format: 'csv' }, responseType: 'blob' });

// Node Management APIs
export type NodeStatus = 'unmanaged' | 'disabled' | 'shared' | 'exclusive';

export interface NodeCondition {
  type: string;
  status: string;
  reason: string;
  message: string;
}

export interface NodeInfo {
  name: string;
  status: NodeStatus;
  team?: string;
  labels: Record<string, string>;
  taints: Array<{ key: string; value: string; effect: string }>;
  conditions: NodeCondition[];
  capacity: Record<string, string>;
  allocatable: Record<string, string>;
  architecture: string;
  os: string;
  kernelVersion: string;
  runtime: string;
  kubeletVersion: string;
  internalIP: string;
  hostname: string;
  podCount: number;
  creationTime: string;
}

export const getNodes = () =>
  api.get<{ items: NodeInfo[] }>('/nodes');
export const getNode = (name: string) =>
  api.get<NodeInfo>(`/nodes/${name}`);
export const getNodeStatusSummary = () =>
  api.get<Record<NodeStatus, number>>('/nodes/summary');
export const getSharedNodes = () =>
  api.get<{ items: NodeInfo[] }>('/nodes/shared');
export const getTeamNodes = (team: string) =>
  api.get<{ items: NodeInfo[] }>(`/nodes/team/${team}`);
export const enableNode = (name: string) =>
  api.post(`/nodes/${name}/enable`);
export const disableNode = (name: string) =>
  api.post(`/nodes/${name}/disable`);
export const assignNodeToTeam = (nodeName: string, teamName: string) =>
  api.post(`/nodes/${nodeName}/assign`, { team: teamName });
export const releaseNode = (name: string) =>
  api.post(`/nodes/${name}/release`);

// System Status APIs
export interface ServiceStatus {
  name: string;
  available: boolean;
  message?: string;
  url?: string;
}

export interface TaskExecution {
  taskName: string;
  startTime: string;
  endTime: string;
  status: 'success' | 'failed' | 'skipped';
  error?: string;
}

export interface SystemStatistics {
  totalTeams: number;
  totalProjects: number;
  totalUsers: number;
  totalNodes: number;
  totalBalance: number;
  suspendedTeams: number;
}

export interface SystemStatus {
  opencost: ServiceStatus;
  capsule: ServiceStatus;
  prometheus: ServiceStatus;
  tasks: TaskExecution[];
  statistics: SystemStatistics;
}

export const getSystemStatus = () =>
  api.get<SystemStatus>('/system/status');
export const getTaskHistory = (limit = 20) =>
  api.get<{ items: TaskExecution[] }>('/system/tasks', { params: { limit } });

// Settings APIs (read-only, configured via Helm)
export interface SystemSettings {
  prometheusUrl: string;
  opencostUrl: string;
}

export const getSettings = () => api.get<SystemSettings>('/settings');

// Node Metrics APIs
export interface PrometheusMetric {
  timestamp: number;
  value: number;
}

export interface LabeledMetricSeries {
  labels: Record<string, string>;
  metrics: PrometheusMetric[];
}

export interface NodeMetrics {
  cpuUsage: PrometheusMetric[];
  memoryUsage: PrometheusMetric[];
  // Network IO
  networkReceive?: PrometheusMetric[];
  networkTransmit?: PrometheusMetric[];
  // RDMA IO
  rdmaReceive?: PrometheusMetric[];
  rdmaTransmit?: PrometheusMetric[];
  // GPU (NVIDIA DCGM)
  gpuUtilization?: PrometheusMetric[];
  gpuMemoryUtil?: PrometheusMetric[];
  gpuPerDevice?: LabeledMetricSeries[];
  // NPU (Huawei Ascend)
  npuUtilization?: PrometheusMetric[];
  npuMemoryUtil?: PrometheusMetric[];
  npuTemperature?: PrometheusMetric[];
}

export const getNodeMetrics = (name: string, params?: {
  hours?: number; hasGpu?: boolean; hasNpu?: boolean;
}) =>
  api.get<NodeMetrics>(`/metrics/node/${name}`, {
    params: { hours: params?.hours ?? 24, hasGpu: params?.hasGpu, hasNpu: params?.hasNpu },
  });

// Resource Configuration APIs
export type ResourceCategory = 'compute' | 'memory' | 'storage' | 'accelerator' | 'other';

export interface ResourceDefinition {
  name: string;          // K8s resource name: cpu, memory, nvidia.com/gpu
  displayName: string;   // Display name: CPU, 内存, NVIDIA GPU
  unit: string;          // Display unit: 核, GiB, 卡
  divisor: number;       // Unit divisor: displayValue = rawValue / divisor
  category: ResourceCategory;  // Category: compute, memory, storage, accelerator, other
  enabled: boolean;      // Whether to show this resource
  sortOrder: number;     // Sort order (lower = first)
  showInQuota: boolean;  // Whether to show in quota settings
  price: number;         // Price per unit per hour
}

export interface DiscoveredResource {
  name: string;
  capacity: number;
  allocatable: number;
  configured: boolean;
}

export const getResourceConfigs = () =>
  api.get<{ items: ResourceDefinition[] }>('/resource-configs');
export const getEnabledResourceConfigs = () =>
  api.get<{ items: ResourceDefinition[] }>('/resource-configs/enabled');
export const getQuotaResourceConfigs = () =>
  api.get<{ items: ResourceDefinition[] }>('/resource-configs/quota');
export const discoverClusterResources = () =>
  api.get<{ items: DiscoveredResource[] }>('/resource-configs/discover');
export const saveResourceConfigs = (items: ResourceDefinition[]) =>
  api.put('/resource-configs', { items });
export const updateResourceConfig = (name: string, config: ResourceDefinition) =>
  api.put(`/resource-configs/${encodeURIComponent(name)}`, config);
export const addResourceConfig = (config: ResourceDefinition) =>
  api.post('/resource-configs', config);

// Node Onboarding APIs
export type OnboardingJobStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
export type SubStepStatus = 'pending' | 'running' | 'success' | 'failed' | 'skipped';

export interface NodePlatform {
  os: string;
  version: string;
  arch: string;
}

export interface SubStep {
  name: string;
  status: SubStepStatus;
  error?: string;
}

export interface OnboardingJob {
  id: string;
  nodeIP: string;
  nodeName?: string;
  platform: NodePlatform;
  status: OnboardingJobStatus;
  currentStep: number;
  totalSteps: number;
  stepMessage: string;
  subSteps?: SubStep[];
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface OnboardingRequest {
  nodeIP: string;
  sshPort?: number;
  sshUsername: string;
  authMethod: 'password' | 'privateKey';
  password?: string;
  privateKey?: string;
}

export const startNodeOnboarding = (data: OnboardingRequest) =>
  api.post<OnboardingJob>('/nodes/onboard', data);
export const getOnboardingJob = (jobId: string) =>
  api.get<OnboardingJob>(`/nodes/onboard/${jobId}`);
export const getOnboardingJobs = () =>
  api.get<{ items: OnboardingJob[] }>('/nodes/onboard');
export const cancelOnboardingJob = (jobId: string) =>
  api.delete(`/nodes/onboard/${jobId}`);

// Control Plane Config APIs
export interface ControlPlaneConfig {
  host: string;
  sshPort: number;
  sshUser: string;
  authMethod: 'password' | 'privateKey';
  hasPassword?: boolean;
  hasPrivateKey?: boolean;
  password?: string;
  privateKey?: string;
}

export const getControlPlaneConfig = () =>
  api.get<ControlPlaneConfig>('/settings/control-plane');
export const updateControlPlaneConfig = (config: ControlPlaneConfig) =>
  api.put('/settings/control-plane', config);
export const testControlPlaneConnection = () =>
  api.post('/settings/control-plane/test');

// Init Scripts APIs
export type ScriptPhase = 'pre-join' | 'post-join';

export interface Script {
  id: string;
  os: string;
  arch: string;
  content: string;
}

export interface ScriptGroup {
  id: string;
  name: string;
  description: string;
  phase: ScriptPhase;
  enabled: boolean;
  order: number;
  builtin: boolean;
  scripts: Script[];
}

export const getInitScripts = () =>
  api.get<{ items: ScriptGroup[] }>('/settings/init-scripts');
export const getInitScript = (id: string) =>
  api.get<ScriptGroup>(`/settings/init-scripts/${id}`);
export const createInitScript = (data: Omit<ScriptGroup, 'id' | 'builtin'>) =>
  api.post<ScriptGroup>('/settings/init-scripts', data);
export const updateInitScript = (id: string, data: Partial<ScriptGroup>) =>
  api.put<ScriptGroup>(`/settings/init-scripts/${id}`, data);
export const deleteInitScript = (id: string) =>
  api.delete(`/settings/init-scripts/${id}`);
export const toggleInitScript = (id: string, enabled: boolean) =>
  api.put(`/settings/init-scripts/${id}/toggle`, { enabled });
export const reorderInitScripts = (ids: string[]) =>
  api.put('/settings/init-scripts/reorder', { ids });

// Configuration Import/Export APIs
export interface ExportConfigData {
  version: string;
  exportedAt: string;
  exportedBy: string;
  sections: Record<string, unknown>;
}

export interface FieldChange {
  current: unknown;
  imported: unknown;
}

export interface ResourceChangeSummary {
  added?: string[];
  modified?: string[];
  removed?: string[];
  unchanged?: string[];
}

export interface SectionPreview {
  present: boolean;
  valid: boolean;
  hasSensitiveData: boolean;
  changes?: Record<string, FieldChange>;
  summary?: ResourceChangeSummary;
  warnings?: string[];
  errors?: string[];
}

export interface ImportPreviewResult {
  valid: boolean;
  version: string;
  exportedAt?: string;
  sections: Record<string, SectionPreview>;
  errors: string[];
  warnings: string[];
}

export interface ImportApplyRequest {
  config: ExportConfigData;
  sections: string[];
  preserveSensitive: boolean;
}

export interface ImportApplyResult {
  message: string;
  applied: string[];
  skipped: string[];
  warnings: string[];
}

export const exportConfig = (params?: { sections?: string; includeSensitive?: boolean }) =>
  api.get('/settings/export', { params, responseType: 'blob' });

export const previewImport = (config: ExportConfigData) =>
  api.post<ImportPreviewResult>('/settings/import/preview', config);

export const applyImport = (data: ImportApplyRequest) =>
  api.post<ImportApplyResult>('/settings/import/apply', data);

export default api;

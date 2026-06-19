import React from 'react';
import { Card, Row, Col, Table, Progress, Typography, Space, Spin, Tag, Empty, List, Button, Tooltip } from 'antd';
import {
  ClusterOutlined,
  TeamOutlined,
  ProjectOutlined,
  DesktopOutlined,
  DollarOutlined,
  WarningOutlined,
  TrophyOutlined,
  RiseOutlined,
  DashboardOutlined,
  ReloadOutlined,
  DatabaseOutlined,
} from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import {
  getOverview,
  getTeamUsage,
  getProjectsUsageReport,
  getEnabledResourceConfigs,
  getQuotaAlerts,
  getCostTrend,
  getTopConsumers,
  ResourceDefinition,
  QuotaAlert,
  TopConsumer,
} from '../../services/api';
import { useFeatures } from '../../hooks/useFeatures';
import PageHeader from '../../components/PageHeader';
import StatCard from '../../components/StatCard';
import DonutChart from '../../components/charts/DonutChart';
import AreaChart from '../../components/charts/AreaChart';

const { Text } = Typography;

// Node status labels and colors
const nodeStatusLabels: Record<string, string> = {
  shared: '共享池',
  exclusive: '独占',
  disabled: '已禁用',
  unmanaged: '未管理',
};

const nodeStatusColors: Record<string, string> = {
  shared: '#0a84ff',
  exclusive: '#34c759',
  disabled: '#ff453a',
  unmanaged: '#c7c7cc',
};

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: features } = useFeatures();

  const {
    data: overview,
    isLoading: overviewLoading,
    dataUpdatedAt,
  } = useQuery({
    queryKey: ['overview'],
    queryFn: () => getOverview().then(res => res.data),
    refetchInterval: 30000,
  });

  const { data: teamUsage, isLoading: usageLoading } = useQuery({
    queryKey: ['teamUsage', '7d'],
    queryFn: () => getTeamUsage('7d').then(res => res.data),
    refetchInterval: 60000,
    enabled: features?.capsuleEnabled !== false && (overview?.costEnabled ?? false),
  });

  const { data: projectUsage, isLoading: projectUsageLoading } = useQuery({
    queryKey: ['projectUsage', '7d'],
    queryFn: () => getProjectsUsageReport('7d').then(res => res.data),
    refetchInterval: 60000,
    enabled: features?.capsuleEnabled !== false && (overview?.costEnabled ?? false),
  });

  // Fetch resource configs for display names and units
  const { data: resourceConfigs } = useQuery({
    queryKey: ['enabledResourceConfigs'],
    queryFn: () => getEnabledResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000,
  });

  // Fetch quota alerts
  const { data: quotaAlertsData } = useQuery({
    queryKey: ['quotaAlerts', 80],
    queryFn: () => getQuotaAlerts(80).then(res => res.data.items),
    refetchInterval: 60000,
    enabled: features?.capsuleEnabled !== false,
  });

  // Fetch cost trend
  const { data: costTrendData } = useQuery({
    queryKey: ['costTrend', '7d'],
    queryFn: () => getCostTrend('7d').then(res => res.data.items),
    refetchInterval: 60000,
    enabled: overview?.costEnabled ?? false,
  });

  // Fetch top consumers
  const { data: topConsumersData } = useQuery({
    queryKey: ['topConsumers', '7d', 5],
    queryFn: () => getTopConsumers('7d', 5).then(res => res.data.items),
    refetchInterval: 60000,
    enabled: overview?.costEnabled ?? false,
  });

  // Get resource config by name
  const getResourceConfig = (name: string): ResourceDefinition | undefined => {
    return resourceConfigs?.find(r => r.name === name);
  };

  // Get display name from config or fallback
  const getDisplayName = (name: string): string => {
    const config = getResourceConfig(name);
    return config?.displayName || name;
  };

  // Format resource value
  const formatResourceValue = (name: string, value: number) => {
    const config = getResourceConfig(name);
    const unit = config?.unit || '';
    const formattedValue = value < 10
      ? value.toFixed(2)
      : value < 1000
        ? value.toFixed(1)
        : Math.round(value).toLocaleString();
    return unit ? `${formattedValue} ${unit}` : formattedValue;
  };

  // Resource usage columns
  const resourceColumns = [
    {
      title: '资源',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => getDisplayName(name),
    },
    {
      title: '容量',
      dataIndex: 'capacity',
      key: 'capacity',
      render: (val: number, record: { name: string }) => formatResourceValue(record.name, val),
    },
    {
      title: '可分配',
      dataIndex: 'allocatable',
      key: 'allocatable',
      render: (val: number, record: { name: string }) => formatResourceValue(record.name, val),
    },
    {
      title: '使用率',
      key: 'usage',
      render: (_: unknown, record: { capacity: number; allocatable: number }) => {
        const used = record.capacity - record.allocatable;
        const percent = record.capacity > 0 ? (used / record.capacity) * 100 : 0;
        return <Progress percent={Math.round(percent)} size="small" />;
      },
    },
  ];

  // Team usage columns
  const usageColumns = [
    {
      title: '团队',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <a onClick={() => navigate(`/teams/${name}`)}>{name}</a>
      ),
    },
    {
      title: 'CPU 时长 (核时)',
      dataIndex: 'cpuCoreHours',
      key: 'cpuCoreHours',
      render: (val: number) => val?.toFixed(2) || '0.00',
    },
    {
      title: '内存时长 (GB时)',
      dataIndex: 'ramGBHours',
      key: 'ramGBHours',
      render: (val: number) => val?.toFixed(2) || '0.00',
    },
    {
      title: 'GPU 时长',
      dataIndex: 'gpuHours',
      key: 'gpuHours',
      render: (val: number) => val?.toFixed(2) || '0.00',
    },
    {
      title: '费用',
      dataIndex: 'totalCost',
      key: 'totalCost',
      render: (val: number) => `$${val?.toFixed(2) || '0.00'}`,
    },
  ];

  // Project usage columns
  const projectUsageColumns = [
    {
      title: '项目',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => (
        <a onClick={() => navigate(`/projects/${name}`)}>{name}</a>
      ),
    },
    {
      title: 'CPU 时长 (核时)',
      dataIndex: 'cpuCoreHours',
      key: 'cpuCoreHours',
      render: (val: number) => val?.toFixed(2) || '0.00',
    },
    {
      title: '内存时长 (GB时)',
      dataIndex: 'ramGBHours',
      key: 'ramGBHours',
      render: (val: number) => val?.toFixed(2) || '0.00',
    },
    {
      title: '费用',
      dataIndex: 'totalCost',
      key: 'totalCost',
      render: (val: number) => `$${val?.toFixed(2) || '0.00'}`,
    },
  ];

  // Prepare node status donut data
  const nodeStatusData = overview?.nodesByStatus
    ? Object.entries(overview.nodesByStatus).map(([status, count]) => ({
        name: nodeStatusLabels[status] || status,
        value: count,
        color: nodeStatusColors[status] || '#c7c7cc',
      }))
    : [];

  // Prepare cost trend data for the area chart
  const costTrendPoints = (costTrendData || []).map(d => ({
    date: d.date,
    value: d.totalCost,
  }));

  const handleRefresh = () => {
    queryClient.invalidateQueries();
  };

  if (overviewLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="dashboard">
      <PageHeader
        icon={<DashboardOutlined />}
        title="资源总览"
        subtitle="实时掌握集群节点、团队配额与成本消耗情况"
        extra={
          <Space size={12}>
            <Text type="secondary" style={{ fontSize: 13 }}>
              更新于 {dayjs(dataUpdatedAt).format('HH:mm:ss')}
            </Text>
            <Tooltip title="刷新数据">
              <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                刷新
              </Button>
            </Tooltip>
          </Space>
        }
      />

      {/* Summary Cards */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="集群节点"
            value={overview?.totalNodes || 0}
            suffix="个"
            icon={<ClusterOutlined />}
            gradient="var(--gradient-blue)"
            onClick={() => navigate('/cluster/nodes')}
          />
        </Col>
        {features?.capsuleEnabled !== false && (
          <>
            <Col xs={24} sm={12} lg={6}>
              <StatCard
                title="团队数量"
                value={overview?.totalTeams || 0}
                suffix="个"
                icon={<TeamOutlined />}
                gradient="var(--gradient-purple)"
                onClick={() => navigate('/teams')}
              />
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <StatCard
                title="项目数量"
                value={overview?.totalProjects || 0}
                suffix="个"
                icon={<ProjectOutlined />}
                gradient="var(--gradient-teal)"
                onClick={() => navigate('/projects')}
              />
            </Col>
          </>
        )}
        {features?.costEnabled !== false && (
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="费用统计"
              value={overview?.costEnabled ? '已启用' : '未启用'}
              icon={<DollarOutlined />}
              gradient="var(--gradient-green)"
              valueColor={overview?.costEnabled ? 'var(--success)' : 'var(--text-tertiary)'}
              hint={overview?.costEnabled ? 'OpenCost 成本追踪运行中' : '前往设置启用成本追踪'}
            />
          </Col>
        )}
      </Row>

      {/* Node Status and Quota Alerts */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card
            title={<><ClusterOutlined /> 节点状态分布</>}
            extra={<a onClick={() => navigate('/cluster/nodes')}>查看全部</a>}
          >
            <DonutChart data={nodeStatusData} centerLabel="节点" />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={<><WarningOutlined style={{ color: 'var(--warning)' }} /> 配额预警 (≥80%)</>}
            styles={{ body: { maxHeight: 240, overflow: 'auto' } }}
          >
            {quotaAlertsData && quotaAlertsData.length > 0 ? (
              <List
                size="small"
                dataSource={quotaAlertsData.slice(0, 5)}
                renderItem={(alert: QuotaAlert) => (
                  <List.Item>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
                      <Tag color={alert.type === 'team' ? 'blue' : 'green'}>
                        {alert.type === 'team' ? '团队' : '项目'}
                      </Tag>
                      <a onClick={() => navigate(`/${alert.type === 'team' ? 'teams' : 'projects'}/${alert.name}`)}>
                        {alert.displayName || alert.name}
                      </a>
                      <span style={{ color: 'var(--text-tertiary)', marginLeft: 'auto' }}>
                        {alert.resource}: {alert.used}/{alert.limit}
                      </span>
                      <Progress
                        percent={Math.round(alert.usagePercent)}
                        size="small"
                        style={{ width: 80 }}
                        status={alert.usagePercent >= 90 ? 'exception' : 'active'}
                      />
                    </div>
                  </List.Item>
                )}
              />
            ) : (
              <Empty description="暂无配额预警" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>
      </Row>

      {/* Cost Trend and Top Consumers (if cost enabled) */}
      {overview?.costEnabled && (
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={14}>
            <Card title={<><RiseOutlined /> 费用趋势 (近 7 天)</>}>
              <AreaChart data={costTrendPoints} height={200} valuePrefix="$" />
            </Card>
          </Col>
          <Col xs={24} lg={10}>
            <Card title={<><TrophyOutlined style={{ color: 'var(--warning)' }} /> 资源消耗 Top 5</>}>
              {topConsumersData && topConsumersData.length > 0 ? (
                <List
                  size="small"
                  dataSource={topConsumersData}
                  renderItem={(item: TopConsumer, index: number) => (
                    <List.Item>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
                        <span
                          style={{
                            width: 22,
                            height: 22,
                            borderRadius: '50%',
                            color: index < 3 ? '#fff' : 'var(--text-secondary)',
                            background: index < 3
                              ? ['linear-gradient(135deg,#f7b500,#ffd84d)', 'linear-gradient(135deg,#9ca3af,#cbd5e1)', 'linear-gradient(135deg,#cd7f32,#e0a96d)'][index]
                              : 'var(--border)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: 12,
                            fontWeight: 700,
                            flexShrink: 0,
                          }}
                        >
                          {index + 1}
                        </span>
                        <Tag color={item.type === 'team' ? 'blue' : 'green'}>
                          {item.type === 'team' ? '团队' : '项目'}
                        </Tag>
                        <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {item.displayName || item.name}
                        </span>
                        <span style={{ fontWeight: 600, color: 'var(--accent)' }}>
                          ${item.totalCost.toFixed(2)}
                        </span>
                      </div>
                    </List.Item>
                  )}
                />
              ) : (
                <Empty description="暂无消耗数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>
          </Col>
        </Row>
      )}

      {/* Resource Usage and Arch Distribution */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card title={<><DatabaseOutlined /> 集群资源</>}>
            {overview?.resources && overview.resources.length > 0 ? (
              <Table
                dataSource={overview.resources}
                columns={resourceColumns}
                rowKey="name"
                pagination={false}
                size="small"
              />
            ) : (
              <Empty
                description={
                  <span>
                    暂无资源数据，请先在{' '}
                    <a onClick={() => navigate('/settings/resources')}>系统设置 → 资源配置</a>
                    {' '}中配置要显示的资源
                  </span>
                }
              />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title={<><DesktopOutlined /> 节点架构分布</>}>
            {overview?.nodesByArch && overview.nodesByArch.length > 0 ? (
              <Space direction="vertical" style={{ width: '100%' }} size={12}>
                {overview.nodesByArch.map(item => (
                  <div key={item.arch} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Space>
                      <DesktopOutlined style={{ color: 'var(--accent)' }} />
                      <Text>{item.arch}</Text>
                    </Space>
                    <Tag color="blue">{item.count} 节点</Tag>
                  </div>
                ))}
              </Space>
            ) : (
              <Empty description="暂无架构数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </Col>
      </Row>

      {/* Team Usage (if cost and capsule enabled) */}
      {overview?.costEnabled && features?.capsuleEnabled !== false && (
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={12}>
            <Card
              title="团队资源使用 (近 7 天)"
              extra={
                <Text type="secondary">
                  总费用: ${teamUsage?.totalCost?.toFixed(2) || '0.00'}
                </Text>
              }
            >
              {usageLoading ? (
                <Spin />
              ) : teamUsage?.data && teamUsage.data.length > 0 ? (
                <Table
                  dataSource={teamUsage.data.slice(0, 5)}
                  columns={usageColumns}
                  rowKey="name"
                  pagination={false}
                  size="small"
                />
              ) : (
                <Empty description="暂无使用数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card
              title="项目资源使用 (近 7 天)"
              extra={
                <Text type="secondary">
                  总费用: ${projectUsage?.totalCost?.toFixed(2) || '0.00'}
                </Text>
              }
            >
              {projectUsageLoading ? (
                <Spin />
              ) : projectUsage?.data && projectUsage.data.length > 0 ? (
                <Table
                  dataSource={projectUsage.data.slice(0, 5)}
                  columns={projectUsageColumns}
                  rowKey="name"
                  pagination={false}
                  size="small"
                />
              ) : (
                <Empty description="暂无使用数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>
          </Col>
        </Row>
      )}
    </div>
  );
};

export default Dashboard;

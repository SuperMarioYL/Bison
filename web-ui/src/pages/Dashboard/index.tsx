import React from 'react';
import { Card, Row, Col, Statistic, Table, Progress, Typography, Space, Spin, Tag, Empty, List, Tooltip } from 'antd';
import { 
  ClusterOutlined, 
  TeamOutlined, 
  ProjectOutlined,
  DesktopOutlined,
  DollarOutlined,
  WarningOutlined,
  TrophyOutlined,
  RiseOutlined
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { 
  getOverview, 
  getTeamUsage, 
  getProjectUsage,
  getEnabledResourceConfigs, 
  getQuotaAlerts,
  getCostTrend,
  getTopConsumers,
  ResourceDefinition,
  QuotaAlert,
  CostTrendPoint,
  TopConsumer
} from '../../services/api';

const { Title, Text } = Typography;

// Node status labels and colors
const nodeStatusLabels: Record<string, string> = {
  shared: '共享池',
  exclusive: '独占',
  disabled: '已禁用',
  unmanaged: '未管理',
};

const nodeStatusColors: Record<string, string> = {
  shared: '#1890ff',
  exclusive: '#52c41a',
  disabled: '#ff4d4f',
  unmanaged: '#d9d9d9',
};

// Simple pie chart component using CSS
const SimplePieChart: React.FC<{ data: { name: string; value: number; color: string }[] }> = ({ data }) => {
  const total = data.reduce((sum, d) => sum + d.value, 0);
  if (total === 0) return <Empty description="暂无数据" />;

  let cumulativePercent = 0;
  const segments = data.map(d => {
    const percent = (d.value / total) * 100;
    const startPercent = cumulativePercent;
    cumulativePercent += percent;
    return { ...d, percent, startPercent };
  });

  // Create conic gradient
  const gradientStops = segments.map(s => {
    return `${s.color} ${s.startPercent}% ${s.startPercent + s.percent}%`;
  }).join(', ');

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
      <div
        style={{
          width: 120,
          height: 120,
          borderRadius: '50%',
          background: `conic-gradient(${gradientStops})`,
          boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
        }}
      />
      <div style={{ flex: 1 }}>
        {segments.map(s => (
          <div key={s.name} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
            <div style={{ width: 12, height: 12, borderRadius: 2, backgroundColor: s.color }} />
            <span style={{ flex: 1 }}>{s.name}</span>
            <span style={{ fontWeight: 500 }}>{s.value}</span>
            <span style={{ color: '#999', fontSize: 12 }}>({s.percent.toFixed(0)}%)</span>
          </div>
        ))}
      </div>
    </div>
  );
};

// Simple line chart component using SVG
const SimpleLineChart: React.FC<{ data: CostTrendPoint[]; height?: number }> = ({ data, height = 150 }) => {
  if (!data || data.length === 0) return <Empty description="暂无数据" />;

  const maxCost = Math.max(...data.map(d => d.totalCost), 1);
  const width = 100;
  const padding = 10;

  const points = data.map((d, i) => {
    const x = padding + (i / (data.length - 1)) * (width - 2 * padding);
    const y = height - padding - (d.totalCost / maxCost) * (height - 2 * padding);
    return { x: x * 4, y, ...d };
  });

  const pathD = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
  const areaD = `${pathD} L ${points[points.length - 1].x} ${height - padding} L ${points[0].x} ${height - padding} Z`;

  return (
    <div>
      <svg width="100%" height={height} viewBox={`0 0 ${width * 4} ${height}`} preserveAspectRatio="xMidYMid meet">
        {/* Area fill */}
        <path d={areaD} fill="rgba(24, 144, 255, 0.1)" />
        {/* Line */}
        <path d={pathD} fill="none" stroke="#1890ff" strokeWidth="2" />
        {/* Points */}
        {points.map((p, i) => (
          <Tooltip key={i} title={`${p.date}: $${p.totalCost.toFixed(2)}`}>
            <circle cx={p.x} cy={p.y} r="4" fill="#1890ff" style={{ cursor: 'pointer' }} />
          </Tooltip>
        ))}
      </svg>
      <div style={{ display: 'flex', justifyContent: 'space-between', paddingTop: 4, fontSize: 11, color: '#999' }}>
        {data.length > 0 && <span>{data[0].date}</span>}
        {data.length > 1 && <span>{data[data.length - 1].date}</span>}
      </div>
    </div>
  );
};

const Dashboard: React.FC = () => {
  const navigate = useNavigate();

  const { data: overview, isLoading: overviewLoading } = useQuery({
    queryKey: ['overview'],
    queryFn: () => getOverview().then(res => res.data),
    refetchInterval: 30000,
  });

  const { data: teamUsage, isLoading: usageLoading } = useQuery({
    queryKey: ['teamUsage', '7d'],
    queryFn: () => getTeamUsage('7d').then(res => res.data),
    refetchInterval: 60000,
  });

  const { data: projectUsage, isLoading: projectUsageLoading } = useQuery({
    queryKey: ['projectUsage', '7d'],
    queryFn: () => getProjectUsage('7d').then(res => res.data),
    refetchInterval: 60000,
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
      render: (name: string) => getDisplayName(name)
    },
    { 
      title: '容量', 
      dataIndex: 'capacity', 
      key: 'capacity',
      render: (val: number, record: { name: string }) => formatResourceValue(record.name, val)
    },
    { 
      title: '可分配', 
      dataIndex: 'allocatable', 
      key: 'allocatable',
      render: (val: number, record: { name: string }) => formatResourceValue(record.name, val)
    },
    {
      title: '使用率',
      key: 'usage',
      render: (_: unknown, record: { capacity: number; allocatable: number }) => {
        const used = record.capacity - record.allocatable;
        const percent = record.capacity > 0 ? (used / record.capacity) * 100 : 0;
        return <Progress percent={Math.round(percent)} size="small" />;
      }
    }
  ];

  // Team usage columns
  const usageColumns = [
    { 
      title: '团队', 
      dataIndex: 'name', 
      key: 'name',
      render: (name: string) => (
        <a onClick={() => navigate(`/teams/${name}`)}>{name}</a>
      )
    },
    { 
      title: 'CPU 时长 (核时)', 
      dataIndex: 'cpuCoreHours', 
      key: 'cpuCoreHours',
      render: (val: number) => val?.toFixed(2) || '0.00'
    },
    { 
      title: '内存时长 (GB时)', 
      dataIndex: 'ramGBHours', 
      key: 'ramGBHours',
      render: (val: number) => val?.toFixed(2) || '0.00'
    },
    { 
      title: 'GPU 时长', 
      dataIndex: 'gpuHours', 
      key: 'gpuHours',
      render: (val: number) => val?.toFixed(2) || '0.00'
    },
    { 
      title: '费用', 
      dataIndex: 'totalCost', 
      key: 'totalCost',
      render: (val: number) => `$${val?.toFixed(2) || '0.00'}`
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
      )
    },
    { 
      title: 'CPU 时长 (核时)', 
      dataIndex: 'cpuCoreHours', 
      key: 'cpuCoreHours',
      render: (val: number) => val?.toFixed(2) || '0.00'
    },
    { 
      title: '内存时长 (GB时)', 
      dataIndex: 'ramGBHours', 
      key: 'ramGBHours',
      render: (val: number) => val?.toFixed(2) || '0.00'
    },
    { 
      title: '费用', 
      dataIndex: 'totalCost', 
      key: 'totalCost',
      render: (val: number) => `$${val?.toFixed(2) || '0.00'}`
    },
  ];

  // Prepare node status pie chart data
  const nodeStatusData = overview?.nodesByStatus ? Object.entries(overview.nodesByStatus).map(([status, count]) => ({
    name: nodeStatusLabels[status] || status,
    value: count,
    color: nodeStatusColors[status] || '#d9d9d9',
  })) : [];

  if (overviewLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="dashboard">
      <Title level={2}>资源总览</Title>
      
      {/* Summary Cards */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            hoverable 
            onClick={() => navigate('/cluster/nodes')}
            className="glass-card"
          >
            <Statistic
              title="集群节点"
              value={overview?.totalNodes || 0}
              prefix={<ClusterOutlined />}
              suffix="个"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            hoverable 
            onClick={() => navigate('/teams')}
            className="glass-card"
          >
            <Statistic
              title="团队数量"
              value={overview?.totalTeams || 0}
              prefix={<TeamOutlined />}
              suffix="个"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            hoverable 
            onClick={() => navigate('/projects')}
            className="glass-card"
          >
            <Statistic
              title="项目数量"
              value={overview?.totalProjects || 0}
              prefix={<ProjectOutlined />}
              suffix="个"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="glass-card">
            <Statistic
              title="费用统计"
              value={overview?.costEnabled ? '已启用' : '未启用'}
              prefix={<DollarOutlined />}
              valueStyle={{ color: overview?.costEnabled ? '#52c41a' : '#999' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Node Status and Quota Alerts */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card 
            title={<><ClusterOutlined /> 节点状态分布</>} 
            className="glass-card"
            extra={<a onClick={() => navigate('/cluster/nodes')}>查看全部</a>}
          >
            <SimplePieChart data={nodeStatusData} />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card 
            title={<><WarningOutlined style={{ color: '#faad14' }} /> 配额预警 (≥80%)</>} 
            className="glass-card"
            bodyStyle={{ maxHeight: 200, overflow: 'auto' }}
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
                      <span style={{ color: '#999', marginLeft: 'auto' }}>
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
            <Card 
              title={<><RiseOutlined /> 费用趋势 (7天)</>} 
              className="glass-card"
            >
              <SimpleLineChart data={costTrendData || []} height={180} />
            </Card>
          </Col>
          <Col xs={24} lg={10}>
            <Card 
              title={<><TrophyOutlined style={{ color: '#faad14' }} /> 资源消耗 Top 5</>} 
              className="glass-card"
            >
              {topConsumersData && topConsumersData.length > 0 ? (
                <List
                  size="small"
                  dataSource={topConsumersData}
                  renderItem={(item: TopConsumer, index: number) => (
                    <List.Item>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
                        <span style={{ 
                          width: 20, 
                          height: 20, 
                          borderRadius: '50%', 
                          backgroundColor: index < 3 ? ['#FFD700', '#C0C0C0', '#CD7F32'][index] : '#d9d9d9',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 12,
                          fontWeight: 'bold',
                        }}>
                          {index + 1}
                        </span>
                        <Tag color={item.type === 'team' ? 'blue' : 'green'}>
                          {item.type === 'team' ? '团队' : '项目'}
                        </Tag>
                        <span style={{ flex: 1 }}>{item.displayName || item.name}</span>
                        <span style={{ fontWeight: 500, color: '#1890ff' }}>
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
          <Card title="集群资源" className="glass-card">
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
          <Card title="节点架构分布" className="glass-card">
            {overview?.nodesByArch && overview.nodesByArch.length > 0 ? (
              <Space direction="vertical" style={{ width: '100%' }}>
                {overview.nodesByArch.map(item => (
                  <div key={item.arch} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Space>
                      <DesktopOutlined />
                      <Text>{item.arch}</Text>
                    </Space>
                    <Tag color="blue">{item.count} 节点</Tag>
                  </div>
                ))}
              </Space>
            ) : (
              <Empty description="暂无架构数据" />
            )}
          </Card>
        </Col>
      </Row>

      {/* Team Usage (if cost enabled) */}
      {overview?.costEnabled && (
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={12}>
            <Card 
              title="团队资源使用 (过去 7 天)" 
              className="glass-card"
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
                <Empty description="暂无使用数据" />
              )}
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card 
              title="项目资源使用 (过去 7 天)" 
              className="glass-card"
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
                <Empty description="暂无使用数据" />
              )}
            </Card>
          </Col>
        </Row>
      )}
    </div>
  );
};

export default Dashboard;

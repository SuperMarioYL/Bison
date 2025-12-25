import React from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Button } from 'antd';
import { 
  DashboardOutlined, 
  CheckCircleOutlined, 
  CloseCircleOutlined,
  ReloadOutlined,
  TeamOutlined,
  ProjectOutlined,
  UserOutlined,
  ClusterOutlined,
  DollarOutlined
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { getSystemStatus, ServiceStatus } from '../../services/api';
import dayjs from 'dayjs';

const SystemStatusPage: React.FC = () => {
  const { data: statusData, isLoading, refetch } = useQuery({
    queryKey: ['systemStatus'],
    queryFn: getSystemStatus,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  const status = statusData?.data;

  const renderServiceStatus = (service: ServiceStatus) => (
    <Card size="small">
      <Space>
        {service.available ? (
          <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 24 }} />
        ) : (
          <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 24 }} />
        )}
        <div>
          <div style={{ fontWeight: 500 }}>{service.name}</div>
          <div style={{ color: '#666', fontSize: 12 }}>
            {service.message || (service.available ? '运行正常' : '连接失败')}
          </div>
        </div>
      </Space>
    </Card>
  );

  const taskColumns = [
    {
      title: '任务',
      dataIndex: 'taskName',
      key: 'taskName',
      render: (name: string) => {
        const labels: Record<string, string> = {
          billing: '计费任务',
          auto_recharge: '自动充值',
          alert_check: '告警检查',
        };
        return labels[name] || name;
      },
    },
    {
      title: '开始时间',
      dataIndex: 'startTime',
      key: 'startTime',
      render: (ts: string) => dayjs(ts).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '结束时间',
      dataIndex: 'endTime',
      key: 'endTime',
      render: (ts: string) => dayjs(ts).format('HH:mm:ss'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const colors: Record<string, string> = {
          success: 'success',
          failed: 'error',
          skipped: 'warning',
        };
        return <Tag color={colors[status]}>{status}</Tag>;
      },
    },
    {
      title: '错误',
      dataIndex: 'error',
      key: 'error',
      ellipsis: true,
      render: (error: string) => error || '-',
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 style={{ margin: 0 }}>
          <DashboardOutlined style={{ marginRight: 8 }} />
          系统状态
        </h3>
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
          刷新
        </Button>
      </div>

      <Card title="服务状态" loading={isLoading} style={{ marginBottom: 16 }}>
        <Row gutter={16}>
          <Col span={8}>
            {status?.opencost && renderServiceStatus(status.opencost)}
          </Col>
          <Col span={8}>
            {status?.capsule && renderServiceStatus(status.capsule)}
          </Col>
          <Col span={8}>
            {status?.prometheus && renderServiceStatus(status.prometheus)}
          </Col>
        </Row>
      </Card>

      <Card title="系统统计" loading={isLoading} style={{ marginBottom: 16 }}>
        <Row gutter={16}>
          <Col span={4}>
            <Statistic
              title="团队数"
              value={status?.statistics?.totalTeams || 0}
              prefix={<TeamOutlined />}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="项目数"
              value={status?.statistics?.totalProjects || 0}
              prefix={<ProjectOutlined />}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="用户数"
              value={status?.statistics?.totalUsers || 0}
              prefix={<UserOutlined />}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="节点数"
              value={status?.statistics?.totalNodes || 0}
              prefix={<ClusterOutlined />}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="总余额"
              value={status?.statistics?.totalBalance || 0}
              precision={2}
              prefix={<DollarOutlined />}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="欠费团队"
              value={status?.statistics?.suspendedTeams || 0}
              valueStyle={{ color: (status?.statistics?.suspendedTeams || 0) > 0 ? '#cf1322' : '#3f8600' }}
            />
          </Col>
        </Row>
      </Card>

      <Card title="定时任务执行历史" loading={isLoading}>
        <Table
          columns={taskColumns}
          dataSource={status?.tasks || []}
          rowKey={(record) => `${record.taskName}-${record.startTime}`}
          pagination={{ pageSize: 10 }}
        />
      </Card>
    </div>
  );
};

export default SystemStatusPage;


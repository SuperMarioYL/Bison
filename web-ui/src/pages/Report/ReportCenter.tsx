import React, { useState } from 'react';
import { Card, Row, Col, Statistic, Table, Select, Button, Space, Spin, message } from 'antd';
import { 
  BarChartOutlined, 
  TeamOutlined, 
  DownloadOutlined,
  DollarOutlined,
  TrophyOutlined 
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { 
  getSummaryReport, 
  exportSummaryReport,
} from '../../services/api';
import PageHeader from '../../components/PageHeader';

const ReportCenter: React.FC = () => {
  const [window, setWindow] = useState('30d');

  const { data: reportData, isLoading } = useQuery({
    queryKey: ['summaryReport', window],
    queryFn: () => getSummaryReport(window),
  });

  const report = reportData?.data;

  const handleExport = async () => {
    try {
      const response = await exportSummaryReport(window);
      const blob = new Blob([response.data], { type: 'text/csv' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `summary-report-${window}.csv`;
      a.click();
      URL.revokeObjectURL(url);
      message.success('导出成功');
    } catch (error) {
      message.error('导出失败');
    }
  };

  const rankColumns = [
    {
      title: '排名',
      dataIndex: 'rank',
      key: 'rank',
      width: 80,
      render: (rank: number) => {
        const colors = ['#FFD700', '#C0C0C0', '#CD7F32'];
        if (rank <= 3) {
          return (
            <TrophyOutlined style={{ color: colors[rank - 1], fontSize: 18 }} />
          );
        }
        return rank;
      },
    },
    {
      title: '团队',
      dataIndex: 'teamName',
      key: 'teamName',
      render: (name: string) => (
        <Link to={`/teams/${name}`}>
          <Space>
            <TeamOutlined />
            {name}
          </Space>
        </Link>
      ),
    },
    {
      title: '消费金额',
      dataIndex: 'cost',
      key: 'cost',
      render: (cost: number) => `¥${cost.toFixed(2)}`,
    },
    {
      title: '占比',
      dataIndex: 'percentage',
      key: 'percentage',
      render: (percentage: number) => `${percentage.toFixed(1)}%`,
    },
  ];

  if (isLoading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />;
  }

  return (
    <div>
      <PageHeader
        icon={<BarChartOutlined />}
        gradient="var(--gradient-green)"
        title="报表中心"
        subtitle="按时间区间汇总团队与项目的成本消耗"
        extra={
          <Space>
            <Select
              value={window}
              onChange={setWindow}
              style={{ width: 120 }}
            >
              <Select.Option value="7d">近 7 天</Select.Option>
              <Select.Option value="30d">近 30 天</Select.Option>
              <Select.Option value="90d">近 90 天</Select.Option>
            </Select>
            <Button icon={<DownloadOutlined />} onClick={handleExport}>
              导出报表
            </Button>
          </Space>
        }
      />

      <Row gutter={[16, 16]}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总消费"
              value={report?.totalCost || 0}
              precision={2}
              prefix={<DollarOutlined />}
              suffix="元"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="团队数"
              value={report?.totalTeams || 0}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="项目数"
              value={report?.totalProjects || 0}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="统计周期"
              value={window}
              valueStyle={{ fontSize: 24 }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card title="团队消费排行榜 Top 10">
            <Table
              columns={rankColumns}
              dataSource={report?.topTeams || []}
              rowKey="teamName"
              pagination={false}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default ReportCenter;


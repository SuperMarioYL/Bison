import React, { useState } from 'react';
import { Table, Card, Tag, Space, DatePicker, Select, Input, Button, Tooltip } from 'antd';
import { AuditOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { getAuditLogs, AuditFilter } from '../../services/api';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

const AuditList: React.FC = () => {
  const [filter, setFilter] = useState<AuditFilter>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['auditLogs', filter, page, pageSize],
    queryFn: () => getAuditLogs(filter, page, pageSize),
  });

  const logs = data?.data?.items || [];
  const total = data?.data?.total || 0;

  const actionColors: Record<string, string> = {
    create: 'green',
    update: 'blue',
    delete: 'red',
    recharge: 'gold',
    suspend: 'orange',
    resume: 'cyan',
  };

  const resourceLabels: Record<string, string> = {
    team: '团队',
    project: '项目',
    user: '用户',
    config: '配置',
  };

  const columns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (timestamp: string) => dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作者',
      dataIndex: 'operator',
      key: 'operator',
      width: 120,
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 100,
      render: (action: string) => (
        <Tag color={actionColors[action] || 'default'}>{action}</Tag>
      ),
    },
    {
      title: '资源类型',
      dataIndex: 'resource',
      key: 'resource',
      width: 100,
      render: (resource: string) => resourceLabels[resource] || resource,
    },
    {
      title: '目标',
      dataIndex: 'target',
      key: 'target',
      width: 150,
    },
    {
      title: '详情',
      dataIndex: 'detail',
      key: 'detail',
      ellipsis: true,
      render: (detail: Record<string, unknown>) => (
        detail ? (
          <Tooltip title={JSON.stringify(detail, null, 2)}>
            <span>{JSON.stringify(detail).substring(0, 50)}...</span>
          </Tooltip>
        ) : '-'
      ),
    },
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      width: 120,
      render: (ip: string) => ip || '-',
    },
  ];

  const handleDateChange = (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      setFilter({
        ...filter,
        from: dates[0].toISOString(),
        to: dates[1].toISOString(),
      });
    } else {
      const { from, to, ...rest } = filter;
      setFilter(rest);
    }
    setPage(1);
  };

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>
          <AuditOutlined style={{ marginRight: 8 }} />
          审计日志
        </h2>
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
          刷新
        </Button>
      </div>

      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="操作类型"
            style={{ width: 120 }}
            allowClear
            onChange={(value) => {
              setFilter({ ...filter, action: value });
              setPage(1);
            }}
          >
            <Select.Option value="create">创建</Select.Option>
            <Select.Option value="update">更新</Select.Option>
            <Select.Option value="delete">删除</Select.Option>
            <Select.Option value="recharge">充值</Select.Option>
            <Select.Option value="suspend">暂停</Select.Option>
            <Select.Option value="resume">恢复</Select.Option>
          </Select>

          <Select
            placeholder="资源类型"
            style={{ width: 120 }}
            allowClear
            onChange={(value) => {
              setFilter({ ...filter, resource: value });
              setPage(1);
            }}
          >
            <Select.Option value="team">团队</Select.Option>
            <Select.Option value="project">项目</Select.Option>
            <Select.Option value="user">用户</Select.Option>
            <Select.Option value="config">配置</Select.Option>
          </Select>

          <Input
            placeholder="操作者"
            prefix={<SearchOutlined />}
            style={{ width: 150 }}
            onChange={(e) => {
              setFilter({ ...filter, operator: e.target.value });
              setPage(1);
            }}
          />

          <Input
            placeholder="目标"
            prefix={<SearchOutlined />}
            style={{ width: 150 }}
            onChange={(e) => {
              setFilter({ ...filter, target: e.target.value });
              setPage(1);
            }}
          />

          <RangePicker
            showTime
            onChange={handleDateChange}
          />
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={logs}
          loading={isLoading}
          rowKey="id"
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>
    </div>
  );
};

export default AuditList;


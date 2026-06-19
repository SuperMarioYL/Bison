import React, { useMemo } from 'react';
import { Table, Card, Button, Tag, Space, Typography, message, Popconfirm, Empty, Spin, Tooltip } from 'antd';
import { PlusOutlined, TeamOutlined, EditOutlined, DeleteOutlined, CloudServerOutlined, ShareAltOutlined, WalletOutlined, WarningOutlined, ApartmentOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient, useMutation, useQueries } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { getTeams, deleteTeam, getTeamBalance, Team, OwnerRef, TeamMode, Balance, getBillingConfig } from '../../services/api';
import { ResourceQuotaUsageDisplay } from '../../components/ResourceQuotaInput';
import PageHeader from '../../components/PageHeader';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Text } = Typography;

const TeamList: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: teamsData, isLoading } = useQuery({
    queryKey: ['teams'],
    queryFn: () => getTeams().then(res => res.data),
  });

  const { data: billingConfig } = useQuery({
    queryKey: ['billingConfig'],
    queryFn: () => getBillingConfig().then(res => res.data),
  });

  // Fetch balances for all teams in parallel
  const balanceQueries = useQueries({
    queries: (teamsData?.items || []).map(team => ({
      queryKey: ['teamBalance', team.name],
      queryFn: () => getTeamBalance(team.name).then(res => res.data),
      staleTime: 30000, // 30 seconds
    })),
  });

  // Create a map of team name to balance
  const balanceMap = useMemo(() => {
    const map: Record<string, Balance> = {};
    if (teamsData?.items) {
      teamsData.items.forEach((team, index) => {
        const balanceQuery = balanceQueries[index];
        if (balanceQuery?.data) {
          map[team.name] = balanceQuery.data;
        }
      });
    }
    return map;
  }, [teamsData?.items, balanceQueries]);

  const deleteMutation = useMutation({
    mutationFn: deleteTeam,
    onSuccess: (_, teamName) => {
      message.success('团队删除成功');
      // Optimistic update
      queryClient.setQueryData(['teams'], (old: { items: Team[] } | undefined) => {
        if (!old) return old;
        return {
          ...old,
          items: old.items.filter(t => t.name !== teamName)
        };
      });
      queryClient.refetchQueries({ queryKey: ['teams'] });
    },
    onError: (error: Error) => {
      message.error(`删除失败: ${error.message}`);
    },
  });

  // Helper function to render balance status
  const renderBalanceStatus = (teamName: string) => {
    const balance = balanceMap[teamName];
    if (!balance) return <Spin size="small" />;

    const amount = balance.amount;
    const isOverdue = amount < 0;
    const currencySymbol = billingConfig?.currencySymbol || '¥';

    return (
      <Tooltip title={`日均消耗: ${currencySymbol}${(balance.dailyConsumption || 0).toFixed(2)}`}>
        <Space>
          <WalletOutlined />
          <Text
            style={{
              color: isOverdue ? '#ff4d4f' : amount < 100 ? '#faad14' : '#52c41a',
              fontWeight: 500,
            }}
          >
            {currencySymbol}{amount.toFixed(2)}
          </Text>
        </Space>
      </Tooltip>
    );
  };

  // Helper function to render estimated overdue time
  const renderEstimatedOverdue = (teamName: string) => {
    const balance = balanceMap[teamName];
    if (!balance) return '-';

    // Already overdue
    if (balance.amount < 0) {
      const overdueAt = balance.overdueAt ? dayjs(balance.overdueAt) : null;
      const graceRemaining = balance.graceRemaining;

      if (overdueAt) {
        return (
          <Tooltip title={`欠费开始: ${overdueAt.format('YYYY-MM-DD HH:mm')}`}>
            <Tag color="error" icon={<WarningOutlined />}>
              已欠费 {overdueAt.fromNow(true)}
              {graceRemaining && graceRemaining !== '已到期' && ` (宽限: ${graceRemaining})`}
            </Tag>
          </Tooltip>
        );
      }
      return <Tag color="error">已欠费</Tag>;
    }

    // Estimate future overdue
    if (balance.estimatedOverdueAt) {
      const estimatedAt = dayjs(balance.estimatedOverdueAt);
      const daysRemaining = estimatedAt.diff(dayjs(), 'day');

      if (daysRemaining <= 7) {
        return (
          <Tooltip title={`预计欠费时间: ${estimatedAt.format('YYYY-MM-DD HH:mm')}`}>
            <Tag color="warning">{estimatedAt.fromNow()}</Tag>
          </Tooltip>
        );
      }
      return (
        <Tooltip title={`预计欠费时间: ${estimatedAt.format('YYYY-MM-DD HH:mm')}`}>
          <Tag color="default">{estimatedAt.fromNow()}</Tag>
        </Tooltip>
      );
    }

    // No consumption data
    if (!balance.dailyConsumption || balance.dailyConsumption === 0) {
      return <Tag color="default">无消耗</Tag>;
    }

    return <Tag color="success">充足</Tag>;
  };

  const columns = [
    {
      title: '团队名称',
      dataIndex: 'name',
      key: 'name',
      width: 120,
      fixed: 'left' as const,
      render: (name: string, record: Team) => (
        <Space>
          <TeamOutlined />
          <a onClick={() => navigate(`/teams/${name}`)}>
            {record.displayName || name}
          </a>
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 150,
      ellipsis: true,
    },
    {
      title: '所有者',
      dataIndex: 'owners',
      key: 'owners',
      width: 180,
      render: (owners: OwnerRef[]) => (
        <Space wrap>
          {owners?.map((owner, index) => (
            <Tag key={`${owner.kind}-${owner.name}-${index}`} color={owner.kind === 'User' ? 'green' : 'blue'}>
              {owner.kind === 'User' ? '👤' : '👥'} {owner.name}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '资源模式',
      dataIndex: 'mode',
      key: 'mode',
      width: 140,
      render: (mode: TeamMode, record: Team) => (
        <Space>
          {mode === 'exclusive' ? (
            <Tag color="success" icon={<CloudServerOutlined />}>
              独占 ({record.exclusiveNodes?.length || 0}节点)
            </Tag>
          ) : (
            <Tag color="processing" icon={<ShareAltOutlined />}>
              共享
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: '余额',
      key: 'balance',
      width: 120,
      render: (_: unknown, record: Team) => renderBalanceStatus(record.name),
    },
    {
      title: '预计欠费',
      key: 'estimatedOverdue',
      width: 150,
      render: (_: unknown, record: Team) => renderEstimatedOverdue(record.name),
    },
    {
      title: '资源使用',
      key: 'quota',
      width: 240,
      render: (_: unknown, record: Team) => (
        <Tooltip title={record.mode === 'exclusive' ? '独占节点资源总量' : '配额限制'}>
          <ResourceQuotaUsageDisplay 
            quota={record.quota || {}} 
            quotaUsed={record.quotaUsed} 
            compact 
            maxItems={3}
            label={record.mode === 'exclusive' ? '节点资源' : '配额'}
          />
        </Tooltip>
      ),
    },
    {
      title: '项目数',
      dataIndex: 'projectCount',
      key: 'projectCount',
      width: 80,
    },
    {
      title: '状态',
      key: 'status',
      width: 80,
      render: (_: unknown, record: Team) => {
        if (record.suspended) {
          return <Tag color="error">已暂停</Tag>;
        }
        return record.status?.ready ? (
          <Tag color="success">正常</Tag>
        ) : (
          <Tag color="warning">{record.status?.state || '未知'}</Tag>
        );
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      fixed: 'right' as const,
      render: (_: unknown, record: Team) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => navigate(`/teams/${record.name}`)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此团队？"
            description="删除团队将同时删除所有关联的项目和资源"
            onConfirm={() => deleteMutation.mutate(record.name)}
            okText="确定"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 50 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        icon={<ApartmentOutlined />}
        gradient="var(--gradient-purple)"
        title="团队管理"
        subtitle="管理多租户团队、资源配额与计费余额"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate('/teams/create')}
          >
            创建团队
          </Button>
        }
      />

      <Card className="glass-card">
        {teamsData?.items && teamsData.items.length > 0 ? (
          <Table
            dataSource={teamsData.items}
            columns={columns}
            rowKey="name"
            pagination={{ pageSize: 10 }}
            scroll={{ x: 1400 }}
          />
        ) : (
          <Empty description="暂无团队">
            <Button type="primary" onClick={() => navigate('/teams/create')}>
              创建第一个团队
            </Button>
          </Empty>
        )}
      </Card>
    </div>
  );
};

export default TeamList;

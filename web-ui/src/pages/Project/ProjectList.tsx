import React, { useMemo } from 'react';
import { Table, Card, Button, Tag, Space, Typography, message, Popconfirm, Empty, Spin, Select, Tooltip } from 'antd';
import { PlusOutlined, ProjectOutlined, EditOutlined, DeleteOutlined, DeploymentUnitOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient, useMutation, useQueries } from '@tanstack/react-query';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { getProjects, deleteProject, getTeams, getProjectWorkloadSummary, getProjectUsage, Project, WorkloadSummary, ProjectUsage } from '../../services/api';

const { Title, Text } = Typography;

const ProjectList: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const teamFilter = searchParams.get('team') || undefined;

  const { data: projectsData, isLoading } = useQuery({
    queryKey: ['projects', teamFilter],
    queryFn: () => getProjects(teamFilter).then(res => res.data),
  });

  const { data: teamsData } = useQuery({
    queryKey: ['teams'],
    queryFn: () => getTeams().then(res => res.data),
  });

  // Fetch workload summary for all projects in parallel
  const workloadQueries = useQueries({
    queries: (projectsData?.items || []).map(project => ({
      queryKey: ['workloadSummary', project.name],
      queryFn: () => getProjectWorkloadSummary(project.name).then(res => res.data),
      staleTime: 30000, // 30 seconds
    })),
  });

  // Fetch usage data for all projects in parallel
  const usageQueries = useQueries({
    queries: (projectsData?.items || []).map(project => ({
      queryKey: ['projectUsage', project.name],
      queryFn: () => getProjectUsage(project.name).then(res => res.data),
      staleTime: 30000, // 30 seconds
    })),
  });

  // Create a map of project name to workload summary
  const workloadMap = useMemo(() => {
    const map: Record<string, WorkloadSummary> = {};
    if (projectsData?.items) {
      projectsData.items.forEach((project, index) => {
        const workloadQuery = workloadQueries[index];
        if (workloadQuery?.data) {
          map[project.name] = workloadQuery.data;
        }
      });
    }
    return map;
  }, [projectsData?.items, workloadQueries]);

  // Create a map of project name to usage data
  const usageMap = useMemo(() => {
    const map: Record<string, ProjectUsage> = {};
    if (projectsData?.items) {
      projectsData.items.forEach((project, index) => {
        const usageQuery = usageQueries[index];
        if (usageQuery?.data) {
          map[project.name] = usageQuery.data;
        }
      });
    }
    return map;
  }, [projectsData?.items, usageQueries]);

  // Helper function to render workload summary
  const renderWorkloadSummary = (projectName: string) => {
    const summary = workloadMap[projectName];
    if (!summary) return <Spin size="small" />;

    const parts = [];
    if (summary.deployments > 0) parts.push(`${summary.deployments} Deploy`);
    if (summary.statefulSets > 0) parts.push(`${summary.statefulSets} STS`);
    if (summary.jobs > 0) parts.push(`${summary.jobs} Job`);
    if (summary.cronJobs > 0) parts.push(`${summary.cronJobs} CronJob`);
    if (summary.pods > 0) parts.push(`${summary.pods} Pod`);

    if (parts.length === 0) {
      return <Text type="secondary">无负载</Text>;
    }

    return (
      <Tooltip title={`总共 ${summary.totalPods} 个 Pod`}>
        <Space size={4} wrap>
          <DeploymentUnitOutlined style={{ color: '#1890ff' }} />
          <Text style={{ fontSize: 12 }}>{parts.join(' / ')}</Text>
        </Space>
      </Tooltip>
    );
  };

  // Helper function to render resource usage (dynamic from config)
  const renderResourceUsage = (projectName: string) => {
    const usage = usageMap[projectName];
    if (!usage) return <Spin size="small" />;

    if (!usage.resources || usage.resources.length === 0) {
      return <Text type="secondary">无资源使用</Text>;
    }

    // Display up to 3 resources
    const displayResources = usage.resources.slice(0, 3);
    const hasMore = usage.resources.length > 3;

    return (
      <Tooltip 
        title={
          usage.resources.map(r => 
            `${r.displayName || r.name}: ${r.used.toFixed(2)} ${r.unit || ''}`
          ).join(', ')
        }
      >
        <Space direction="vertical" size={2} style={{ width: '100%' }}>
          {displayResources.map(resource => (
            <div key={resource.name} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text style={{ fontSize: 12, width: 60 }} ellipsis>
                {resource.displayName || resource.name}
              </Text>
              <Text style={{ fontSize: 12, minWidth: 50 }}>
                {resource.used.toFixed(1)} {resource.unit || ''}
              </Text>
            </div>
          ))}
          {hasMore && (
            <Text type="secondary" style={{ fontSize: 11 }}>
              +{usage.resources.length - 3} more
            </Text>
          )}
        </Space>
      </Tooltip>
    );
  };

  const deleteMutation = useMutation({
    mutationFn: deleteProject,
    onSuccess: (_, projectName) => {
      message.success('项目删除成功');
      queryClient.setQueryData(['projects', teamFilter], (old: { items: Project[] } | undefined) => {
        if (!old) return old;
        return {
          ...old,
          items: old.items.filter(p => p.name !== projectName)
        };
      });
      queryClient.refetchQueries({ queryKey: ['projects'] });
    },
    onError: (error: Error) => {
      message.error(`删除失败: ${error.message}`);
    },
  });

  const columns = [
    {
      title: '项目名称',
      dataIndex: 'displayName',
      key: 'displayName',
      width: 120,
      fixed: 'left' as const,
      render: (displayName: string, record: Project) => (
        <Space>
          <ProjectOutlined />
          <a onClick={() => navigate(`/projects/${record.name}`)}>
            {displayName || record.name}
          </a>
        </Space>
      ),
    },
    {
      title: '项目标识',
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: '所属团队',
      dataIndex: 'team',
      key: 'team',
      width: 120,
      render: (team: string) => (
        <a onClick={() => navigate(`/teams/${team}`)}>{team}</a>
      ),
    },
    {
      title: '工作负载',
      key: 'workloads',
      width: 180,
      render: (_: unknown, record: Project) => renderWorkloadSummary(record.name),
    },
    {
      title: '资源使用',
      key: 'usage',
      width: 200,
      render: (_: unknown, record: Project) => renderResourceUsage(record.name),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => (
        <Tag color={status === 'Active' ? 'success' : 'default'}>{status}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      fixed: 'right' as const,
      render: (_: unknown, record: Project) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => navigate(`/projects/${record.name}`)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此项目？"
            description="删除项目将同时删除该命名空间下的所有资源"
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
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={2}>项目管理</Title>
        <Space>
          <Select
            placeholder="按团队筛选"
            allowClear
            style={{ width: 200 }}
            value={teamFilter}
            onChange={(value) => {
              if (value) {
                setSearchParams({ team: value });
              } else {
                setSearchParams({});
              }
            }}
          >
            {teamsData?.items?.map(team => (
              <Select.Option key={team.name} value={team.name}>
                {team.displayName || team.name}
              </Select.Option>
            ))}
          </Select>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate('/projects/create')}
          >
            创建项目
          </Button>
        </Space>
      </div>

      <Card className="glass-card">
        {projectsData?.items && projectsData.items.length > 0 ? (
          <Table
            dataSource={projectsData.items}
            columns={columns}
            rowKey="name"
            pagination={{ pageSize: 10 }}
            scroll={{ x: 1000 }}
          />
        ) : (
          <Empty description="暂无项目">
            <Button type="primary" onClick={() => navigate('/projects/create')}>
              创建第一个项目
            </Button>
          </Empty>
        )}
      </Card>
    </div>
  );
};

export default ProjectList;

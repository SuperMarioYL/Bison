import { ArrowLeftOutlined, DashboardOutlined, DeleteOutlined, DeploymentUnitOutlined, PlusOutlined, ReloadOutlined, SaveOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form, Input,
  message,
  Modal,
  Popconfirm,
  Progress,
  Radio,
  Row,
  Segmented,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { deleteProject, getProject, getProjectUsage, getProjectWorkloads, getProjectWorkloadSummary, getUsers, ProjectMember, ResourceUsage, updateProject, User, Workload } from '../../services/api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const roleConfig: Record<string, { color: string; label: string; description: string }> = {
  admin: { color: 'red', label: '管理员', description: '完全控制，可创建/删除/修改所有资源' },
  edit: { color: 'blue', label: '编辑者', description: '可修改大部分资源，不能改权限' },
  view: { color: 'green', label: '只读', description: '仅查看访问' },
};

const ProjectDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [addMemberModalVisible, setAddMemberModalVisible] = useState(false);
  const [addMemberForm] = Form.useForm();

  const { data: projectData, isLoading } = useQuery({
    queryKey: ['project', name],
    queryFn: () => getProject(name!).then(res => res.data),
    enabled: !!name,
  });

  const { data: usersData } = useQuery({
    queryKey: ['users'],
    queryFn: () => getUsers(),
  });

  // Fetch workload summary
  const { data: workloadSummary, refetch: refetchSummary } = useQuery({
    queryKey: ['workloadSummary', name],
    queryFn: () => getProjectWorkloadSummary(name!).then(res => res.data),
    enabled: !!name,
  });

  // Fetch workloads list
  const { data: workloadsData, isLoading: workloadsLoading, refetch: refetchWorkloads } = useQuery({
    queryKey: ['workloads', name],
    queryFn: () => getProjectWorkloads(name!).then(res => res.data),
    enabled: !!name,
  });

  // Fetch dynamic resource usage
  const { data: resourceUsage, refetch: refetchUsage } = useQuery({
    queryKey: ['projectUsage', name],
    queryFn: () => getProjectUsage(name!).then(res => res.data),
    enabled: !!name,
  });

  const users: User[] = usersData?.data?.items || [];
  const workloads = workloadsData?.items || [];
  const [workloadFilter, setWorkloadFilter] = useState<string>('all');

  const updateMutation = useMutation({
    mutationFn: (values: Parameters<typeof updateProject>[1]) => updateProject(name!, values),
    onSuccess: () => {
      message.success('项目更新成功');
      queryClient.invalidateQueries({ queryKey: ['project', name] });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
    onError: (error: Error) => {
      message.error(`更新失败: ${error.message}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteProject(name!),
    onSuccess: () => {
      message.success('项目删除成功');
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      navigate('/projects');
    },
    onError: (error: Error) => {
      message.error(`删除失败: ${error.message}`);
    },
  });

  const onFinish = (values: {
    displayName?: string;
    description?: string;
  }) => {
    updateMutation.mutate({
      displayName: values.displayName,
      description: values.description,
      members: projectData?.project.members,
    });
  };

  const handleAddMember = (values: { user: string; role: string }) => {
    if (!projectData?.project) return;

    const newMember: ProjectMember = {
      user: values.user,
      role: values.role as 'admin' | 'edit' | 'view',
    };

    // Check for duplicates
    if (projectData.project.members?.some(m => m.user === newMember.user)) {
      message.warning('该成员已存在');
      return;
    }

    const newMembers = [...(projectData.project.members || []), newMember];
    updateMutation.mutate({
      members: newMembers,
    });
    setAddMemberModalVisible(false);
    addMemberForm.resetFields();
  };

  const handleRemoveMember = (user: string) => {
    if (!projectData?.project) return;

    const newMembers = (projectData.project.members || []).filter(m => m.user !== user);
    updateMutation.mutate({
      members: newMembers,
    });
  };

  const handleUpdateMemberRole = (user: string, newRole: string) => {
    if (!projectData?.project) return;

    const newMembers = (projectData.project.members || []).map(m => 
      m.user === user ? { ...m, role: newRole as 'admin' | 'edit' | 'view' } : m
    );
    updateMutation.mutate({
      members: newMembers,
    });
  };

  const memberColumns = [
    {
      title: '用户',
      dataIndex: 'user',
      key: 'user',
      render: (user: string) => (
        <Link to={`/users/${encodeURIComponent(user)}`}>
          <Space>
            <UserOutlined />
            {user}
          </Space>
        </Link>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 150,
      render: (role: string, record: ProjectMember) => (
        <Select
          value={role}
          size="small"
          style={{ width: 120 }}
          onChange={(newRole) => handleUpdateMemberRole(record.user, newRole)}
        >
          {Object.entries(roleConfig).map(([value, config]) => (
            <Select.Option key={value} value={value}>
              <Tag color={config.color}>{config.label}</Tag>
            </Select.Option>
          ))}
        </Select>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: ProjectMember) => (
        <Popconfirm
          title="确定移除此成员？"
          onConfirm={() => handleRemoveMember(record.user)}
        >
          <Button type="link" danger size="small" icon={<DeleteOutlined />}>
            移除
          </Button>
        </Popconfirm>
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

  if (!projectData?.project) {
    return <div>项目不存在</div>;
  }

  const project = projectData.project;
  const usage = projectData.usage;

  // Get users not already in the project
  const availableUsers = users.filter(u => 
    !project.members?.some(m => m.user === u.email)
  );

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/projects')}>
          返回
        </Button>
      </Space>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2}>{project.displayName || project.name}</Title>
        <Popconfirm
          title="确定删除此项目？"
          description="删除项目将同时删除该命名空间下的所有资源"
          onConfirm={() => deleteMutation.mutate()}
          okText="确定"
          cancelText="取消"
          okButtonProps={{ danger: true }}
        >
          <Button danger icon={<DeleteOutlined />} loading={deleteMutation.isPending}>
            删除项目
          </Button>
        </Popconfirm>
      </div>

      {/* Dynamic Resource Usage */}
      <Card 
        title={
          <Space>
            <DashboardOutlined />
            资源使用
          </Space>
        }
        style={{ marginBottom: 16 }} 
        className="glass-card"
        extra={
          <Button icon={<ReloadOutlined />} onClick={() => refetchUsage()}>刷新</Button>
        }
      >
        {resourceUsage?.resources && resourceUsage.resources.length > 0 ? (
          <Row gutter={[16, 16]}>
            {resourceUsage.resources.map((resource: ResourceUsage) => (
              <Col span={8} key={resource.name}>
                <Tooltip title={`原始值: ${resource.rawUsed.toFixed(4)}`}>
                  <div style={{ marginBottom: 8 }}>
                    <Space>
                      <Text strong>{resource.displayName || resource.name}</Text>
                      <Text type="secondary">
                        {resource.used.toFixed(2)} {resource.unit || ''}
                      </Text>
                    </Space>
                  </div>
                  <Progress 
                    percent={Math.min(resource.used * 10, 100)} 
                    showInfo={false}
                    strokeColor="#1890ff"
                    size="small"
                  />
                </Tooltip>
              </Col>
            ))}
          </Row>
        ) : (
          <Empty description="暂无资源使用数据（请先在系统设置中配置资源）" />
        )}
      </Card>

      {/* Cost Usage Statistics */}
      {usage && (
        <Card title="费用统计 (过去 7 天)" style={{ marginBottom: 16 }} className="glass-card">
          <Row gutter={16}>
            <Col span={6}>
              <Statistic title="CPU 时长" value={usage.cpuCoreHours?.toFixed(2) || 0} suffix="核时" />
            </Col>
            <Col span={6}>
              <Statistic title="内存时长" value={usage.ramGBHours?.toFixed(2) || 0} suffix="GB时" />
            </Col>
            <Col span={6}>
              <Statistic title="GPU 时长" value={usage.gpuHours?.toFixed(2) || 0} suffix="小时" />
            </Col>
            <Col span={6}>
              <Statistic title="总费用" value={usage.totalCost?.toFixed(2) || 0} prefix="$" />
            </Col>
          </Row>
        </Card>
      )}

      {/* Project Info */}
      <Card title="项目信息" style={{ marginBottom: 16 }} className="glass-card">
        <Descriptions column={2}>
          <Descriptions.Item label="项目标识">{project.name}</Descriptions.Item>
          <Descriptions.Item label="所属团队">
            <Link to={`/teams/${project.team}`}>{project.team}</Link>
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={project.status === 'Active' ? 'success' : 'default'}>{project.status}</Tag>
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Members */}
      <Card 
        title={`项目成员 (${project.members?.length || 0})`}
        style={{ marginBottom: 16 }} 
        className="glass-card"
        extra={
          <Button 
            type="primary" 
            size="small" 
            icon={<PlusOutlined />}
            onClick={() => setAddMemberModalVisible(true)}
            disabled={availableUsers.length === 0}
          >
            添加成员
          </Button>
        }
      >
        <Table
          dataSource={project.members || []}
          columns={memberColumns}
          rowKey="user"
          pagination={false}
          locale={{ emptyText: '暂无成员' }}
        />
        <div style={{ marginTop: 16, color: '#666', fontSize: 12 }}>
          <div><strong>角色说明:</strong></div>
          {Object.entries(roleConfig).map(([key, config]) => (
            <div key={key}>• {config.label}: {config.description}</div>
          ))}
        </div>
      </Card>

      {/* Workloads */}
      <Card 
        title={
          <Space>
            <DeploymentUnitOutlined />
            工作负载
            {workloadSummary && (
              <Text type="secondary" style={{ fontSize: 14 }}>
                ({workloadSummary.totalPods} Pod)
              </Text>
            )}
          </Space>
        }
        style={{ marginBottom: 16 }} 
        className="glass-card"
        extra={
          <Space>
            <Segmented
              size="small"
              value={workloadFilter}
              onChange={(value) => setWorkloadFilter(value as string)}
              options={[
                { label: '全部', value: 'all' },
                { label: 'Deployment', value: 'Deployment' },
                { label: 'StatefulSet', value: 'StatefulSet' },
                { label: 'Job', value: 'Job' },
                { label: 'CronJob', value: 'CronJob' },
                { label: 'Pod', value: 'Pod' },
              ]}
            />
            <Button 
              icon={<ReloadOutlined />} 
              onClick={() => { refetchSummary(); refetchWorkloads(); }}
            >
              刷新
            </Button>
          </Space>
        }
      >
        {/* Summary Stats */}
        {workloadSummary && (
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={4}>
              <Statistic title="Deployment" value={workloadSummary.deployments} />
            </Col>
            <Col span={4}>
              <Statistic title="StatefulSet" value={workloadSummary.statefulSets} />
            </Col>
            <Col span={4}>
              <Statistic title="Job" value={workloadSummary.jobs} />
            </Col>
            <Col span={4}>
              <Statistic title="CronJob" value={workloadSummary.cronJobs} />
            </Col>
            <Col span={4}>
              <Statistic title="独立 Pod" value={workloadSummary.pods} />
            </Col>
            <Col span={4}>
              <Statistic title="总 Pod 数" value={workloadSummary.totalPods} />
            </Col>
          </Row>
        )}

        {/* Workload Table */}
        {workloadsLoading ? (
          <div style={{ textAlign: 'center', padding: 20 }}><Spin /></div>
        ) : workloads.length === 0 ? (
          <Empty description="暂无工作负载" />
        ) : (
          <Table
            dataSource={workloadFilter === 'all' ? workloads : workloads.filter(w => w.kind === workloadFilter)}
            columns={[
              {
                title: '类型',
                dataIndex: 'kind',
                key: 'kind',
                width: 100,
                render: (kind: string) => {
                  const colors: Record<string, string> = {
                    Deployment: 'blue',
                    StatefulSet: 'purple',
                    Job: 'orange',
                    CronJob: 'cyan',
                    Pod: 'green',
                  };
                  return <Tag color={colors[kind] || 'default'}>{kind}</Tag>;
                },
              },
              {
                title: '名称',
                dataIndex: 'name',
                key: 'name',
                ellipsis: true,
              },
              {
                title: '副本',
                key: 'replicas',
                width: 100,
                render: (_: unknown, record: Workload) => {
                  if (record.kind === 'CronJob') return '-';
                  return `${record.ready}/${record.replicas}`;
                },
              },
              {
                title: '状态',
                dataIndex: 'status',
                key: 'status',
                width: 100,
                render: (status: string) => {
                  const colors: Record<string, string> = {
                    Running: 'success',
                    Pending: 'processing',
                    Failed: 'error',
                    Succeeded: 'success',
                    Active: 'success',
                    Suspended: 'warning',
                    Progressing: 'processing',
                  };
                  return <Tag color={colors[status] || 'default'}>{status}</Tag>;
                },
              },
              {
                title: '镜像',
                dataIndex: 'image',
                key: 'image',
                ellipsis: true,
                render: (image: string) => (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {image?.split('/').pop() || '-'}
                  </Text>
                ),
              },
              {
                title: '创建时间',
                dataIndex: 'createdAt',
                key: 'createdAt',
                width: 150,
                render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm'),
              },
            ]}
            rowKey={(record) => `${record.kind}-${record.name}`}
            pagination={{ pageSize: 10 }}
            size="small"
          />
        )}
      </Card>

      {/* Edit Form */}
      <Card title="编辑项目" style={{ marginBottom: 16 }} className="glass-card">
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{
            displayName: project.displayName,
            description: project.description,
          }}
        >
          <Form.Item name="displayName" label="显示名称">
            <Input />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={3} />
          </Form.Item>

          <Form.Item style={{ marginTop: 16 }}>
            <Button 
              type="primary" 
              htmlType="submit" 
              icon={<SaveOutlined />}
              loading={updateMutation.isPending}
            >
              保存修改
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* Add Member Modal */}
      <Modal
        title="添加成员"
        open={addMemberModalVisible}
        onCancel={() => {
          setAddMemberModalVisible(false);
          addMemberForm.resetFields();
        }}
        footer={null}
      >
        <Form
          form={addMemberForm}
          layout="vertical"
          onFinish={handleAddMember}
          initialValues={{ role: 'edit' }}
        >
          <Form.Item
            name="user"
            label="选择用户"
            rules={[{ required: true, message: '请选择用户' }]}
          >
            <Select
              placeholder="选择要添加的用户"
              showSearch
              optionFilterProp="label"
              options={availableUsers.map(u => ({ 
                value: u.email, 
                label: u.email 
              }))}
            />
          </Form.Item>

          <Form.Item
            name="role"
            label="分配角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Radio.Group>
              {Object.entries(roleConfig).map(([value, config]) => (
                <Radio key={value} value={value} style={{ display: 'block', marginBottom: 8 }}>
                  <Tag color={config.color}>{config.label}</Tag>
                  <span style={{ color: '#666', fontSize: 12 }}> - {config.description}</span>
                </Radio>
              ))}
            </Radio.Group>
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => {
                setAddMemberModalVisible(false);
                addMemberForm.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={updateMutation.isPending}>
                添加
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ProjectDetail;

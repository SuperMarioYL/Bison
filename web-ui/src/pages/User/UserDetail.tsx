import React, { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { 
  Card, Descriptions, Table, Tag, Space, Statistic, Row, Col, Spin, Empty, 
  Button, Modal, Form, Select, Radio, message, Popconfirm 
} from 'antd';
import { 
  UserOutlined, TeamOutlined, ClockCircleOutlined, DollarOutlined,
  ArrowLeftOutlined, EditOutlined, DeleteOutlined, PlusOutlined
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { 
  getUser, updateUser, deleteUser, setUserStatus, 
  addUserToTeam, removeUserFromTeam, addUserToProject, removeUserFromProject, updateUserProjectRole,
  getTeams, getProjects, Team, Project, UserDetail as UserDetailType
} from '../../services/api';
import dayjs from 'dayjs';

const UserDetail: React.FC = () => {
  const { email } = useParams<{ email: string }>();
  const decodedEmail = email ? decodeURIComponent(email) : '';
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [addTeamModalVisible, setAddTeamModalVisible] = useState(false);
  const [addProjectModalVisible, setAddProjectModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [addTeamForm] = Form.useForm();
  const [addProjectForm] = Form.useForm();

  const { data: userData, isLoading: loadingUser } = useQuery({
    queryKey: ['user', decodedEmail],
    queryFn: () => getUser(decodedEmail),
    enabled: !!decodedEmail,
  });

  const { data: teamsData } = useQuery({
    queryKey: ['teams'],
    queryFn: () => getTeams(),
  });

  const { data: projectsData } = useQuery({
    queryKey: ['projects'],
    queryFn: () => getProjects(),
  });

  const user: UserDetailType | undefined = userData?.data;
  const allTeams: Team[] = teamsData?.data?.items || [];
  const allProjects: Project[] = projectsData?.data?.items || [];

  // Mutations
  const updateMutation = useMutation({
    mutationFn: (data: { displayName?: string; status?: string }) => updateUser(decodedEmail, data),
    onSuccess: () => {
      message.success('用户更新成功');
      setEditModalVisible(false);
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('更新失败: ' + error.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteUser(decodedEmail),
    onSuccess: () => {
      message.success('用户删除成功');
      navigate('/users');
    },
    onError: (error: Error) => {
      message.error('删除失败: ' + error.message);
    },
  });

  const statusMutation = useMutation({
    mutationFn: (status: string) => setUserStatus(decodedEmail, status),
    onSuccess: () => {
      message.success('状态更新成功');
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('更新失败: ' + error.message);
    },
  });

  const addTeamMutation = useMutation({
    mutationFn: (teamName: string) => addUserToTeam(decodedEmail, teamName),
    onSuccess: () => {
      message.success('已添加到团队');
      setAddTeamModalVisible(false);
      addTeamForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('添加失败: ' + error.message);
    },
  });

  const removeTeamMutation = useMutation({
    mutationFn: (teamName: string) => removeUserFromTeam(decodedEmail, teamName),
    onSuccess: () => {
      message.success('已从团队移除');
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('移除失败: ' + error.message);
    },
  });

  const addProjectMutation = useMutation({
    mutationFn: ({ projectName, role }: { projectName: string; role: string }) => 
      addUserToProject(decodedEmail, projectName, role),
    onSuccess: () => {
      message.success('已添加到项目');
      setAddProjectModalVisible(false);
      addProjectForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('添加失败: ' + error.message);
    },
  });

  const removeProjectMutation = useMutation({
    mutationFn: (projectName: string) => removeUserFromProject(decodedEmail, projectName),
    onSuccess: () => {
      message.success('已从项目移除');
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('移除失败: ' + error.message);
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ projectName, role }: { projectName: string; role: string }) => 
      updateUserProjectRole(decodedEmail, projectName, role),
    onSuccess: () => {
      message.success('角色更新成功');
      queryClient.invalidateQueries({ queryKey: ['user', decodedEmail] });
    },
    onError: (error: Error) => {
      message.error('更新失败: ' + error.message);
    },
  });

  if (loadingUser) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />;
  }

  if (!user) {
    return <Empty description="用户不存在" />;
  }

  const handleEdit = () => {
    editForm.setFieldsValue({
      displayName: user.displayName,
      status: user.status,
    });
    setEditModalVisible(true);
  };

  const teamColumns = [
    {
      title: '团队名称',
      dataIndex: 'teamName',
      key: 'teamName',
      render: (name: string, record: { displayName?: string }) => (
        <Link to={`/teams/${name}`}>
          <Space>
            <TeamOutlined />
            <span>{record.displayName || name}</span>
          </Space>
        </Link>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: (role: string) => <Tag color="blue">{role === 'owner' ? 'Owner' : role}</Tag>,
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: { teamName: string }) => (
        <Popconfirm
          title="确定从团队中移除此用户？"
          onConfirm={() => removeTeamMutation.mutate(record.teamName)}
        >
          <Button type="link" size="small" danger>移除</Button>
        </Popconfirm>
      ),
    },
  ];

  const projectColumns = [
    {
      title: '项目名称',
      dataIndex: 'projectName',
      key: 'projectName',
      render: (name: string, record: { displayName?: string }) => (
        <Link to={`/projects/${name}`}>
          <span>{record.displayName || name}</span>
        </Link>
      ),
    },
    {
      title: '所属团队',
      dataIndex: 'teamName',
      key: 'teamName',
      render: (teamName: string) => teamName ? (
        <Link to={`/teams/${teamName}`}>{teamName}</Link>
      ) : '-',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 120,
      render: (role: string, record: { projectName: string }) => (
        <Select
          value={role}
          size="small"
          style={{ width: 100 }}
          onChange={(newRole) => updateRoleMutation.mutate({ projectName: record.projectName, role: newRole })}
          options={[
            { value: 'admin', label: 'admin' },
            { value: 'edit', label: 'edit' },
            { value: 'view', label: 'view' },
          ]}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: { projectName: string }) => (
        <Popconfirm
          title="确定从项目中移除此用户？"
          onConfirm={() => removeProjectMutation.mutate(record.projectName)}
        >
          <Button type="link" size="small" danger>移除</Button>
        </Popconfirm>
      ),
    },
  ];

  // Get teams user is not in
  const availableTeams = allTeams.filter(t => 
    !user.teams?.some(ut => ut.teamName === t.name)
  );

  // Get projects user is not in
  const availableProjects = allProjects.filter(p => 
    !user.projects?.some(up => up.projectName === p.name)
  );

  const roleLabels: Record<string, string> = {
    admin: '管理员 - 完全控制',
    edit: '编辑者 - 可修改资源',
    view: '只读 - 仅查看',
  };

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/users')}>
            返回
          </Button>
          <h2 style={{ margin: 0 }}>
            <UserOutlined style={{ marginRight: 8 }} />
            {decodedEmail}
          </h2>
          <Tag color={user.status === 'active' ? 'green' : 'default'}>
            {user.status === 'active' ? '● 启用' : '○ 禁用'}
          </Tag>
        </Space>
        <Space>
          <Button icon={<EditOutlined />} onClick={handleEdit}>
            编辑
          </Button>
          <Button 
            onClick={() => statusMutation.mutate(user.status === 'active' ? 'disabled' : 'active')}
          >
            {user.status === 'active' ? '禁用' : '启用'}
          </Button>
          <Popconfirm
            title="确定删除此用户？"
            description="删除后，用户将从所有团队和项目中移除"
            onConfirm={() => deleteMutation.mutate()}
          >
            <Button danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        {/* Basic Info */}
        <Col span={24}>
          <Card title="基本信息">
            <Descriptions column={3}>
              <Descriptions.Item label="显示名称">{user.displayName || '-'}</Descriptions.Item>
              <Descriptions.Item label="邮箱">{user.email}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={user.status === 'active' ? 'green' : 'default'}>
                  {user.status === 'active' ? '启用' : '禁用'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="来源">
                <Tag color={user.source === 'oidc' ? 'blue' : 'purple'}>
                  {user.source === 'oidc' ? 'OIDC (自动同步)' : '手动创建'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {user.createdAt ? dayjs(user.createdAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="最后登录">
                {user.lastLogin ? dayjs(user.lastLogin).format('YYYY-MM-DD HH:mm:ss') : '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        {/* Resource Usage */}
        <Col span={24}>
          <Card 
            title="资源使用 (过去7天)" 
            extra={<Link to="/reports">查看详细账单</Link>}
          >
            <Row gutter={16}>
              <Col span={6}>
                <Statistic
                  title="CPU 使用"
                  value={user.usage?.cpuCoreHours?.toFixed(2) || 0}
                  suffix="核·时"
                  prefix={<ClockCircleOutlined />}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="内存使用"
                  value={user.usage?.ramGBHours?.toFixed(2) || 0}
                  suffix="GB·时"
                  prefix={<ClockCircleOutlined />}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="GPU 使用"
                  value={user.usage?.gpuHours?.toFixed(2) || 0}
                  suffix="卡·时"
                  prefix={<ClockCircleOutlined />}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="总消费"
                  value={user.usage?.totalCost?.toFixed(2) || 0}
                  prefix={<DollarOutlined />}
                  suffix="¥"
                />
              </Col>
            </Row>
          </Card>
        </Col>

        {/* Teams */}
        <Col span={24}>
          <Card 
            title={`所属团队 (${user.teams?.length || 0})`}
            extra={
              <Button 
                type="primary" 
                size="small" 
                icon={<PlusOutlined />}
                onClick={() => setAddTeamModalVisible(true)}
                disabled={availableTeams.length === 0}
              >
                添加到团队
              </Button>
            }
          >
            <Table
              columns={teamColumns}
              dataSource={user.teams || []}
              rowKey="teamName"
              pagination={false}
              locale={{ emptyText: '暂未加入任何团队' }}
            />
          </Card>
        </Col>

        {/* Projects */}
        <Col span={24}>
          <Card 
            title={`关联项目 (${user.projects?.length || 0})`}
            extra={
              <Button 
                type="primary" 
                size="small" 
                icon={<PlusOutlined />}
                onClick={() => setAddProjectModalVisible(true)}
                disabled={availableProjects.length === 0}
              >
                添加到项目
              </Button>
            }
          >
            <Table
              columns={projectColumns}
              dataSource={user.projects || []}
              rowKey="projectName"
              pagination={false}
              locale={{ emptyText: '暂未加入任何项目' }}
            />
            <div style={{ marginTop: 16, color: '#666', fontSize: 12 }}>
              <div><strong>角色说明:</strong></div>
              <div>• admin: 完全控制，可创建/删除/修改所有资源</div>
              <div>• edit: 可修改大部分资源，不能改权限</div>
              <div>• view: 只读访问</div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* Edit User Modal */}
      <Modal
        title="编辑用户"
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        footer={null}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={(values) => updateMutation.mutate(values)}
        >
          <Form.Item name="displayName" label="显示名称">
            <input className="ant-input" placeholder="用户显示名称" />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Radio.Group>
              <Radio value="active">启用</Radio>
              <Radio value="disabled">禁用</Radio>
            </Radio.Group>
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setEditModalVisible(false)}>取消</Button>
              <Button type="primary" htmlType="submit" loading={updateMutation.isPending}>
                保存
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Add to Team Modal */}
      <Modal
        title="添加到团队"
        open={addTeamModalVisible}
        onCancel={() => {
          setAddTeamModalVisible(false);
          addTeamForm.resetFields();
        }}
        footer={null}
      >
        <Form
          form={addTeamForm}
          layout="vertical"
          onFinish={(values) => addTeamMutation.mutate(values.teamName)}
        >
          <Form.Item
            name="teamName"
            label="选择团队"
            rules={[{ required: true, message: '请选择团队' }]}
          >
            <Select
              placeholder="选择要加入的团队"
              showSearch
              optionFilterProp="label"
              options={availableTeams.map(t => ({ 
                value: t.name, 
                label: t.displayName || t.name 
              }))}
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => {
                setAddTeamModalVisible(false);
                addTeamForm.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={addTeamMutation.isPending}>
                添加
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Add to Project Modal */}
      <Modal
        title="添加到项目"
        open={addProjectModalVisible}
        onCancel={() => {
          setAddProjectModalVisible(false);
          addProjectForm.resetFields();
        }}
        footer={null}
      >
        <Form
          form={addProjectForm}
          layout="vertical"
          onFinish={(values) => addProjectMutation.mutate(values)}
          initialValues={{ role: 'edit' }}
        >
          <Form.Item
            name="projectName"
            label="选择项目"
            rules={[{ required: true, message: '请选择项目' }]}
          >
            <Select
              placeholder="选择要加入的项目"
              showSearch
              optionFilterProp="label"
              options={availableProjects.map(p => ({ 
                value: p.name, 
                label: `${p.displayName || p.name} (${p.team || '无团队'})` 
              }))}
            />
          </Form.Item>
          <Form.Item
            name="role"
            label="分配角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Radio.Group>
              {Object.entries(roleLabels).map(([value, label]) => (
                <Radio key={value} value={value} style={{ display: 'block', marginBottom: 8 }}>
                  {label}
                </Radio>
              ))}
            </Radio.Group>
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => {
                setAddProjectModalVisible(false);
                addProjectForm.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={addProjectMutation.isPending}>
                添加
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default UserDetail;

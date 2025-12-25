import React, { useState } from 'react';
import { Table, Input, Card, Tag, Space, Button, Select, Modal, Form, Radio, message, Popconfirm } from 'antd';
import { UserOutlined, PlusOutlined, SearchOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getUsers, createUser, deleteUser, setUserStatus, getTeams, User, Team } from '../../services/api';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const UserList: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [sourceFilter, setSourceFilter] = useState('all');
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: usersData, isLoading } = useQuery({
    queryKey: ['users', searchQuery, statusFilter, sourceFilter],
    queryFn: () => getUsers(searchQuery, statusFilter, sourceFilter),
  });

  const { data: teamsData } = useQuery({
    queryKey: ['teams'],
    queryFn: () => getTeams(),
  });

  const teams: Team[] = teamsData?.data?.items || [];
  const users: User[] = usersData?.data?.items || [];

  const createMutation = useMutation({
    mutationFn: (data: { email: string; displayName?: string; status?: string; initialTeam?: string }) => 
      createUser(data),
    onSuccess: () => {
      message.success('用户创建成功');
      setCreateModalVisible(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
    onError: (error: Error) => {
      message.error('创建失败: ' + error.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (email: string) => deleteUser(email),
    onSuccess: () => {
      message.success('用户删除成功');
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
    onError: (error: Error) => {
      message.error('删除失败: ' + error.message);
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ email, status }: { email: string; status: string }) => setUserStatus(email, status),
    onSuccess: () => {
      message.success('状态更新成功');
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
    onError: (error: Error) => {
      message.error('更新失败: ' + error.message);
    },
  });

  const handleCreate = (values: { email: string; displayName?: string; status: string; initialTeam?: string }) => {
    createMutation.mutate(values);
  };

  const handleStatusToggle = (user: User) => {
    const newStatus = user.status === 'active' ? 'disabled' : 'active';
    statusMutation.mutate({ email: user.email, status: newStatus });
  };

  const columns = [
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      render: (email: string) => (
        <Link to={`/users/${encodeURIComponent(email)}`}>
          <Space>
            <UserOutlined />
            <span style={{ fontWeight: 500 }}>{email}</span>
          </Space>
        </Link>
      ),
    },
    {
      title: '显示名',
      dataIndex: 'displayName',
      key: 'displayName',
      width: 150,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'default'}>
          {status === 'active' ? '● 启用' : '○ 禁用'}
        </Tag>
      ),
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (source: string) => (
        <Tag color={source === 'oidc' ? 'blue' : 'purple'}>
          {source === 'oidc' ? 'OIDC' : '手动'}
        </Tag>
      ),
    },
    {
      title: '最后登录',
      dataIndex: 'lastLogin',
      key: 'lastLogin',
      width: 150,
      render: (lastLogin: string) => 
        lastLogin ? dayjs(lastLogin).fromNow() : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: unknown, record: User) => (
        <Space>
          <Link to={`/users/${encodeURIComponent(record.email)}`}>
            <Button type="link" size="small" icon={<EditOutlined />}>
              详情
            </Button>
          </Link>
          <Button 
            type="link" 
            size="small" 
            onClick={() => handleStatusToggle(record)}
          >
            {record.status === 'active' ? '禁用' : '启用'}
          </Button>
          <Popconfirm
            title="确定删除此用户？"
            description="删除后，用户将从所有团队和项目中移除"
            onConfirm={() => deleteMutation.mutate(record.email)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>
          <UserOutlined style={{ marginRight: 8 }} />
          用户管理
        </h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)}>
          创建用户
        </Button>
      </div>

      <Card>
        <div style={{ marginBottom: 16, display: 'flex', gap: 16 }}>
          <Input.Search
            placeholder="搜索用户邮箱或名称"
            prefix={<SearchOutlined />}
            style={{ width: 300 }}
            onSearch={setSearchQuery}
            allowClear
          />
          <Select
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 120 }}
            options={[
              { value: 'all', label: '全部状态' },
              { value: 'active', label: '已启用' },
              { value: 'disabled', label: '已禁用' },
            ]}
          />
          <Select
            value={sourceFilter}
            onChange={setSourceFilter}
            style={{ width: 120 }}
            options={[
              { value: 'all', label: '全部来源' },
              { value: 'manual', label: '手动创建' },
              { value: 'oidc', label: 'OIDC同步' },
            ]}
          />
        </div>

        <Table
          columns={columns}
          dataSource={users}
          loading={isLoading}
          rowKey="email"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>

      {/* Create User Modal */}
      <Modal
        title="创建用户"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          form.resetFields();
        }}
        footer={null}
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreate}
          initialValues={{ status: 'active' }}
        >
          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '请输入有效的邮箱地址' },
            ]}
          >
            <Input placeholder="user@example.com" />
          </Form.Item>

          <Form.Item
            name="displayName"
            label="显示名称"
          >
            <Input placeholder="用户显示名称（可选）" />
          </Form.Item>

          <Form.Item
            name="status"
            label="初始状态"
          >
            <Radio.Group>
              <Radio value="active">启用</Radio>
              <Radio value="disabled">禁用</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item
            name="initialTeam"
            label="添加到团队（可选）"
          >
            <Select
              placeholder="选择团队"
              allowClear
              options={teams.map(t => ({ value: t.name, label: t.displayName || t.name }))}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => {
                setCreateModalVisible(false);
                form.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending}>
                创建
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default UserList;

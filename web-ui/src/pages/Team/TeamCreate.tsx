import { CloudServerOutlined, DeleteOutlined, PlusOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button, Card,
  Checkbox, Divider,
  Form, Input, message, Modal, Radio, Select, Space,
  Table, Tag, Typography
} from 'antd';
import React, { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ResourceQuotaInput from '../../components/ResourceQuotaInput';
import {
  createTeam,
  getSharedNodes,
  getUsers,
  NodeInfo,
  OwnerRef,
  TeamMode,
  User
} from '../../services/api';

const { Title, Text } = Typography;
const { TextArea } = Input;

// Reserved team names that cannot be used
const RESERVED_TEAM_NAMES = ['shared', 'disabled', 'unmanaged', 'system', 'default', 'admin'];

const TeamCreate: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [owners, setOwners] = useState<OwnerRef[]>([]);
  const [addOwnerModalVisible, setAddOwnerModalVisible] = useState(false);
  const [addOwnerForm] = Form.useForm();
  const [mode, setMode] = useState<TeamMode>('shared');
  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
  const [quota, setQuota] = useState<Record<string, string>>({});

  const { data: usersData } = useQuery({
    queryKey: ['users'],
    queryFn: () => getUsers(),
  });

  const users: User[] = usersData?.data?.items || [];

  // Fetch shared nodes (available for exclusive assignment)
  const { data: sharedNodes } = useQuery({
    queryKey: ['sharedNodes'],
    queryFn: () => getSharedNodes().then(res => res.data.items),
    enabled: mode === 'exclusive',
  });

  // Calculate total resources for selected nodes
  const selectedNodesResources = useMemo(() => {
    if (!sharedNodes || selectedNodes.length === 0) return null;

    const totals: Record<string, number> = {};
    selectedNodes.forEach(nodeName => {
      const node = sharedNodes.find(n => n.name === nodeName);
      if (node) {
        Object.entries(node.capacity || {}).forEach(([key, value]) => {
          // Parse value
          const numValue = parseFloat(value) || 0;
          totals[key] = (totals[key] || 0) + numValue;
        });
      }
    });

    return totals;
  }, [sharedNodes, selectedNodes]);

  const createMutation = useMutation({
    mutationFn: createTeam,
    onSuccess: () => {
      message.success('团队创建成功');
      queryClient.invalidateQueries({ queryKey: ['teams'] });
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
      queryClient.invalidateQueries({ queryKey: ['sharedNodes'] });
      navigate('/teams');
    },
    onError: (error: any) => {
      message.error(`创建失败: ${error.response?.data?.error || error.message}`);
    },
  });

  const onFinish = (values: {
    name: string;
    displayName?: string;
    description?: string;
  }) => {
    if (owners.length === 0) {
      message.error('请至少添加一个所有者');
      return;
    }

    if (mode === 'exclusive' && selectedNodes.length === 0) {
      message.error('独占模式需要选择至少一个节点');
      return;
    }

    const team = {
      name: values.name,
      displayName: values.displayName || values.name,
      description: values.description,
      owners: owners,
      mode: mode,
      exclusiveNodes: mode === 'exclusive' ? selectedNodes : undefined,
      // Only include quota for shared mode (empty object for exclusive)
      quota: mode === 'shared' ? quota : {},
    };

    createMutation.mutate(team);
  };

  const handleAddOwner = (values: { kind: 'User' | 'Group'; name: string; groupName?: string }) => {
    const newOwner: OwnerRef = {
      kind: values.kind,
      name: values.kind === 'User' ? values.name : values.groupName!,
    };

    // Check for duplicates
    if (owners.some(o => o.kind === newOwner.kind && o.name === newOwner.name)) {
      message.warning('该所有者已存在');
      return;
    }

    setOwners([...owners, newOwner]);
    setAddOwnerModalVisible(false);
    addOwnerForm.resetFields();
  };

  const handleRemoveOwner = (owner: OwnerRef) => {
    setOwners(owners.filter(o => !(o.kind === owner.kind && o.name === owner.name)));
  };

  const handleNodeSelect = (nodeName: string, checked: boolean) => {
    if (checked) {
      setSelectedNodes([...selectedNodes, nodeName]);
    } else {
      setSelectedNodes(selectedNodes.filter(n => n !== nodeName));
    }
  };

  const ownerColumns = [
    {
      title: '类型',
      dataIndex: 'kind',
      key: 'kind',
      width: 100,
      render: (kind: string) => (
        <Tag color={kind === 'User' ? 'green' : 'blue'} icon={kind === 'User' ? <UserOutlined /> : <TeamOutlined />}>
          {kind === 'User' ? '用户' : '组'}
        </Tag>
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: OwnerRef) => (
        <Button 
          type="link" 
          danger 
          size="small" 
          icon={<DeleteOutlined />}
          onClick={() => handleRemoveOwner(record)}
        >
          移除
        </Button>
      ),
    },
  ];

  const nodeColumns = [
    {
      title: '选择',
      key: 'select',
      width: 60,
      render: (_: unknown, record: NodeInfo) => (
        <Checkbox
          checked={selectedNodes.includes(record.name)}
          onChange={(e) => handleNodeSelect(record.name, e.target.checked)}
        />
      ),
    },
    {
      title: '节点名',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '架构',
      dataIndex: 'architecture',
      key: 'architecture',
      width: 100,
      render: (arch: string) => <Tag>{arch}</Tag>,
    },
    {
      title: 'CPU',
      key: 'cpu',
      width: 80,
      render: (_: unknown, record: NodeInfo) => record.capacity?.cpu || '-',
    },
    {
      title: '内存',
      key: 'memory',
      width: 100,
      render: (_: unknown, record: NodeInfo) => record.capacity?.memory || '-',
    },
    {
      title: 'IP',
      dataIndex: 'internalIP',
      key: 'internalIP',
      width: 130,
    },
  ];

  return (
    <div>
      <Title level={2}>创建团队</Title>
      <Text type="secondary">
        团队对应 Capsule Tenant，用于管理一组项目和资源配额
      </Text>

      <Card style={{ marginTop: 16 }} className="glass-card">
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
        >
          <Form.Item
            name="name"
            label="团队标识"
            rules={[
              { required: true, message: '请输入团队标识' },
              { pattern: /^[a-z0-9][a-z0-9-]*[a-z0-9]$/, message: '只能包含小写字母、数字和连字符' },
              { 
                validator: (_, value) => {
                  if (value && RESERVED_TEAM_NAMES.includes(value.toLowerCase())) {
                    return Promise.reject(new Error(`"${value}" 是保留名称，不能使用`));
                  }
                  return Promise.resolve();
                }
              },
            ]}
            tooltip="唯一标识，创建后不可修改"
            extra={<Text type="secondary">保留名称: {RESERVED_TEAM_NAMES.join(', ')}</Text>}
          >
            <Input placeholder="例如: team-alpha" />
          </Form.Item>

          <Form.Item
            name="displayName"
            label="显示名称"
          >
            <Input placeholder="例如: Alpha 团队" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea rows={3} placeholder="团队描述..." />
          </Form.Item>

          {/* Owners Section */}
          <div style={{ marginBottom: 24 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <div>
                <Text strong>团队所有者</Text>
                <Text type="secondary" style={{ marginLeft: 8 }}>
                  (用户或 OIDC 用户组)
                </Text>
              </div>
              <Button 
                type="primary" 
                size="small" 
                icon={<PlusOutlined />}
                onClick={() => setAddOwnerModalVisible(true)}
              >
                添加所有者
              </Button>
            </div>
            <Table
              columns={ownerColumns}
              dataSource={owners}
              rowKey={(record) => `${record.kind}-${record.name}`}
              pagination={false}
              size="small"
              locale={{ emptyText: '请添加至少一个所有者' }}
            />
          </div>

          <Divider />

          {/* Resource Mode Section */}
          <Title level={5}>资源模式</Title>
          <Text type="secondary">
            选择团队的资源使用模式
          </Text>
          
          <Form.Item style={{ marginTop: 16 }}>
            <Radio.Group value={mode} onChange={(e) => {
              setMode(e.target.value);
              if (e.target.value === 'shared') {
                setSelectedNodes([]);
              }
            }}>
              <Space direction="vertical">
                <Radio value="shared">
                  <Space>
                    <span>共享模式</span>
                    <Text type="secondary">使用共享节点池，按配额限制资源使用</Text>
                  </Space>
                </Radio>
                <Radio value="exclusive">
                  <Space>
                    <span>独占模式</span>
                    <Text type="secondary">独占指定节点，节点资源完全归该团队使用</Text>
                  </Space>
                </Radio>
              </Space>
            </Radio.Group>
          </Form.Item>

          {/* Exclusive Node Selection */}
          {mode === 'exclusive' && (
            <Card 
              title={
                <Space>
                  <CloudServerOutlined />
                  选择独占节点
                </Space>
              }
              size="small"
              style={{ marginBottom: 24 }}
            >
              {sharedNodes && sharedNodes.length > 0 ? (
                <>
                  <Alert
                    type="info"
                    message="从共享池中选择要分配给该团队的节点"
                    description="选中的节点将从共享池中移除，仅供该团队使用"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                  <Table
                    columns={nodeColumns}
                    dataSource={sharedNodes}
                    rowKey="name"
                    pagination={false}
                    size="small"
                    scroll={{ y: 300 }}
                  />
                  {selectedNodes.length > 0 && selectedNodesResources && (
                    <div style={{ marginTop: 16, padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
                      <Text strong>已选择 {selectedNodes.length} 个节点</Text>
                      <div style={{ marginTop: 8 }}>
                        <Text type="secondary">总资源: </Text>
                        {Object.entries(selectedNodesResources)
                          .filter(([key]) => ['cpu', 'memory'].includes(key))
                          .map(([key, value]) => (
                            <Tag key={key} color="blue">
                              {key}: {typeof value === 'number' ? value.toFixed(0) : value}
                            </Tag>
                          ))}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <Alert
                  type="warning"
                  message="没有可用的共享节点"
                  description="请先在节点管理中启用节点并添加到共享池"
                  showIcon
                />
              )}
            </Card>
          )}

          {/* Quota Section - Only show for shared mode */}
          {mode === 'shared' && (
            <>
              <Divider />
              <Title level={5}>资源配额 (可选)</Title>
              <Text type="secondary">
                配置团队级别的资源配额，由 Capsule 强制执行。资源类型可在系统设置中配置。
              </Text>

              <div style={{ marginTop: 16 }}>
                <ResourceQuotaInput
                  value={quota}
                  onChange={setQuota}
                  showPrice
                />
              </div>
            </>
          )}

          <Form.Item style={{ marginTop: 24 }}>
            <Space>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending}>
                创建团队
              </Button>
              <Button onClick={() => navigate('/teams')}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* Add Owner Modal */}
      <Modal
        title="添加所有者"
        open={addOwnerModalVisible}
        onCancel={() => {
          setAddOwnerModalVisible(false);
          addOwnerForm.resetFields();
        }}
        footer={null}
      >
        <Form
          form={addOwnerForm}
          layout="vertical"
          onFinish={handleAddOwner}
          initialValues={{ kind: 'User' }}
        >
          <Form.Item name="kind" label="类型">
            <Radio.Group>
              <Radio value="User">用户</Radio>
              <Radio value="Group">组</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.kind !== cur.kind}>
            {({ getFieldValue }) => 
              getFieldValue('kind') === 'User' ? (
                <Form.Item
                  name="name"
                  label="选择用户"
                  rules={[{ required: true, message: '请选择用户' }]}
                >
                  <Select
                    placeholder="选择用户"
                    showSearch
                    optionFilterProp="label"
                    options={users.map(u => ({ value: u.email, label: u.email }))}
                  />
                </Form.Item>
              ) : (
                <Form.Item
                  name="groupName"
                  label="输入组名"
                  rules={[{ required: true, message: '请输入组名' }]}
                >
                  <Input placeholder="例如: dev-team" />
                </Form.Item>
              )
            }
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => {
                setAddOwnerModalVisible(false);
                addOwnerForm.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                添加
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TeamCreate;

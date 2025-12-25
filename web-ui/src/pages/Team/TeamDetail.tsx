import React, { useState, useEffect, useMemo } from 'react';
import { 
  Card, Form, Input, Button, Typography, message, Space, Select, 
  Descriptions, Spin, Tag, Table, Statistic, Row, Col, Divider, Popconfirm,
  Modal, Radio, Alert, Checkbox, InputNumber, Timeline
} from 'antd';
import { 
  ArrowLeftOutlined, SaveOutlined, DeleteOutlined, PlusOutlined, 
  UserOutlined, TeamOutlined, CloudServerOutlined, WalletOutlined,
  WarningOutlined, DollarOutlined, HistoryOutlined, RocketOutlined,
  PlayCircleOutlined, PauseCircleOutlined
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { 
  getTeam, updateTeam, deleteTeam, getProjects, getUsers, 
  getSharedNodes, getTeamNodes,
  getTeamBalance, rechargeTeam, getRechargeHistory, getAutoRechargeConfig,
  updateAutoRechargeConfig, suspendTeam, resumeTeam, getBillingConfig,
  Project, OwnerRef, User, TeamMode, NodeInfo, RechargeRecord, AutoRechargeConfig
} from '../../services/api';
import ResourceQuotaInput from '../../components/ResourceQuotaInput';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Title, Text } = Typography;
const { TextArea } = Input;

const TeamDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [addOwnerModalVisible, setAddOwnerModalVisible] = useState(false);
  const [addOwnerForm] = Form.useForm();
  const [quota, setQuota] = useState<Record<string, string>>({});
  const [mode, setMode] = useState<TeamMode>('shared');
  const [selectedNodes, setSelectedNodes] = useState<string[]>([]);
  const [nodeModalVisible, setNodeModalVisible] = useState(false);
  const [rechargeModalVisible, setRechargeModalVisible] = useState(false);
  const [autoRechargeModalVisible, setAutoRechargeModalVisible] = useState(false);
  const [rechargeForm] = Form.useForm();
  const [autoRechargeForm] = Form.useForm();

  const { data: teamData, isLoading } = useQuery({
    queryKey: ['team', name],
    queryFn: () => getTeam(name!).then(res => res.data),
    enabled: !!name,
  });

  const { data: projectsData } = useQuery({
    queryKey: ['projects', name],
    queryFn: () => getProjects(name).then(res => res.data),
    enabled: !!name,
  });

  const { data: usersData } = useQuery({
    queryKey: ['users'],
    queryFn: () => getUsers(),
  });

  // Fetch shared nodes (available for exclusive assignment)
  const { data: sharedNodes } = useQuery({
    queryKey: ['sharedNodes'],
    queryFn: () => getSharedNodes().then(res => res.data.items),
  });

  // Fetch team's exclusive nodes
  const { data: teamNodes } = useQuery({
    queryKey: ['teamNodes', name],
    queryFn: () => getTeamNodes(name!).then(res => res.data.items),
    enabled: !!name && mode === 'exclusive',
  });

  // Fetch balance information
  const { data: balanceData } = useQuery({
    queryKey: ['teamBalance', name],
    queryFn: () => getTeamBalance(name!).then(res => res.data),
    enabled: !!name,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  // Fetch recharge history
  const { data: rechargeHistoryData } = useQuery({
    queryKey: ['rechargeHistory', name],
    queryFn: () => getRechargeHistory(name!).then(res => res.data),
    enabled: !!name,
  });

  // Fetch auto-recharge config
  const { data: autoRechargeData } = useQuery({
    queryKey: ['autoRecharge', name],
    queryFn: () => getAutoRechargeConfig(name!).then(res => res.data),
    enabled: !!name,
  });

  // Fetch billing config for currency symbol
  const { data: billingConfig } = useQuery({
    queryKey: ['billingConfig'],
    queryFn: () => getBillingConfig().then(res => res.data),
  });

  const users: User[] = usersData?.data?.items || [];
  const balance = balanceData;
  const rechargeHistory = rechargeHistoryData?.items || [];
  const autoRecharge = autoRechargeData;
  const currencySymbol = billingConfig?.currencySymbol || '¥';

  // Set state when team data is loaded
  useEffect(() => {
    if (teamData?.team) {
      setQuota(teamData.team.quota || {});
      setMode(teamData.team.mode || 'shared');
      setSelectedNodes(teamData.team.exclusiveNodes || []);
    }
  }, [teamData?.team]);

  // All available nodes for exclusive selection
  const availableNodes = useMemo(() => {
    const nodes: NodeInfo[] = [];
    if (sharedNodes) {
      nodes.push(...sharedNodes);
    }
    if (teamNodes) {
      // Add current team's nodes that aren't in shared list
      const sharedNames = new Set(sharedNodes?.map(n => n.name) || []);
      nodes.push(...teamNodes.filter(n => !sharedNames.has(n.name)));
    }
    return nodes;
  }, [sharedNodes, teamNodes]);

  const updateMutation = useMutation({
    mutationFn: (values: Parameters<typeof updateTeam>[1]) => updateTeam(name!, values),
    onSuccess: () => {
      message.success('团队更新成功');
      queryClient.invalidateQueries({ queryKey: ['team', name] });
      queryClient.invalidateQueries({ queryKey: ['teams'] });
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
      queryClient.invalidateQueries({ queryKey: ['sharedNodes'] });
      queryClient.invalidateQueries({ queryKey: ['teamNodes', name] });
    },
    onError: (error: any) => {
      message.error(`更新失败: ${error.response?.data?.error || error.message}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteTeam(name!),
    onSuccess: () => {
      message.success('团队删除成功');
      queryClient.invalidateQueries({ queryKey: ['teams'] });
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
      queryClient.invalidateQueries({ queryKey: ['sharedNodes'] });
      navigate('/teams');
    },
    onError: (error: any) => {
      message.error(`删除失败: ${error.response?.data?.error || error.message}`);
    },
  });

  // Recharge mutation
  const rechargeMutation = useMutation({
    mutationFn: (data: { amount: number; remark?: string }) => 
      rechargeTeam(name!, { ...data, operator: 'admin' }),
    onSuccess: () => {
      message.success('充值成功');
      queryClient.invalidateQueries({ queryKey: ['teamBalance', name] });
      queryClient.invalidateQueries({ queryKey: ['rechargeHistory', name] });
      setRechargeModalVisible(false);
      rechargeForm.resetFields();
    },
    onError: (error: any) => {
      message.error(`充值失败: ${error.response?.data?.error || error.message}`);
    },
  });

  // Auto-recharge mutation
  const autoRechargeMutation = useMutation({
    mutationFn: (config: AutoRechargeConfig) => updateAutoRechargeConfig(name!, config),
    onSuccess: () => {
      message.success('自动充值配置已更新');
      queryClient.invalidateQueries({ queryKey: ['autoRecharge', name] });
      setAutoRechargeModalVisible(false);
    },
    onError: (error: any) => {
      message.error(`配置失败: ${error.response?.data?.error || error.message}`);
    },
  });

  // Suspend/Resume mutations
  const suspendMutation = useMutation({
    mutationFn: () => suspendTeam(name!),
    onSuccess: () => {
      message.success('团队已暂停');
      queryClient.invalidateQueries({ queryKey: ['team', name] });
      queryClient.invalidateQueries({ queryKey: ['teams'] });
    },
    onError: (error: any) => {
      message.error(`操作失败: ${error.response?.data?.error || error.message}`);
    },
  });

  const resumeMutation = useMutation({
    mutationFn: () => resumeTeam(name!),
    onSuccess: () => {
      message.success('团队已恢复');
      queryClient.invalidateQueries({ queryKey: ['team', name] });
      queryClient.invalidateQueries({ queryKey: ['teams'] });
    },
    onError: (error: any) => {
      message.error(`操作失败: ${error.response?.data?.error || error.message}`);
    },
  });

  const onFinish = (values: {
    displayName?: string;
    description?: string;
  }) => {
    if (mode === 'exclusive' && selectedNodes.length === 0) {
      message.error('独占模式需要选择至少一个节点');
      return;
    }

    updateMutation.mutate({
      displayName: values.displayName,
      description: values.description,
      owners: teamData?.team.owners,
      mode,
      exclusiveNodes: mode === 'exclusive' ? selectedNodes : [],
      // Only include quota for shared mode (empty object for exclusive)
      quota: mode === 'shared' ? quota : {},
    });
  };

  const handleAddOwner = (values: { kind: 'User' | 'Group'; name: string; groupName?: string }) => {
    if (!teamData?.team) return;

    const newOwner: OwnerRef = {
      kind: values.kind,
      name: values.kind === 'User' ? values.name : values.groupName!,
    };

    // Check for duplicates
    if (teamData.team.owners.some(o => o.kind === newOwner.kind && o.name === newOwner.name)) {
      message.warning('该所有者已存在');
      return;
    }

    const newOwners = [...teamData.team.owners, newOwner];
    updateMutation.mutate({
      owners: newOwners,
    });
    setAddOwnerModalVisible(false);
    addOwnerForm.resetFields();
  };

  const handleRemoveOwner = (owner: OwnerRef) => {
    if (!teamData?.team) return;

    const newOwners = teamData.team.owners.filter(
      o => !(o.kind === owner.kind && o.name === owner.name)
    );

    if (newOwners.length === 0) {
      message.error('团队必须至少有一个所有者');
      return;
    }

    updateMutation.mutate({
      owners: newOwners,
    });
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
        <Popconfirm
          title="确定移除此所有者？"
          onConfirm={() => handleRemoveOwner(record)}
        >
          <Button type="link" danger size="small" icon={<DeleteOutlined />}>
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const projectColumns = [
    {
      title: '项目名称',
      dataIndex: 'displayName',
      key: 'displayName',
      render: (displayName: string, record: Project) => (
        <a onClick={() => navigate(`/projects/${record.name}`)}>{displayName}</a>
      ),
    },
    {
      title: '标识',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'Active' ? 'success' : 'default'}>{status}</Tag>
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
      title: '当前状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string, record: NodeInfo) => {
        if (record.team === name) {
          return <Tag color="success">当前独占</Tag>;
        }
        if (status === 'shared') {
          return <Tag color="processing">共享池</Tag>;
        }
        return <Tag>{status}</Tag>;
      },
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
  ];

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 50 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!teamData?.team) {
    return <div>团队不存在</div>;
  }

  const team = teamData.team;
  const usage = teamData.usage;

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/teams')}>
          返回
        </Button>
      </Space>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          <Title level={2} style={{ margin: 0 }}>{team.displayName || team.name}</Title>
          <Tag color={team.mode === 'exclusive' ? 'success' : 'processing'}>
            {team.mode === 'exclusive' ? '独占模式' : '共享模式'}
          </Tag>
        </Space>
        <Popconfirm
          title="确定删除此团队？"
          description="删除团队将同时删除所有关联的项目和资源，独占节点将返回共享池"
          onConfirm={() => deleteMutation.mutate()}
          okText="确定"
          cancelText="取消"
          okButtonProps={{ danger: true }}
        >
          <Button danger icon={<DeleteOutlined />} loading={deleteMutation.isPending}>
            删除团队
          </Button>
        </Popconfirm>
      </div>

      {/* Balance Card */}
      <Card 
        title={
          <Space>
            <WalletOutlined />
            账户余额
          </Space>
        }
        style={{ marginBottom: 16 }} 
        className="glass-card"
        extra={
          <Space>
            {team.suspended ? (
              <Popconfirm
                title="恢复团队？"
                description="恢复团队后，所有工作负载将重新启动"
                onConfirm={() => resumeMutation.mutate()}
              >
                <Button 
                  icon={<PlayCircleOutlined />}
                  type="primary"
                  loading={resumeMutation.isPending}
                >
                  恢复团队
                </Button>
              </Popconfirm>
            ) : (
              <Popconfirm
                title="暂停团队？"
                description="暂停团队后，所有工作负载将被缩容或删除"
                onConfirm={() => suspendMutation.mutate()}
              >
                <Button 
                  icon={<PauseCircleOutlined />}
                  danger
                  loading={suspendMutation.isPending}
                >
                  暂停团队
                </Button>
              </Popconfirm>
            )}
            <Button 
              type="primary"
              icon={<DollarOutlined />}
              onClick={() => setRechargeModalVisible(true)}
            >
              充值
            </Button>
            <Button 
              icon={<RocketOutlined />}
              onClick={() => {
                autoRechargeForm.setFieldsValue({
                  enabled: autoRecharge?.enabled || false,
                  amount: autoRecharge?.amount || 1000,
                  schedule: autoRecharge?.schedule || 'monthly',
                  dayOfWeek: autoRecharge?.dayOfWeek || 1,
                  dayOfMonth: autoRecharge?.dayOfMonth || 1,
                });
                setAutoRechargeModalVisible(true);
              }}
            >
              自动充值
            </Button>
          </Space>
        }
      >
        {team.suspended && (
          <Alert
            type="error"
            message="团队已暂停"
            description="由于余额不足，该团队已被暂停。所有工作负载已被缩容或删除。请充值后恢复团队。"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}
        
        <Row gutter={16}>
          <Col span={6}>
            <Statistic 
              title="当前余额" 
              value={balance?.amount || 0} 
              precision={2}
              prefix={currencySymbol}
              valueStyle={{ 
                color: (balance?.amount || 0) < 0 ? '#ff4d4f' : 
                       (balance?.amount || 0) < 100 ? '#faad14' : '#52c41a' 
              }}
            />
          </Col>
          <Col span={6}>
            <Statistic 
              title="日均消耗" 
              value={balance?.dailyConsumption || 0} 
              precision={2}
              prefix={currencySymbol}
            />
          </Col>
          <Col span={6}>
            {balance && balance.amount < 0 ? (
              <Statistic 
                title={
                  <Space>
                    <WarningOutlined style={{ color: '#ff4d4f' }} />
                    <span>已欠费</span>
                  </Space>
                }
                value={balance.overdueAt ? dayjs(balance.overdueAt).fromNow(true) : '-'}
                valueStyle={{ color: '#ff4d4f' }}
              />
            ) : balance?.estimatedOverdueAt ? (
              <Statistic 
                title="预计欠费" 
                value={dayjs(balance.estimatedOverdueAt).fromNow()}
                valueStyle={{ 
                  color: dayjs(balance.estimatedOverdueAt).diff(dayjs(), 'day') <= 7 ? '#faad14' : undefined 
                }}
              />
            ) : (
              <Statistic title="预计欠费" value="无" />
            )}
          </Col>
          <Col span={6}>
            <Statistic 
              title="宽限期剩余" 
              value={balance?.graceRemaining || '-'}
              valueStyle={{
                color: balance?.graceRemaining === '已到期' ? '#ff4d4f' : undefined
              }}
            />
          </Col>
        </Row>

        {/* Auto Recharge Status */}
        {autoRecharge?.enabled && (
          <Alert
            type="info"
            message={
              <Space>
                <RocketOutlined />
                <span>
                  自动充值已启用：每{autoRecharge.schedule === 'weekly' ? '周' : '月'}
                  {autoRecharge.schedule === 'weekly' 
                    ? ['日', '一', '二', '三', '四', '五', '六'][autoRecharge.dayOfWeek || 0]
                    : `${autoRecharge.dayOfMonth}日`}
                  自动充值 {currencySymbol}{autoRecharge.amount}
                </span>
              </Space>
            }
            style={{ marginTop: 16 }}
          />
        )}

        {/* Recent Transactions */}
        {rechargeHistory.length > 0 && (
          <>
            <Divider orientation="left">
              <Space>
                <HistoryOutlined />
                最近交易记录
              </Space>
            </Divider>
            <Timeline
              items={rechargeHistory.slice(0, 5).map((record: RechargeRecord) => ({
                color: record.type === 'recharge' || record.type === 'auto_recharge' ? 'green' : 'red',
                children: (
                  <div>
                    <Space>
                      <Text strong>
                        {record.type === 'recharge' ? '充值' : 
                         record.type === 'auto_recharge' ? '自动充值' : '扣费'}
                      </Text>
                      <Text 
                        style={{ 
                          color: record.amount > 0 ? '#52c41a' : '#ff4d4f',
                          fontWeight: 500
                        }}
                      >
                        {record.amount > 0 ? '+' : ''}{currencySymbol}{record.amount.toFixed(2)}
                      </Text>
                    </Space>
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {dayjs(record.timestamp).format('YYYY-MM-DD HH:mm:ss')}
                        {record.reason && ` - ${record.reason}`}
                      </Text>
                    </div>
                  </div>
                ),
              }))}
            />
          </>
        )}
      </Card>

      {/* Usage Statistics */}
      {usage && (
        <Card title="资源使用统计 (过去 7 天)" style={{ marginBottom: 16 }} className="glass-card">
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
              <Statistic title="总费用" value={usage.totalCost?.toFixed(2) || 0} prefix={currencySymbol} />
            </Col>
          </Row>
        </Card>
      )}

      {/* Team Info */}
      <Card title="团队信息" style={{ marginBottom: 16 }} className="glass-card">
        <Descriptions column={2}>
          <Descriptions.Item label="团队标识">{team.name}</Descriptions.Item>
          <Descriptions.Item label="状态">
            {team.status?.ready ? (
              <Tag color="success">正常</Tag>
            ) : (
              <Tag color="warning">{team.status?.state || '未知'}</Tag>
            )}
          </Descriptions.Item>
          <Descriptions.Item label="资源模式">
            <Tag color={team.mode === 'exclusive' ? 'success' : 'processing'}>
              {team.mode === 'exclusive' ? '独占模式' : '共享模式'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="项目数量">{team.projectCount}</Descriptions.Item>
          {team.mode === 'exclusive' && (
            <Descriptions.Item label="独占节点数">
              {team.exclusiveNodes?.length || 0}
            </Descriptions.Item>
          )}
          <Descriptions.Item label="命名空间数">{team.status?.namespaces || 0}</Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Exclusive Nodes */}
      {team.mode === 'exclusive' && teamNodes && teamNodes.length > 0 && (
        <Card 
          title={
            <Space>
              <CloudServerOutlined />
              独占节点
            </Space>
          }
          style={{ marginBottom: 16 }} 
          className="glass-card"
        >
          <Table
            columns={[
              { title: '节点名', dataIndex: 'name', key: 'name' },
              { title: '架构', dataIndex: 'architecture', key: 'architecture', render: (a: string) => <Tag>{a}</Tag> },
              { title: 'CPU', key: 'cpu', render: (_: unknown, r: NodeInfo) => r.capacity?.cpu || '-' },
              { title: '内存', key: 'memory', render: (_: unknown, r: NodeInfo) => r.capacity?.memory || '-' },
              { title: 'IP', dataIndex: 'internalIP', key: 'ip' },
            ]}
            dataSource={teamNodes}
            rowKey="name"
            pagination={false}
            size="small"
          />
        </Card>
      )}

      {/* Owners Section */}
      <Card 
        title="团队所有者" 
        style={{ marginBottom: 16 }} 
        className="glass-card"
        extra={
          <Button 
            type="primary" 
            size="small" 
            icon={<PlusOutlined />}
            onClick={() => setAddOwnerModalVisible(true)}
          >
            添加所有者
          </Button>
        }
      >
        <Table
          columns={ownerColumns}
          dataSource={team.owners}
          rowKey={(record) => `${record.kind}-${record.name}`}
          pagination={false}
          size="small"
          locale={{ emptyText: '暂无所有者' }}
        />
      </Card>

      {/* Edit Form */}
      <Card title="编辑团队" style={{ marginBottom: 16 }} className="glass-card">
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{
            displayName: team.displayName,
            description: team.description,
          }}
        >
          <Form.Item name="displayName" label="显示名称">
            <Input />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={3} />
          </Form.Item>

          <Divider>资源模式</Divider>
          
          <Form.Item>
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

          {mode === 'exclusive' && (
            <Card size="small" style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <div>
                  <Text strong>独占节点配置</Text>
                  <Text type="secondary" style={{ marginLeft: 8 }}>
                    已选择 {selectedNodes.length} 个节点
                  </Text>
                </div>
                <Button 
                  type="primary" 
                  size="small"
                  onClick={() => setNodeModalVisible(true)}
                >
                  选择节点
                </Button>
              </div>
              {selectedNodes.length > 0 && (
                <Space wrap>
                  {selectedNodes.map(nodeName => (
                    <Tag 
                      key={nodeName} 
                      closable 
                      onClose={() => handleNodeSelect(nodeName, false)}
                    >
                      {nodeName}
                    </Tag>
                  ))}
                </Space>
              )}
              {selectedNodes.length === 0 && (
                <Alert type="warning" message="独占模式需要选择至少一个节点" showIcon />
              )}
            </Card>
          )}

          {/* Quota Section - Only show for shared mode */}
          {mode === 'shared' && (
            <>
              <Divider>资源配额</Divider>
              <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                配置团队级别的资源配额，由 Capsule 强制执行。资源类型可在系统设置中配置。
              </Text>

              <ResourceQuotaInput
                value={quota}
                onChange={setQuota}
                showPrice
              />
            </>
          )}

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

      {/* Projects */}
      <Card title="关联项目" className="glass-card">
        <Table
          dataSource={projectsData?.items || []}
          columns={projectColumns}
          rowKey="name"
          pagination={false}
          locale={{ emptyText: '暂无项目' }}
        />
        <Button
          type="dashed"
          style={{ marginTop: 16 }}
          onClick={() => navigate(`/projects/create?team=${name}`)}
        >
          创建新项目
        </Button>
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
              <Button type="primary" htmlType="submit" loading={updateMutation.isPending}>
                添加
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Node Selection Modal */}
      <Modal
        title="选择独占节点"
        open={nodeModalVisible}
        onCancel={() => setNodeModalVisible(false)}
        onOk={() => setNodeModalVisible(false)}
        width={800}
      >
        <Alert
          type="info"
          message="从可用节点中选择要分配给该团队的节点"
          description="选中的节点将从共享池中移除，仅供该团队使用"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Table
          columns={nodeColumns}
          dataSource={availableNodes}
          rowKey="name"
          pagination={false}
          size="small"
          scroll={{ y: 400 }}
        />
      </Modal>

      {/* Recharge Modal */}
      <Modal
        title="充值"
        open={rechargeModalVisible}
        onCancel={() => {
          setRechargeModalVisible(false);
          rechargeForm.resetFields();
        }}
        footer={null}
      >
        <Form
          form={rechargeForm}
          layout="vertical"
          onFinish={(values) => rechargeMutation.mutate(values)}
          initialValues={{ amount: 1000 }}
        >
          <Form.Item
            name="amount"
            label="充值金额"
            rules={[
              { required: true, message: '请输入充值金额' },
              { type: 'number', min: 1, message: '金额必须大于0' }
            ]}
          >
            <InputNumber
              prefix={currencySymbol}
              style={{ width: '100%' }}
              min={1}
              precision={2}
              placeholder="请输入充值金额"
            />
          </Form.Item>

          <Form.Item
            name="remark"
            label="备注"
          >
            <TextArea rows={2} placeholder="可选，填写充值备注" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => {
                setRechargeModalVisible(false);
                rechargeForm.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={rechargeMutation.isPending}>
                确认充值
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Auto-Recharge Modal */}
      <Modal
        title="自动充值配置"
        open={autoRechargeModalVisible}
        onCancel={() => setAutoRechargeModalVisible(false)}
        footer={null}
        width={500}
      >
        <Form
          form={autoRechargeForm}
          layout="vertical"
          onFinish={(values) => autoRechargeMutation.mutate(values as AutoRechargeConfig)}
        >
          <Form.Item
            name="enabled"
            valuePropName="checked"
          >
            <Checkbox>启用自动充值</Checkbox>
          </Form.Item>

          <Form.Item
            name="amount"
            label="充值金额"
            rules={[{ required: true, message: '请输入充值金额' }]}
          >
            <InputNumber
              prefix={currencySymbol}
              style={{ width: '100%' }}
              min={1}
              precision={2}
            />
          </Form.Item>

          <Form.Item
            name="schedule"
            label="充值周期"
            rules={[{ required: true, message: '请选择充值周期' }]}
          >
            <Radio.Group>
              <Radio value="weekly">每周</Radio>
              <Radio value="monthly">每月</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.schedule !== cur.schedule}>
            {({ getFieldValue }) => 
              getFieldValue('schedule') === 'weekly' ? (
                <Form.Item
                  name="dayOfWeek"
                  label="每周几"
                  rules={[{ required: true, message: '请选择' }]}
                >
                  <Select>
                    <Select.Option value={0}>周日</Select.Option>
                    <Select.Option value={1}>周一</Select.Option>
                    <Select.Option value={2}>周二</Select.Option>
                    <Select.Option value={3}>周三</Select.Option>
                    <Select.Option value={4}>周四</Select.Option>
                    <Select.Option value={5}>周五</Select.Option>
                    <Select.Option value={6}>周六</Select.Option>
                  </Select>
                </Form.Item>
              ) : (
                <Form.Item
                  name="dayOfMonth"
                  label="每月几号"
                  rules={[{ required: true, message: '请选择' }]}
                >
                  <Select>
                    {Array.from({ length: 28 }, (_, i) => (
                      <Select.Option key={i + 1} value={i + 1}>{i + 1}号</Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              )
            }
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setAutoRechargeModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={autoRechargeMutation.isPending}>
                保存配置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TeamDetail;

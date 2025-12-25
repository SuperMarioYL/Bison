import {
  CheckCircleOutlined,
  LockOutlined,
  QuestionCircleOutlined,
  StopOutlined,
  TeamOutlined,
  UnlockOutlined
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Badge,
  Button,
  Modal,
  Popconfirm,
  Progress,
  Select,
  Space,
  Table, Tag,
  Tooltip,
  Typography,
  message
} from 'antd';
import React, { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  NodeInfo, NodeStatus,
  ResourceDefinition,
  assignNodeToTeam,
  disableNode,
  enableNode,
  getEnabledResourceConfigs,
  getNodes,
  getTeams,
  releaseNode
} from '../../services/api';

const { Text } = Typography;

// Status tag colors
const statusColors: Record<NodeStatus, string> = {
  unmanaged: 'default',
  disabled: 'error',
  shared: 'processing',
  exclusive: 'success',
};

// Status labels
const statusLabels: Record<NodeStatus, string> = {
  unmanaged: '未管理',
  disabled: '已禁用',
  shared: '共享池',
  exclusive: '独占',
};

const ClusterNodes: React.FC = () => {
  const queryClient = useQueryClient();
  const [archFilter, setArchFilter] = useState<string | undefined>(undefined);
  const [statusFilter, setStatusFilter] = useState<NodeStatus | undefined>(undefined);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [selectedTeam, setSelectedTeam] = useState<string | null>(null);

  // Fetch nodes
  const { data: nodes, isLoading } = useQuery({
    queryKey: ['managedNodes'],
    queryFn: () => getNodes().then(res => res.data.items),
    refetchInterval: 30000,
  });

  // Fetch teams for assignment
  const { data: teamsData } = useQuery({
    queryKey: ['teams'],
    queryFn: () => getTeams().then(res => res.data),
  });
  const teams = teamsData?.items;

  // Fetch resource configs
  const { data: resourceConfigs } = useQuery({
    queryKey: ['enabledResourceConfigs'],
    queryFn: () => getEnabledResourceConfigs().then(res => res.data.items),
    staleTime: 5 * 60 * 1000,
  });

  // Mutations
  const enableMutation = useMutation({
    mutationFn: enableNode,
    onSuccess: () => {
      message.success('节点已启用');
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
    },
    onError: (err: any) => {
      message.error(`启用失败: ${err.response?.data?.error || err.message}`);
    },
  });

  const disableMutation = useMutation({
    mutationFn: disableNode,
    onSuccess: () => {
      message.success('节点已禁用');
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
    },
    onError: (err: any) => {
      message.error(`禁用失败: ${err.response?.data?.error || err.message}`);
    },
  });

  const assignMutation = useMutation({
    mutationFn: ({ nodeName, teamName }: { nodeName: string; teamName: string }) => 
      assignNodeToTeam(nodeName, teamName),
    onSuccess: () => {
      message.success('节点已分配');
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
      setAssignModalVisible(false);
      setSelectedNode(null);
      setSelectedTeam(null);
    },
    onError: (err: any) => {
      message.error(`分配失败: ${err.response?.data?.error || err.message}`);
    },
  });

  const releaseMutation = useMutation({
    mutationFn: releaseNode,
    onSuccess: () => {
      message.success('节点已释放');
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
    },
    onError: (err: any) => {
      message.error(`释放失败: ${err.response?.data?.error || err.message}`);
    },
  });

  // Get unique architectures
  const architectures = useMemo(() => {
    const archs = new Set<string>();
    nodes?.forEach(node => {
      if (node.architecture) archs.add(node.architecture);
    });
    return Array.from(archs);
  }, [nodes]);

  // Filter nodes
  const filteredNodes = useMemo(() => {
    let result = nodes || [];
    if (archFilter) {
      result = result.filter(n => n.architecture === archFilter);
    }
    if (statusFilter) {
      result = result.filter(n => n.status === statusFilter);
    }
    return result;
  }, [nodes, archFilter, statusFilter]);

  // Handle assign node
  const handleAssign = (nodeName: string) => {
    setSelectedNode(nodeName);
    setAssignModalVisible(true);
  };

  const handleAssignConfirm = () => {
    if (selectedNode && selectedTeam) {
      assignMutation.mutate({ nodeName: selectedNode, teamName: selectedTeam });
    }
  };

  // Get K8s ready status from conditions
  const isNodeReady = (node: NodeInfo): boolean => {
    const readyCondition = node.conditions?.find(c => c.type === 'Ready');
    return readyCondition?.status === 'True';
  };

  // Parse resource value (raw K8s value to number)
  const parseResourceValue = (value: string): number => {
    if (!value) return 0;
    // Handle Ki, Mi, Gi, etc.
    const match = value.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti|k|M|G|T|m)?$/);
    if (match) {
      const num = parseFloat(match[1]);
      const unit = match[2];
      switch (unit) {
        case 'Ki': return num * 1024;
        case 'Mi': return num * 1024 * 1024;
        case 'Gi': return num * 1024 * 1024 * 1024;
        case 'Ti': return num * 1024 * 1024 * 1024 * 1024;
        case 'k': return num * 1000;
        case 'M': return num * 1000000;
        case 'G': return num * 1000000000;
        case 'T': return num * 1000000000000;
        case 'm': return num / 1000;
        default: return num;
      }
    }
    return parseFloat(value) || 0;
  };

  // Get node resource value by name
  const getNodeResourceValue = (node: NodeInfo, resourceName: string, type: 'capacity' | 'allocatable'): string => {
    const resources = type === 'capacity' ? node.capacity : node.allocatable;
    return resources?.[resourceName] || '0';
  };

  // Format display value with unit conversion
  const formatResourceDisplay = (rawValue: number, config: ResourceDefinition): string => {
    const divisor = config.divisor || 1;
    const value = divisor > 1 ? rawValue / divisor : rawValue;
    const unit = config.unit || '';
    
    // Format number
    const formattedValue = value < 10 
      ? value.toFixed(1) 
      : Math.round(value).toString();
    
    return unit ? `${formattedValue}${unit}` : formattedValue;
  };

  // Get all enabled resource configs for display
  const allResourceConfigs = useMemo(() => {
    if (!resourceConfigs) return [];
    return resourceConfigs;
  }, [resourceConfigs]);

  const columns = [
    {
      title: '节点名',
      dataIndex: 'name',
      key: 'name',
      fixed: 'left' as const,
      width: 200,
      render: (name: string) => (
        <Link to={`/cluster/nodes/${name}`} style={{ fontWeight: 500 }}>
          {name}
        </Link>
      ),
    },
    {
      title: 'Bison 状态',
      dataIndex: 'status',
      key: 'status',
      width: 140,
      render: (status: NodeStatus, record: NodeInfo) => (
        <Space>
          <Tag color={statusColors[status]}>
            {statusLabels[status]}
          </Tag>
          {record.team && (
            <Tooltip title={`独占团队: ${record.team}`}>
              <Tag color="blue" icon={<TeamOutlined />}>
                {record.team}
              </Tag>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: 'K8s 状态',
      key: 'ready',
      width: 100,
      render: (_: unknown, record: NodeInfo) => {
        const ready = isNodeReady(record);
        return (
          <Tag color={ready ? 'success' : 'error'}>
            {ready ? '正常' : '异常'}
          </Tag>
        );
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
      title: '资源使用',
      key: 'resources',
      minWidth: 240,
      render: (_: unknown, record: NodeInfo) => {
        if (allResourceConfigs.length === 0) {
          return <Text type="secondary">未配置</Text>;
        }

        return (
          <div style={{ lineHeight: 1.4, minWidth: 200 }}>
            {allResourceConfigs.map((config, index) => {
              const rawCapacity = parseResourceValue(getNodeResourceValue(record, config.name, 'capacity'));
              const rawAllocatable = parseResourceValue(getNodeResourceValue(record, config.name, 'allocatable'));
              const used = rawCapacity - rawAllocatable;
              const percent = rawCapacity > 0 ? Math.round((used / rawCapacity) * 100) : 0;
              const displayCapacity = formatResourceDisplay(rawCapacity, config);
              const displayUsed = formatResourceDisplay(used, config);

              // Skip resources with 0 capacity on this node
              if (rawCapacity === 0) return null;

              return (
                <Tooltip 
                  key={config.name}
                  title={`${config.displayName}: 已分配 ${displayUsed} / 总计 ${displayCapacity}`}
                >
                  <div style={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    gap: 6, 
                    marginBottom: index < allResourceConfigs.length - 1 ? 6 : 0 
                  }}>
                    <span style={{ fontSize: 12, color: '#666', width: 50, flexShrink: 0, whiteSpace: 'nowrap' }}>
                      {config.displayName.length > 6 ? config.displayName.slice(0, 6) : config.displayName}
                    </span>
                    <Progress 
                      percent={percent} 
                      size="small" 
                      format={() => displayCapacity}
                      status={percent > 90 ? 'exception' : 'normal'}
                      style={{ flex: 1, margin: 0, minWidth: 100 }}
                    />
                  </div>
                </Tooltip>
              );
            })}
          </div>
        );
      },
    },
    {
      title: 'Pod 数',
      dataIndex: 'podCount',
      key: 'podCount',
      width: 80,
      render: (count: number, record: NodeInfo) => {
        const maxPods = parseInt(record.capacity?.pods || '110', 10);
        return (
          <Tooltip title={`${count} / ${maxPods}`}>
            <Badge 
              count={count} 
              showZero 
              style={{ backgroundColor: count > maxPods * 0.8 ? '#ff4d4f' : '#52c41a' }}
            />
          </Tooltip>
        );
      },
    },
    {
      title: 'IP',
      dataIndex: 'internalIP',
      key: 'internalIP',
      width: 130,
      render: (ip: string) => <Text code>{ip}</Text>,
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right' as const,
      width: 200,
      render: (_: unknown, record: NodeInfo) => {
        const { status, name } = record;
        return (
          <Space size="small">
            {status === 'unmanaged' && (
              <Tooltip title="启用节点，加入共享池">
                <Button 
                  type="link" 
                  size="small"
                  icon={<CheckCircleOutlined />}
                  onClick={() => enableMutation.mutate(name)}
                  loading={enableMutation.isPending}
                >
                  启用
                </Button>
              </Tooltip>
            )}
            {status === 'disabled' && (
              <Tooltip title="启用节点，加入共享池">
                <Button 
                  type="link" 
                  size="small"
                  icon={<CheckCircleOutlined />}
                  onClick={() => enableMutation.mutate(name)}
                  loading={enableMutation.isPending}
                >
                  启用
                </Button>
              </Tooltip>
            )}
            {status === 'shared' && (
              <>
                <Tooltip title="分配给团队独占">
                  <Button 
                    type="link" 
                    size="small"
                    icon={<LockOutlined />}
                    onClick={() => handleAssign(name)}
                  >
                    分配
                  </Button>
                </Tooltip>
                <Popconfirm
                  title="确定禁用此节点？"
                  description="禁用后将不再接受新的调度"
                  onConfirm={() => disableMutation.mutate(name)}
                  okText="确定"
                  cancelText="取消"
                >
                  <Button 
                    type="link" 
                    size="small" 
                    danger
                    icon={<StopOutlined />}
                    loading={disableMutation.isPending}
                  >
                    禁用
                  </Button>
                </Popconfirm>
              </>
            )}
            {status === 'exclusive' && (
              <Popconfirm
                title="确定释放此节点？"
                description="释放后将返回共享池"
                onConfirm={() => releaseMutation.mutate(name)}
                okText="确定"
                cancelText="取消"
              >
                <Button 
                  type="link" 
                  size="small"
                  icon={<UnlockOutlined />}
                  loading={releaseMutation.isPending}
                >
                  释放
                </Button>
              </Popconfirm>
            )}
          </Space>
        );
      },
    },
  ];

  // Status summary
  const statusSummary = useMemo(() => {
    const summary: Record<NodeStatus, number> = {
      unmanaged: 0,
      disabled: 0,
      shared: 0,
      exclusive: 0,
    };
    nodes?.forEach(node => {
      summary[node.status]++;
    });
    return summary;
  }, [nodes]);

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <h2 style={{ margin: 0, marginBottom: 16 }}>节点管理</h2>
        
        {/* Status summary */}
        <Space style={{ marginBottom: 16 }}>
          <Tag>共 {nodes?.length || 0} 个节点</Tag>
          <Tag color="default">未管理 {statusSummary.unmanaged}</Tag>
          <Tag color="error">已禁用 {statusSummary.disabled}</Tag>
          <Tag color="processing">共享池 {statusSummary.shared}</Tag>
          <Tag color="success">独占 {statusSummary.exclusive}</Tag>
        </Space>

        {/* Filters */}
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <Space>
            <span>架构:</span>
            <Select
              allowClear
              placeholder="全部架构"
              style={{ width: 150 }}
              value={archFilter}
              onChange={setArchFilter}
              options={[
                { value: undefined, label: '全部架构' },
                ...architectures.map(arch => ({ value: arch, label: arch })),
              ]}
            />
          </Space>
          <Space>
            <span>状态:</span>
            <Select
              allowClear
              placeholder="全部状态"
              style={{ width: 150 }}
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: undefined, label: '全部状态' },
                { value: 'unmanaged', label: '未管理' },
                { value: 'disabled', label: '已禁用' },
                { value: 'shared', label: '共享池' },
                { value: 'exclusive', label: '独占' },
              ]}
            />
          </Space>
        </div>
      </div>

      <Table
        dataSource={filteredNodes}
        columns={columns}
        rowKey="name"
        loading={isLoading}
        pagination={false}
        scroll={{ x: 1200 }}
        size="small"
      />

      {/* Assign to Team Modal */}
      <Modal
        title="分配节点给团队"
        open={assignModalVisible}
        onOk={handleAssignConfirm}
        onCancel={() => {
          setAssignModalVisible(false);
          setSelectedNode(null);
          setSelectedTeam(null);
        }}
        confirmLoading={assignMutation.isPending}
        okButtonProps={{ disabled: !selectedTeam }}
      >
        <p>将节点 <Text strong>{selectedNode}</Text> 分配给团队独占使用。</p>
        <p>分配后，该节点将只接受该团队的工作负载调度。</p>
        <Select
          placeholder="选择团队"
          style={{ width: '100%', marginTop: 16 }}
          value={selectedTeam}
          onChange={setSelectedTeam}
          options={teams?.filter(t => t.mode === 'exclusive' || t.mode === undefined)
            .map(t => ({ value: t.name, label: t.displayName || t.name })) || []}
        />
        <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
          <QuestionCircleOutlined /> 只有独占模式的团队可以被分配节点
        </Text>
      </Modal>
    </div>
  );
};

export default ClusterNodes;

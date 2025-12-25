import React, { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Card,
  Descriptions,
  Table,
  Tag,
  Button,
  Row,
  Col,
  Space,
  Spin,
  Progress,
  Modal,
  Form,
  Input,
  Select,
  message,
  Tabs,
  Popconfirm,
  Alert,
  Empty,
} from 'antd';
import {
  ArrowLeftOutlined,
  EditOutlined,
  PlusOutlined,
  DeleteOutlined,
  LineChartOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import ReactECharts from 'echarts-for-react';
import {
  getClusterNode,
  getNodePods,
  updateNodeLabels,
  updateNodeTaints,
  getNodeMetrics,
  getSettings,
  getEnabledResourceConfigs,
  NodeTaint,
  ResourceDefinition,
} from '../../services/api';

const NodeDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [labelsModalOpen, setLabelsModalOpen] = useState(false);
  const [taintsModalOpen, setTaintsModalOpen] = useState(false);
  const [labelsForm] = Form.useForm();
  const [taintsForm] = Form.useForm();

  const { data: node, isLoading: nodeLoading } = useQuery({
    queryKey: ['node', name],
    queryFn: async () => {
      const { data } = await getClusterNode(name!);
      return data;
    },
    enabled: !!name,
  });

  const { data: pods, isLoading: podsLoading } = useQuery({
    queryKey: ['nodePods', name],
    queryFn: async () => {
      const { data } = await getNodePods(name!);
      return data.items || [];
    },
    enabled: !!name,
  });

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      const { data } = await getSettings();
      return data;
    },
  });

  const { data: resourceConfigs } = useQuery({
    queryKey: ['enabledResourceConfigs'],
    queryFn: async () => {
      const { data } = await getEnabledResourceConfigs();
      return data.items || [];
    },
    staleTime: 5 * 60 * 1000,
  });

  const { data: metrics, isLoading: metricsLoading } = useQuery({
    queryKey: ['nodeMetrics', name],
    queryFn: async () => {
      const { data } = await getNodeMetrics(name!, 24);
      return data;
    },
    enabled: !!name && !!settings?.prometheusUrl,
  });

  const updateLabelsMutation = useMutation({
    mutationFn: (labels: Record<string, string>) => updateNodeLabels(name!, labels),
    onSuccess: () => {
      message.success('标签更新成功');
      queryClient.invalidateQueries({ queryKey: ['node', name] });
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
      queryClient.invalidateQueries({ queryKey: ['nodeAssignments'] });
      setLabelsModalOpen(false);
    },
    onError: (error: Error) => {
      message.error(`更新失败: ${error.message}`);
    },
  });

  const updateTaintsMutation = useMutation({
    mutationFn: (taints: NodeTaint[]) => updateNodeTaints(name!, taints),
    onSuccess: () => {
      message.success('污点更新成功');
      queryClient.invalidateQueries({ queryKey: ['node', name] });
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
      setTaintsModalOpen(false);
    },
    onError: (error: Error) => {
      message.error(`更新失败: ${error.message}`);
    },
  });

  const handleEditLabels = () => {
    if (node) {
      const labelsText = Object.entries(node.labels || {})
        .map(([k, v]) => `${k}=${v}`)
        .join('\n');
      labelsForm.setFieldsValue({ labels: labelsText });
      setLabelsModalOpen(true);
    }
  };

  const handleSaveLabels = (values: { labels: string }) => {
    const labels: Record<string, string> = {};
    values.labels.split('\n').forEach((line) => {
      const trimmed = line.trim();
      if (trimmed) {
        const [key, ...valueParts] = trimmed.split('=');
        if (key) {
          labels[key.trim()] = valueParts.join('=').trim();
        }
      }
    });
    updateLabelsMutation.mutate(labels);
  };

  const handleEditTaints = () => {
    if (node) {
      taintsForm.setFieldsValue({
        taints: node.taints || [],
      });
      setTaintsModalOpen(true);
    }
  };

  const handleSaveTaints = (values: { taints: NodeTaint[] }) => {
    updateTaintsMutation.mutate(values.taints || []);
  };

  // Get resource config by name
  const getResourceConfig = (name: string): ResourceDefinition | undefined => {
    return resourceConfigs?.find(r => r.name === name);
  };

  // Format resource display value with unit conversion
  const formatResourceDisplay = (rawValue: number, config: ResourceDefinition): string => {
    const divisor = config.divisor || 1;
    const value = divisor > 1 ? rawValue / divisor : rawValue;
    const unit = config.unit || '';
    
    // Format number
    const formattedValue = value < 10 
      ? value.toFixed(1) 
      : Math.round(value).toString();
    
    return unit ? `${formattedValue} ${unit}` : formattedValue;
  };

  // Filter and sort resources based on config
  const categoryOrder = ['compute', 'memory', 'accelerator', 'storage', 'other'];
  const configuredResources = node?.resources?.filter(res => 
    resourceConfigs?.some(config => config.name === res.name)
  ).map(res => {
    const config = getResourceConfig(res.name);
    return {
      ...res,
      displayName: config?.displayName || res.name,
      categoryIndex: categoryOrder.indexOf(config?.category || 'other'),
      config,
    };
  }).sort((a, b) => a.categoryIndex - b.categoryIndex) || [];

  if (nodeLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!node) {
    return <div>节点不存在</div>;
  }

  const podColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      key: 'namespace',
      width: 150,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const color = status === 'Running' ? 'success' : status === 'Pending' ? 'warning' : 'default';
        return <Tag color={color}>{status}</Tag>;
      },
    },
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      width: 140,
    },
    {
      title: 'CPU 请求',
      dataIndex: 'cpuRequest',
      key: 'cpuRequest',
      width: 100,
      render: (v: number) => `${v}m`,
    },
    {
      title: '内存请求',
      dataIndex: 'memoryRequest',
      key: 'memoryRequest',
      width: 100,
      render: (v: number) => `${v}Mi`,
    },
    {
      title: '重启次数',
      dataIndex: 'restarts',
      key: 'restarts',
      width: 90,
    },
  ];

  const tabItems = [
    {
      key: 'info',
      label: '基本信息',
      children: (
        <Row gutter={[16, 16]}>
          <Col span={24}>
            <Card title="节点信息">
              <Descriptions column={2}>
                <Descriptions.Item label="节点名">{node.name}</Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={node.ready ? 'success' : 'error'}>
                    {node.ready ? '正常' : '异常'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="操作系统">{node.nodeInfo?.osImage}</Descriptions.Item>
                <Descriptions.Item label="内核版本">{node.nodeInfo?.kernelVersion}</Descriptions.Item>
                <Descriptions.Item label="容器运行时">{node.nodeInfo?.containerRuntimeVersion}</Descriptions.Item>
                <Descriptions.Item label="Kubelet 版本">{node.nodeInfo?.kubeletVersion}</Descriptions.Item>
                <Descriptions.Item label="架构">{node.nodeInfo?.architecture}</Descriptions.Item>
                <Descriptions.Item label="操作系统类型">{node.nodeInfo?.operatingSystem}</Descriptions.Item>
              </Descriptions>
            </Card>
          </Col>
          <Col span={24}>
            <Card title="网络地址">
              <Space wrap>
                {node.addresses?.map((addr, i) => (
                  <Tag key={i} color="blue">
                    {addr.type}: {addr.address}
                  </Tag>
                ))}
              </Space>
            </Card>
          </Col>
        </Row>
      ),
    },
    {
      key: 'resources',
      label: '资源使用',
      children: (
        <Card>
          {configuredResources.length === 0 ? (
            <Empty 
              description={
                <span>
                  暂无资源数据，请先在{' '}
                  <a onClick={() => navigate('/settings/resources')}>系统设置 → 资源配置</a>
                  {' '}中配置要显示的资源
                </span>
              } 
            />
          ) : (
            <Row gutter={[16, 16]}>
              {configuredResources.map((res) => {
                const used = res.capacity - res.allocatable;
                const percent = res.capacity > 0 ? Math.round((used / res.capacity) * 100) : 0;
                const displayCapacity = res.config 
                  ? formatResourceDisplay(res.capacity, res.config) 
                  : res.capacity.toString();
                const displayUsed = res.config 
                  ? formatResourceDisplay(used, res.config) 
                  : used.toString();
                return (
                  <Col xs={24} sm={12} lg={8} key={res.name}>
                    <Card size="small" title={res.displayName}>
                      <Progress
                        percent={percent}
                        format={() => `${displayUsed} / ${displayCapacity}`}
                        status={percent > 90 ? 'exception' : 'normal'}
                      />
                    </Card>
                  </Col>
                );
              })}
            </Row>
          )}
        </Card>
      ),
    },
    {
      key: 'labels',
      label: '标签',
      children: (
        <Card
          title="节点标签"
          extra={
            <Button icon={<EditOutlined />} onClick={handleEditLabels}>
              编辑标签
            </Button>
          }
        >
          <Space wrap>
            {Object.entries(node.labels || {}).map(([key, value]) => (
              <Tag key={key} color="blue">
                {key}={value}
              </Tag>
            ))}
          </Space>
          {Object.keys(node.labels || {}).length === 0 && (
            <span style={{ color: '#999' }}>暂无标签</span>
          )}
        </Card>
      ),
    },
    {
      key: 'taints',
      label: '污点',
      children: (
        <Card
          title="节点污点"
          extra={
            <Button icon={<EditOutlined />} onClick={handleEditTaints}>
              编辑污点
            </Button>
          }
        >
          <Space wrap>
            {node.taints?.map((taint, i) => (
              <Tag key={i} color="orange">
                {taint.key}={taint.value || ''}:{taint.effect}
              </Tag>
            ))}
          </Space>
          {(!node.taints || node.taints.length === 0) && (
            <span style={{ color: '#999' }}>暂无污点</span>
          )}
        </Card>
      ),
    },
    {
      key: 'pods',
      label: `Pod 列表 (${pods?.length || 0})`,
      children: (
        <Card>
          <Table
            dataSource={pods}
            columns={podColumns}
            rowKey="name"
            loading={podsLoading}
            pagination={{ pageSize: 20 }}
            size="small"
          />
        </Card>
      ),
    },
    {
      key: 'conditions',
      label: '状态条件',
      children: (
        <Card>
          <Table
            dataSource={node.conditions}
            columns={[
              { title: '类型', dataIndex: 'type', key: 'type', width: 150 },
              {
                title: '状态',
                dataIndex: 'status',
                key: 'status',
                width: 100,
                render: (s: string) => (
                  <Tag color={s === 'True' ? 'success' : s === 'False' ? 'default' : 'warning'}>
                    {s}
                  </Tag>
                ),
              },
              { title: '原因', dataIndex: 'reason', key: 'reason', width: 200 },
              { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
            ]}
            rowKey="type"
            pagination={false}
            size="small"
          />
        </Card>
      ),
    },
    {
      key: 'monitoring',
      label: (
        <span>
          <LineChartOutlined /> 监控
        </span>
      ),
      children: (
        <Card>
          {!settings?.prometheusUrl ? (
            <Alert
              message="未配置 Prometheus"
              description="请前往系统设置配置 Prometheus 地址以启用监控功能。"
              type="warning"
              showIcon
            />
          ) : metricsLoading ? (
            <div style={{ textAlign: 'center', padding: 50 }}>
              <Spin tip="加载监控数据..." />
            </div>
          ) : !metrics?.cpuUsage?.length && !metrics?.memoryUsage?.length ? (
            <Empty description="暂无监控数据" />
          ) : (
            <Row gutter={[16, 16]}>
              <Col span={24}>
                <Card size="small" title="CPU 使用率 (%)">
                  <ReactECharts
                    option={{
                      tooltip: { trigger: 'axis' },
                      xAxis: {
                        type: 'time',
                        axisLabel: {
                          formatter: (value: number) => {
                            const date = new Date(value * 1000);
                            return `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
                          },
                        },
                      },
                      yAxis: { type: 'value', min: 0, max: 100 },
                      series: [
                        {
                          data: metrics?.cpuUsage?.map((m) => [m.timestamp * 1000, m.value.toFixed(2)]) || [],
                          type: 'line',
                          smooth: true,
                          areaStyle: { opacity: 0.3 },
                        },
                      ],
                    }}
                    style={{ height: 250 }}
                  />
                </Card>
              </Col>
              <Col span={24}>
                <Card size="small" title="内存使用率 (%)">
                  <ReactECharts
                    option={{
                      tooltip: { trigger: 'axis' },
                      xAxis: {
                        type: 'time',
                        axisLabel: {
                          formatter: (value: number) => {
                            const date = new Date(value * 1000);
                            return `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
                          },
                        },
                      },
                      yAxis: { type: 'value', min: 0, max: 100 },
                      series: [
                        {
                          data: metrics?.memoryUsage?.map((m) => [m.timestamp * 1000, m.value.toFixed(2)]) || [],
                          type: 'line',
                          smooth: true,
                          areaStyle: { opacity: 0.3 },
                          itemStyle: { color: '#52c41a' },
                        },
                      ],
                    }}
                    style={{ height: 250 }}
                  />
                </Card>
              </Col>
            </Row>
          )}
        </Card>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/cluster/nodes')}>
          返回节点列表
        </Button>
      </div>

      <Tabs defaultActiveKey="info" items={tabItems} />

      {/* Labels Edit Modal */}
      <Modal
        title="编辑节点标签"
        open={labelsModalOpen}
        onCancel={() => setLabelsModalOpen(false)}
        footer={null}
        width={600}
      >
        <Form form={labelsForm} layout="vertical" onFinish={handleSaveLabels}>
          <Form.Item
            name="labels"
            label="标签"
            extra="每行一个标签，格式: key=value"
          >
            <Input.TextArea rows={10} placeholder="kubernetes.io/hostname=node1&#10;node-role.kubernetes.io/worker=" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button onClick={() => setLabelsModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit" loading={updateLabelsMutation.isPending}>
                保存
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Taints Edit Modal */}
      <Modal
        title="编辑节点污点"
        open={taintsModalOpen}
        onCancel={() => setTaintsModalOpen(false)}
        footer={null}
        width={700}
      >
        <Form form={taintsForm} layout="vertical" onFinish={handleSaveTaints}>
          <Form.List name="taints">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                    <Form.Item
                      {...restField}
                      name={[name, 'key']}
                      rules={[{ required: true, message: '请输入 Key' }]}
                    >
                      <Input placeholder="Key" style={{ width: 200 }} />
                    </Form.Item>
                    <Form.Item {...restField} name={[name, 'value']}>
                      <Input placeholder="Value (可选)" style={{ width: 150 }} />
                    </Form.Item>
                    <Form.Item
                      {...restField}
                      name={[name, 'effect']}
                      rules={[{ required: true, message: '请选择 Effect' }]}
                    >
                      <Select style={{ width: 150 }} placeholder="Effect">
                        <Select.Option value="NoSchedule">NoSchedule</Select.Option>
                        <Select.Option value="PreferNoSchedule">PreferNoSchedule</Select.Option>
                        <Select.Option value="NoExecute">NoExecute</Select.Option>
                      </Select>
                    </Form.Item>
                    <Popconfirm title="确定删除此污点？" onConfirm={() => remove(name)}>
                      <Button type="text" danger icon={<DeleteOutlined />} />
                    </Popconfirm>
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                    添加污点
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>
          <Form.Item>
            <Space>
              <Button onClick={() => setTaintsModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit" loading={updateTaintsMutation.isPending}>
                保存
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default NodeDetail;


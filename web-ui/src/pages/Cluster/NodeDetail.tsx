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
  Divider,
  Typography,
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
  PrometheusMetric,
} from '../../services/api';
import { useFeatures } from '../../hooks/useFeatures';

// Format bytes per second to human-readable string
const formatBytesPerSec = (value: number): string => {
  if (value >= 1e9) return (value / 1e9).toFixed(2) + ' GB/s';
  if (value >= 1e6) return (value / 1e6).toFixed(2) + ' MB/s';
  if (value >= 1e3) return (value / 1e3).toFixed(1) + ' KB/s';
  return value.toFixed(0) + ' B/s';
};

// Shared X-axis time formatter
const timeAxisLabel = {
  formatter: (value: number) => {
    const date = new Date(value);
    return `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
  },
};

// Build ECharts option for percentage metrics (0-100%)
const buildPercentChartOption = (data?: PrometheusMetric[], color = '#1890ff') => ({
  tooltip: { trigger: 'axis' as const },
  xAxis: { type: 'time' as const, axisLabel: timeAxisLabel },
  yAxis: { type: 'value' as const, min: 0, max: 100, axisLabel: { formatter: '{value}%' } },
  series: [{
    data: data?.map((m) => [m.timestamp * 1000, Number(m.value.toFixed(2))]) || [],
    type: 'line' as const,
    smooth: true,
    areaStyle: { opacity: 0.3 },
    itemStyle: { color },
    showSymbol: false,
  }],
});

// Build ECharts option for bandwidth metrics (auto-scale, bytes/sec)
const buildBandwidthChartOption = (data?: PrometheusMetric[], color = '#722ed1') => ({
  tooltip: {
    trigger: 'axis' as const,
    formatter: (params: { value: [number, number] }[]) => {
      if (!params?.length) return '';
      const p = params[0];
      const date = new Date(p.value[0]);
      const time = `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
      return `${time}<br/>${formatBytesPerSec(p.value[1])}`;
    },
  },
  xAxis: { type: 'time' as const, axisLabel: timeAxisLabel },
  yAxis: {
    type: 'value' as const,
    min: 0,
    axisLabel: { formatter: (v: number) => formatBytesPerSec(v) },
  },
  series: [{
    data: data?.map((m) => [m.timestamp * 1000, m.value]) || [],
    type: 'line' as const,
    smooth: true,
    areaStyle: { opacity: 0.3 },
    itemStyle: { color },
    showSymbol: false,
  }],
});

// Build ECharts option for auto-scale metrics (e.g., temperature)
const buildAutoScaleChartOption = (data?: PrometheusMetric[], color = '#fa8c16', yFormatter?: (v: number) => string) => ({
  tooltip: { trigger: 'axis' as const },
  xAxis: { type: 'time' as const, axisLabel: timeAxisLabel },
  yAxis: {
    type: 'value' as const,
    min: 0,
    axisLabel: yFormatter ? { formatter: (v: number) => yFormatter(v) } : undefined,
  },
  series: [{
    data: data?.map((m) => [m.timestamp * 1000, Number(m.value.toFixed(1))]) || [],
    type: 'line' as const,
    smooth: true,
    areaStyle: { opacity: 0.3 },
    itemStyle: { color },
    showSymbol: false,
  }],
});

// Build ECharts option for GPU with average + per-device breakdown
const buildGpuChartOption = (
  avgData?: PrometheusMetric[],
  perDevice?: { labels: Record<string, string>; metrics: PrometheusMetric[] }[],
  color = '#f5222d',
) => {
  const gpuColors = ['#f5222d', '#fa541c', '#fa8c16', '#fadb14', '#a0d911', '#52c41a', '#13c2c2', '#1890ff'];
  const series: object[] = [
    {
      name: '平均值',
      data: avgData?.map((m) => [m.timestamp * 1000, Number(m.value.toFixed(2))]) || [],
      type: 'line',
      smooth: true,
      lineStyle: { width: 2 },
      areaStyle: { opacity: 0.2 },
      itemStyle: { color },
      showSymbol: false,
    },
    ...(perDevice || []).map((s, i) => ({
      name: `GPU ${s.labels?.gpu ?? s.labels?.GPU_I_ID ?? i}`,
      data: s.metrics?.map((m) => [m.timestamp * 1000, Number(m.value.toFixed(2))]) || [],
      type: 'line',
      smooth: true,
      lineStyle: { width: 1, type: 'dashed' },
      itemStyle: { color: gpuColors[i % gpuColors.length] },
      showSymbol: false,
    })),
  ];

  return {
    tooltip: { trigger: 'axis' as const },
    legend: { show: (perDevice?.length ?? 0) > 0, bottom: 0, type: 'scroll' as const },
    xAxis: { type: 'time' as const, axisLabel: timeAxisLabel },
    yAxis: { type: 'value' as const, min: 0, max: 100, axisLabel: { formatter: '{value}%' } },
    grid: { bottom: (perDevice?.length ?? 0) > 0 ? 40 : 10 },
    series,
  };
};

const NodeDetail: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: features } = useFeatures();
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

  // Detect node accelerator type from resources
  const hasGpu = node?.resources?.some(r => r.name.includes('nvidia.com/gpu') && r.capacity > 0) ?? false;
  const hasNpu = node?.resources?.some(r => r.name.includes('huawei.com/Ascend') && r.capacity > 0) ?? false;

  const { data: metrics, isLoading: metricsLoading } = useQuery({
    queryKey: ['nodeMetrics', name, hasGpu, hasNpu],
    queryFn: async () => {
      const { data } = await getNodeMetrics(name!, { hours: 24, hasGpu, hasNpu });
      return data;
    },
    enabled: !!name && !!settings?.prometheusUrl && features?.prometheusEnabled !== false,
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
          {features?.prometheusEnabled === false || !settings?.prometheusUrl ? (
            <Alert
              message="Prometheus 未启用"
              description={features?.prometheusEnabled === false ? "Prometheus 组件未启用，请在 Helm values 中启用 Prometheus 集成。" : "请前往系统设置配置 Prometheus 地址以启用监控功能。"}
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
              {/* CPU & Memory */}
              <Col xs={24} lg={12}>
                <Card size="small" title="CPU 使用率 (%)">
                  <ReactECharts option={buildPercentChartOption(metrics?.cpuUsage, '#1890ff')} style={{ height: 220 }} />
                </Card>
              </Col>
              <Col xs={24} lg={12}>
                <Card size="small" title="内存使用率 (%)">
                  <ReactECharts option={buildPercentChartOption(metrics?.memoryUsage, '#52c41a')} style={{ height: 220 }} />
                </Card>
              </Col>

              {/* Network IO */}
              <Col span={24}><Divider plain><Typography.Text type="secondary">网络 IO</Typography.Text></Divider></Col>
              <Col xs={24} lg={12}>
                <Card size="small" title="以太网接收带宽">
                  {metrics?.networkReceive?.length ? (
                    <ReactECharts option={buildBandwidthChartOption(metrics.networkReceive, '#722ed1')} style={{ height: 220 }} />
                  ) : (
                    <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                  )}
                </Card>
              </Col>
              <Col xs={24} lg={12}>
                <Card size="small" title="以太网发送带宽">
                  {metrics?.networkTransmit?.length ? (
                    <ReactECharts option={buildBandwidthChartOption(metrics.networkTransmit, '#eb2f96')} style={{ height: 220 }} />
                  ) : (
                    <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                  )}
                </Card>
              </Col>

              {/* RDMA IO - only show if data exists */}
              {(metrics?.rdmaReceive?.length || metrics?.rdmaTransmit?.length) ? (
                <>
                  <Col span={24}><Divider plain><Typography.Text type="secondary">RDMA IO (InfiniBand)</Typography.Text></Divider></Col>
                  <Col xs={24} lg={12}>
                    <Card size="small" title="RDMA 接收带宽">
                      {metrics?.rdmaReceive?.length ? (
                        <ReactECharts option={buildBandwidthChartOption(metrics.rdmaReceive, '#13c2c2')} style={{ height: 220 }} />
                      ) : (
                        <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                      )}
                    </Card>
                  </Col>
                  <Col xs={24} lg={12}>
                    <Card size="small" title="RDMA 发送带宽">
                      {metrics?.rdmaTransmit?.length ? (
                        <ReactECharts option={buildBandwidthChartOption(metrics.rdmaTransmit, '#faad14')} style={{ height: 220 }} />
                      ) : (
                        <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                      )}
                    </Card>
                  </Col>
                </>
              ) : null}

              {/* GPU Metrics (DCGM) - only for GPU nodes */}
              {hasGpu && (metrics?.gpuUtilization?.length || metrics?.gpuMemoryUtil?.length) ? (
                <>
                  <Col span={24}><Divider plain><Typography.Text type="secondary">GPU 监控 (NVIDIA DCGM)</Typography.Text></Divider></Col>
                  <Col xs={24} lg={12}>
                    <Card size="small" title="GPU SM 利用率 (%)">
                      <ReactECharts option={buildGpuChartOption(metrics?.gpuUtilization, metrics?.gpuPerDevice, '#f5222d')} style={{ height: 250 }} />
                    </Card>
                  </Col>
                  <Col xs={24} lg={12}>
                    <Card size="small" title="GPU 显存利用率 (%)">
                      <ReactECharts option={buildPercentChartOption(metrics?.gpuMemoryUtil, '#fa541c')} style={{ height: 250 }} />
                    </Card>
                  </Col>
                </>
              ) : null}

              {/* NPU Metrics (Ascend) - only for NPU nodes */}
              {hasNpu && (metrics?.npuUtilization?.length || metrics?.npuMemoryUtil?.length) ? (
                <>
                  <Col span={24}><Divider plain><Typography.Text type="secondary">NPU 监控 (Huawei Ascend)</Typography.Text></Divider></Col>
                  <Col xs={24} lg={12}>
                    <Card size="small" title="NPU AI Core 利用率 (%)">
                      <ReactECharts option={buildPercentChartOption(metrics?.npuUtilization, '#2f54eb')} style={{ height: 220 }} />
                    </Card>
                  </Col>
                  <Col xs={24} lg={12}>
                    <Card size="small" title="NPU HBM 使用率 (%)">
                      <ReactECharts option={buildPercentChartOption(metrics?.npuMemoryUtil, '#a0d911')} style={{ height: 220 }} />
                    </Card>
                  </Col>
                  {metrics?.npuTemperature?.length ? (
                    <Col xs={24} lg={12}>
                      <Card size="small" title="NPU 温度 (\u00B0C)">
                        <ReactECharts option={buildAutoScaleChartOption(metrics.npuTemperature, '#fa8c16', (v) => `${v.toFixed(0)}\u00B0C`)} style={{ height: 220 }} />
                      </Card>
                    </Col>
                  ) : null}
                </>
              ) : null}
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


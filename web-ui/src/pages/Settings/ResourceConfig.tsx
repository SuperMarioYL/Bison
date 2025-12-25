import React, { useState } from 'react';
import {
  Card,
  Table,
  Button,
  Switch,
  Input,
  InputNumber,
  Select,
  Space,
  Tag,
  Typography,
  message,
  Modal,
  Form,
  Tooltip,
  Alert,
  Popconfirm,
  Dropdown,
} from 'antd';
import {
  ReloadOutlined,
  SaveOutlined,
  PlusOutlined,
  InfoCircleOutlined,
  DragOutlined,
  DownOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getResourceConfigs,
  discoverClusterResources,
  saveResourceConfigs,
  ResourceDefinition,
  ResourceCategory,
  DiscoveredResource,
} from '../../services/api';

const { Title, Text } = Typography;

const categoryOptions = [
  { value: 'compute', label: '计算' },
  { value: 'memory', label: '内存' },
  { value: 'storage', label: '存储' },
  { value: 'accelerator', label: '加速器' },
  { value: 'other', label: '其他' },
];

// Preset divisors for common units
const divisorPresets = [
  { label: '无换算 (1)', value: 1 },
  { label: 'Ki → 个 (1024)', value: 1024 },
  { label: 'Mi → 个 (1048576)', value: 1048576 },
  { label: 'Ki → GiB (1073741824)', value: 1073741824 },
  { label: 'bytes → GiB (1073741824)', value: 1073741824 },
  { label: 'bytes → MiB (1048576)', value: 1048576 },
];

const ResourceConfig: React.FC = () => {
  const queryClient = useQueryClient();
  const [configs, setConfigs] = useState<ResourceDefinition[]>([]);
  const [hasChanges, setHasChanges] = useState(false);
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [form] = Form.useForm();

  // Fetch resource configs
  const { data: configsData, isLoading } = useQuery({
    queryKey: ['resourceConfigs'],
    queryFn: () => getResourceConfigs().then(res => res.data.items),
  });

  // Sync fetched data to local state
  React.useEffect(() => {
    if (configsData) {
      console.log('Loaded configs from API:', configsData);
      setConfigs(configsData);
      setHasChanges(false);
    }
  }, [configsData]);

  // Fetch discovered resources
  const { data: discoveredResources, refetch: refetchDiscovered } = useQuery({
    queryKey: ['discoveredResources'],
    queryFn: () => discoverClusterResources().then(res => res.data.items),
  });

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: saveResourceConfigs,
    onSuccess: () => {
      message.success('资源配置保存成功');
      queryClient.invalidateQueries({ queryKey: ['resourceConfigs'] });
      queryClient.invalidateQueries({ queryKey: ['enabledResourceConfigs'] });
      queryClient.invalidateQueries({ queryKey: ['quotaResourceConfigs'] });
      setHasChanges(false);
    },
    onError: (error: any) => {
      const errorMsg = error?.response?.data?.error || error?.message || '未知错误';
      message.error(`保存失败: ${errorMsg}`);
      console.error('Save error:', error);
    },
  });

  // Update config locally
  const updateConfig = (name: string, field: keyof ResourceDefinition, value: unknown) => {
    setConfigs(prev => 
      prev.map(cfg => 
        cfg.name === name ? { ...cfg, [field]: value } : cfg
      )
    );
    setHasChanges(true);
  };

  // Add new resource from discovered
  const addFromDiscovered = (resource: DiscoveredResource) => {
    const existingIndex = configs.findIndex(c => c.name === resource.name);
    if (existingIndex >= 0) {
      // Enable existing config
      updateConfig(resource.name, 'enabled', true);
    } else {
      // Create new config with defaults
      const newConfig: ResourceDefinition = {
        name: resource.name,
        displayName: resource.name,
        unit: '',
        divisor: 1,
        category: 'other',
        enabled: true,
        sortOrder: configs.length + 1,
        showInQuota: true,
        price: 0,
      };
      setConfigs(prev => [...prev, newConfig]);
      setHasChanges(true);
    }
    message.success(`已添加资源: ${resource.name}`);
  };

  // Add custom resource
  const handleAddCustom = (values: Partial<ResourceDefinition>) => {
    const existingIndex = configs.findIndex(c => c.name === values.name);
    if (existingIndex >= 0) {
      message.error('资源已存在');
      return;
    }

    const newConfig: ResourceDefinition = {
      name: values.name!,
      displayName: values.displayName || values.name!,
      unit: values.unit || '',
      divisor: values.divisor || 1,
      category: values.category || 'other',
      enabled: true,
      sortOrder: configs.length + 1,
      showInQuota: true,
      price: values.price || 0,
    };
    setConfigs(prev => [...prev, newConfig]);
    setHasChanges(true);
    setAddModalVisible(false);
    form.resetFields();
    message.success('资源添加成功');
  };

  // Save all configs
  const handleSave = () => {
    // Ensure all configs have valid divisor (default to 1)
    const validConfigs = configs.map(cfg => ({
      ...cfg,
      divisor: cfg.divisor && cfg.divisor > 0 ? cfg.divisor : 1,
    }));
    console.log('Saving configs:', validConfigs);
    saveMutation.mutate(validConfigs);
  };

  // Move resource up/down
  const moveResource = (index: number, direction: 'up' | 'down') => {
    const newConfigs = [...configs];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= newConfigs.length) return;

    [newConfigs[index], newConfigs[targetIndex]] = [newConfigs[targetIndex], newConfigs[index]];
    
    // Update sortOrder
    newConfigs.forEach((cfg, i) => {
      cfg.sortOrder = i + 1;
    });
    
    setConfigs(newConfigs);
    setHasChanges(true);
  };

  // Delete resource config
  const deleteConfig = (name: string) => {
    setConfigs(prev => prev.filter(c => c.name !== name));
    setHasChanges(true);
  };

  const columns = [
    {
      title: '排序',
      key: 'sort',
      width: 80,
      render: (_: unknown, __: ResourceDefinition, index: number) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<DragOutlined rotate={90} />}
            disabled={index === 0}
            onClick={() => moveResource(index, 'up')}
          />
          <Button
            type="text"
            size="small"
            icon={<DragOutlined rotate={-90} />}
            disabled={index === configs.length - 1}
            onClick={() => moveResource(index, 'down')}
          />
        </Space>
      ),
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 70,
      render: (enabled: boolean, record: ResourceDefinition) => (
        <Switch
          checked={enabled}
          onChange={(checked) => updateConfig(record.name, 'enabled', checked)}
        />
      ),
    },
    {
      title: '资源名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (name: string) => <Text code style={{ fontSize: 12 }}>{name}</Text>,
    },
    {
      title: '显示名称',
      dataIndex: 'displayName',
      key: 'displayName',
      width: 120,
      render: (displayName: string, record: ResourceDefinition) => (
        <Input
          value={displayName}
          onChange={(e) => updateConfig(record.name, 'displayName', e.target.value)}
          style={{ width: 110 }}
        />
      ),
    },
    {
      title: '单位',
      dataIndex: 'unit',
      key: 'unit',
      width: 80,
      render: (unit: string, record: ResourceDefinition) => (
        <Input
          value={unit}
          onChange={(e) => updateConfig(record.name, 'unit', e.target.value)}
          placeholder="单位"
          style={{ width: 70 }}
        />
      ),
    },
    {
      title: (
        <Space>
          换算除数
          <Tooltip title="显示值 = 原始值 / 除数。例如 memory 原始值是字节，除以 1073741824 得到 GiB">
            <InfoCircleOutlined />
          </Tooltip>
        </Space>
      ),
      dataIndex: 'divisor',
      key: 'divisor',
      width: 180,
      render: (divisor: number, record: ResourceDefinition) => (
        <Space.Compact style={{ width: '100%' }}>
          <InputNumber
            value={divisor || 1}
            onChange={(value) => updateConfig(record.name, 'divisor', value || 1)}
            min={1}
            style={{ width: 100 }}
          />
          <Dropdown
            menu={{
              items: divisorPresets.map(preset => ({
                key: String(preset.value),
                label: preset.label,
                onClick: () => updateConfig(record.name, 'divisor', preset.value),
              })),
            }}
            trigger={['click']}
          >
            <Button icon={<DownOutlined />} />
          </Dropdown>
        </Space.Compact>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 100,
      render: (category: ResourceCategory, record: ResourceDefinition) => (
        <Select
          value={category}
          onChange={(value) => updateConfig(record.name, 'category', value)}
          options={categoryOptions}
          style={{ width: 90 }}
        />
      ),
    },
    {
      title: '配额',
      dataIndex: 'showInQuota',
      key: 'showInQuota',
      width: 60,
      render: (showInQuota: boolean, record: ResourceDefinition) => (
        <Switch
          checked={showInQuota}
          size="small"
          onChange={(checked) => updateConfig(record.name, 'showInQuota', checked)}
        />
      ),
    },
    {
      title: '单价 (¥/单位/时)',
      dataIndex: 'price',
      key: 'price',
      width: 130,
      render: (price: number, record: ResourceDefinition) => (
        <InputNumber
          value={price}
          onChange={(value) => updateConfig(record.name, 'price', value || 0)}
          min={0}
          step={0.01}
          precision={2}
          style={{ width: 120 }}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 70,
      render: (_: unknown, record: ResourceDefinition) => (
        <Popconfirm
          title="确定删除此资源配置？"
          onConfirm={() => deleteConfig(record.name)}
          okText="确定"
          cancelText="取消"
        >
          <Button type="link" danger size="small">
            删除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  // Unconfigured resources from cluster
  const unconfiguredResources = discoveredResources?.filter(
    dr => !configs.some(c => c.name === dr.name)
  ) || [];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>资源配置</Title>
        <Space>
          <Button
            icon={<PlusOutlined />}
            onClick={() => setAddModalVisible(true)}
          >
            添加自定义资源
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => refetchDiscovered()}
          >
            刷新集群资源
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            onClick={handleSave}
            disabled={!hasChanges}
            loading={saveMutation.isPending}
          >
            保存配置
          </Button>
        </Space>
      </div>

      <Alert
        type="info"
        message="资源配置说明"
        description={
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            <li>从集群中发现的资源需要手动配置后才会在资源总览中显示</li>
            <li>换算除数用于将 K8s 原始值转换为显示值，例如：memory 原始值是字节，设置除数为 1073741824 (1 GiB) 后显示为 GiB</li>
            <li>单价用于计费，按 (显示值 × 单价 × 使用时间) 计算</li>
          </ul>
        }
        showIcon
        style={{ marginBottom: 16 }}
      />

      {hasChanges && (
        <Alert
          type="warning"
          message="有未保存的更改"
          style={{ marginBottom: 16 }}
          showIcon
        />
      )}

      <Card className="glass-card" style={{ marginBottom: 16 }}>
        {configs.length === 0 && !isLoading ? (
          <Alert
            type="warning"
            message="尚未配置任何资源"
            description="请从下方的「集群中发现的资源」中点击添加，或使用「添加自定义资源」按钮手动添加"
            showIcon
          />
        ) : (
          <Table
            dataSource={configs}
            columns={columns}
            rowKey="name"
            loading={isLoading}
            pagination={false}
            size="small"
            scroll={{ x: 1200 }}
          />
        )}
      </Card>

      {unconfiguredResources.length > 0 && (
        <Card 
          className="glass-card" 
          title={
            <Space>
              <span>集群中发现的资源</span>
              <Tag color="blue">{unconfiguredResources.length}</Tag>
            </Space>
          }
        >
          <Space wrap>
            {unconfiguredResources.map(resource => (
              <Tag
                key={resource.name}
                style={{ cursor: 'pointer', padding: '4px 8px' }}
                onClick={() => addFromDiscovered(resource)}
              >
                <PlusOutlined /> {resource.name}
                <Text type="secondary" style={{ marginLeft: 8 }}>
                  ({resource.capacity.toFixed(0)})
                </Text>
              </Tag>
            ))}
          </Space>
          <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
            点击资源可添加到配置中（添加后需要配置显示名称、单位、换算除数等）
          </Text>
        </Card>
      )}

      {/* Add Custom Resource Modal */}
      <Modal
        title="添加自定义资源"
        open={addModalVisible}
        onCancel={() => {
          setAddModalVisible(false);
          form.resetFields();
        }}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleAddCustom}
          initialValues={{ divisor: 1, category: 'other' }}
        >
          <Form.Item
            name="name"
            label="资源名称"
            rules={[{ required: true, message: '请输入资源名称' }]}
            extra="K8s 资源名，例如: custom.io/resource"
          >
            <Input placeholder="资源名称" />
          </Form.Item>
          <Form.Item
            name="displayName"
            label="显示名称"
          >
            <Input placeholder="显示名称" />
          </Form.Item>
          <Form.Item
            name="unit"
            label="单位"
          >
            <Input placeholder="例如: 核, GiB, 卡" />
          </Form.Item>
          <Form.Item
            name="divisor"
            label="换算除数"
            extra="显示值 = 原始值 / 除数"
          >
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="category"
            label="分类"
          >
            <Select options={categoryOptions} placeholder="选择分类" />
          </Form.Item>
          <Form.Item
            name="price"
            label="单价 (元/单位/小时)"
          >
            <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                添加
              </Button>
              <Button onClick={() => {
                setAddModalVisible(false);
                form.resetFields();
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ResourceConfig;

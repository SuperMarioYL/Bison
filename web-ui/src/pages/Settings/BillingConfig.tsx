import React, { useEffect } from 'react';
import { Card, Form, Input, InputNumber, Switch, Button, Table, message, Typography, Alert, Tag, Select, Space, Divider } from 'antd';
import { DollarOutlined, SaveOutlined, SettingOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getBillingConfig, updateBillingConfig, getEnabledResourceConfigs, ResourceDefinition } from '../../services/api';

const { Text } = Typography;

const BillingConfig: React.FC = () => {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data: configData, isLoading } = useQuery({
    queryKey: ['billingConfig'],
    queryFn: getBillingConfig,
  });

  const { data: resourceConfigs } = useQuery({
    queryKey: ['enabledResourceConfigs'],
    queryFn: () => getEnabledResourceConfigs().then(res => res.data.items),
  });

  const updateMutation = useMutation({
    mutationFn: updateBillingConfig,
    onSuccess: () => {
      message.success('配置已保存');
      queryClient.invalidateQueries({ queryKey: ['billingConfig'] });
    },
    onError: () => {
      message.error('保存失败');
    },
  });

  useEffect(() => {
    if (configData?.data) {
      const config = configData.data;
      form.setFieldsValue({
        enabled: config.enabled,
        interval: config.interval,
        currency: config.currency,
        currencySymbol: config.currencySymbol,
        gracePeriodValue: config.gracePeriodValue || 3,
        gracePeriodUnit: config.gracePeriodUnit || 'days',
      });
    }
  }, [configData, form]);

  const handleSave = async () => {
    const values = await form.validateFields();

    updateMutation.mutate({
      ...values,
      pricing: configData?.data?.pricing || {},
    });
  };

  // Resource pricing columns (read-only, from resource config)
  const pricingColumns = [
    {
      title: '资源',
      dataIndex: 'displayName',
      key: 'displayName',
      render: (displayName: string, record: ResourceDefinition) => (
        <span>
          {displayName}
          <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
            ({record.name})
          </Text>
        </span>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      render: (category: string) => {
        const categoryLabels: Record<string, { color: string; label: string }> = {
          compute: { color: 'blue', label: '计算' },
          memory: { color: 'green', label: '内存' },
          storage: { color: 'orange', label: '存储' },
          accelerator: { color: 'purple', label: '加速器' },
          other: { color: 'default', label: '其他' },
        };
        const cfg = categoryLabels[category] || categoryLabels.other;
        return <Tag color={cfg.color}>{cfg.label}</Tag>;
      },
    },
    {
      title: '单价',
      key: 'price',
      render: (_: unknown, record: ResourceDefinition) => (
        <Text>
          ¥{record.price.toFixed(2)} / {record.unit} / 小时
        </Text>
      ),
    },
  ];

  return (
    <div>
      <Card title={<><DollarOutlined /> 计费配置</>} loading={isLoading}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            enabled: true,
            interval: 1,
            currency: 'CNY',
            currencySymbol: '¥',
            gracePeriodValue: 3,
            gracePeriodUnit: 'days',
          }}
        >
          <Form.Item
            name="enabled"
            label="启用计费"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name="interval"
            label="计费周期（小时）"
            rules={[{ required: true, message: '请输入计费周期' }]}
          >
            <InputNumber min={1} max={24} />
          </Form.Item>

          <Form.Item
            name="currency"
            label="货币代码"
            rules={[{ required: true, message: '请输入货币代码' }]}
          >
            <Input style={{ width: 120 }} />
          </Form.Item>

          <Form.Item
            name="currencySymbol"
            label="货币符号"
            rules={[{ required: true, message: '请输入货币符号' }]}
          >
            <Input style={{ width: 80 }} />
          </Form.Item>

          <Divider>
            <Space>
              <ClockCircleOutlined />
              欠费宽限期
            </Space>
          </Divider>
          
          <Alert
            type="info"
            message="欠费宽限期说明"
            description="当团队余额变为负数时，不会立即暂停其工作负载。系统会等待宽限期结束后才执行暂停操作，给予团队充值的缓冲时间。"
            showIcon
            style={{ marginBottom: 16 }}
          />

          <Form.Item label="宽限期时长">
            <Space>
              <Form.Item
                name="gracePeriodValue"
                noStyle
                rules={[{ required: true, message: '请输入宽限期时长' }]}
              >
                <InputNumber min={1} max={30} style={{ width: 100 }} />
              </Form.Item>
              <Form.Item
                name="gracePeriodUnit"
                noStyle
                rules={[{ required: true, message: '请选择单位' }]}
              >
                <Select style={{ width: 100 }}>
                  <Select.Option value="hours">小时</Select.Option>
                  <Select.Option value="days">天</Select.Option>
                </Select>
              </Form.Item>
            </Space>
          </Form.Item>
          
          <Text type="secondary">
            示例：设置为 3 天，则团队欠费后 3 天内仍可正常使用资源，3 天后系统将暂停该团队的所有工作负载。
          </Text>
        </Form>

        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Button 
            type="primary" 
            icon={<SaveOutlined />} 
            onClick={handleSave}
            loading={updateMutation.isPending}
          >
            保存配置
          </Button>
        </div>
      </Card>

      <Card 
        title="资源单价" 
        style={{ marginTop: 16 }}
        extra={
          <Link to="/settings/resources">
            <Button type="link" icon={<SettingOutlined />}>
              管理资源配置
            </Button>
          </Link>
        }
      >
        <Alert
          type="info"
          message="资源单价在「资源配置」页面中统一管理"
          description="每个资源的单价配置在资源配置页面中维护，这里仅展示当前启用的资源及其单价。"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Table
          columns={pricingColumns}
          dataSource={resourceConfigs || []}
          rowKey="name"
          pagination={false}
          locale={{ emptyText: '暂无启用的资源' }}
        />
      </Card>
    </div>
  );
};

export default BillingConfig;


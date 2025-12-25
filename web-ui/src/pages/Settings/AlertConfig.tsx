import React, { useEffect, useState } from 'react';
import { Card, Form, InputNumber, Button, Table, Space, Modal, Input, Select, Switch, message, Popconfirm, Tag } from 'antd';
import { BellOutlined, PlusOutlined, DeleteOutlined, SaveOutlined, SendOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getAlertConfig, updateAlertConfig, testAlertChannel, getAlertHistory, NotifyChannel } from '../../services/api';
import dayjs from 'dayjs';

const AlertConfig: React.FC = () => {
  const [form] = Form.useForm();
  const [channelModal, setChannelModal] = useState(false);
  const [channelForm] = Form.useForm();
  const [channels, setChannels] = useState<NotifyChannel[]>([]);
  const queryClient = useQueryClient();

  const { data: configData, isLoading } = useQuery({
    queryKey: ['alertConfig'],
    queryFn: getAlertConfig,
  });

  const { data: historyData, isLoading: loadingHistory } = useQuery({
    queryKey: ['alertHistory'],
    queryFn: () => getAlertHistory(20),
  });

  const updateMutation = useMutation({
    mutationFn: updateAlertConfig,
    onSuccess: () => {
      message.success('配置已保存');
      queryClient.invalidateQueries({ queryKey: ['alertConfig'] });
    },
    onError: () => {
      message.error('保存失败');
    },
  });

  const testMutation = useMutation({
    mutationFn: testAlertChannel,
    onSuccess: () => {
      message.success('测试通知已发送');
    },
    onError: () => {
      message.error('测试发送失败');
    },
  });

  useEffect(() => {
    if (configData?.data) {
      form.setFieldsValue({
        balanceThreshold: configData.data.balanceThreshold,
      });
      setChannels(configData.data.channels || []);
    }
  }, [configData, form]);

  const history = historyData?.data?.items || [];

  const handleSave = async () => {
    const values = await form.validateFields();
    updateMutation.mutate({
      balanceThreshold: values.balanceThreshold,
      channels,
    });
  };

  const handleAddChannel = async () => {
    const values = await channelForm.validateFields();
    const newChannel: NotifyChannel = {
      id: Date.now().toString(),
      type: values.type,
      name: values.name,
      config: {},
      enabled: true,
    };

    if (values.type === 'webhook') {
      newChannel.config.url = values.url;
    } else if (values.type === 'dingtalk' || values.type === 'wechat') {
      newChannel.config.webhook = values.webhook;
    } else if (values.type === 'email') {
      newChannel.config.to = values.to;
      newChannel.config.smtp = values.smtp;
    }

    setChannels([...channels, newChannel]);
    setChannelModal(false);
    channelForm.resetFields();
  };

  const handleDeleteChannel = (id: string) => {
    setChannels(channels.filter(c => c.id !== id));
  };

  const handleToggleChannel = (id: string, enabled: boolean) => {
    setChannels(channels.map(c => c.id === id ? { ...c, enabled } : c));
  };

  const channelColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => {
        const typeLabels: Record<string, string> = {
          webhook: 'Webhook',
          dingtalk: '钉钉',
          wechat: '企业微信',
          email: '邮件',
        };
        return typeLabels[type] || type;
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: NotifyChannel) => (
        <Switch checked={enabled} onChange={(v) => handleToggleChannel(record.id, v)} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: NotifyChannel) => (
        <Space>
          <Button 
            type="link" 
            icon={<SendOutlined />} 
            onClick={() => testMutation.mutate(record)}
            loading={testMutation.isPending}
          >
            测试
          </Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDeleteChannel(record.id)}>
            <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const historyColumns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (ts: string) => dayjs(ts).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
    },
    {
      title: '级别',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => {
        const colors: Record<string, string> = {
          info: 'blue',
          warning: 'orange',
          critical: 'red',
        };
        return <Tag color={colors[severity]}>{severity}</Tag>;
      },
    },
    {
      title: '目标',
      dataIndex: 'target',
      key: 'target',
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '已发送',
      dataIndex: 'sent',
      key: 'sent',
      render: (sent: boolean) => sent ? <Tag color="success">是</Tag> : <Tag>否</Tag>,
    },
  ];

  return (
    <div>
      <Card title={<><BellOutlined /> 告警配置</>} loading={isLoading}>
        <Form form={form} layout="vertical">
          <Form.Item
            name="balanceThreshold"
            label="余额预警阈值"
            rules={[{ required: true, message: '请输入预警阈值' }]}
          >
            <InputNumber 
              min={0} 
              step={100}
              prefix="¥"
              style={{ width: 200 }}
            />
          </Form.Item>
        </Form>
      </Card>

      <Card 
        title="通知渠道" 
        style={{ marginTop: 16 }}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setChannelModal(true)}>
            添加渠道
          </Button>
        }
      >
        <Table
          columns={channelColumns}
          dataSource={channels}
          rowKey="id"
          pagination={false}
        />
      </Card>

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

      <Card title="告警历史" style={{ marginTop: 16 }}>
        <Table
          columns={historyColumns}
          dataSource={history}
          loading={loadingHistory}
          rowKey="id"
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title="添加通知渠道"
        open={channelModal}
        onOk={handleAddChannel}
        onCancel={() => setChannelModal(false)}
      >
        <Form form={channelForm} layout="vertical">
          <Form.Item
            name="name"
            label="渠道名称"
            rules={[{ required: true, message: '请输入渠道名称' }]}
          >
            <Input />
          </Form.Item>

          <Form.Item
            name="type"
            label="渠道类型"
            rules={[{ required: true, message: '请选择渠道类型' }]}
          >
            <Select>
              <Select.Option value="webhook">Webhook</Select.Option>
              <Select.Option value="dingtalk">钉钉</Select.Option>
              <Select.Option value="wechat">企业微信</Select.Option>
              <Select.Option value="email">邮件</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.type !== currentValues.type}
          >
            {({ getFieldValue }) => {
              const type = getFieldValue('type');
              if (type === 'webhook') {
                return (
                  <Form.Item name="url" label="Webhook URL" rules={[{ required: true }]}>
                    <Input placeholder="https://..." />
                  </Form.Item>
                );
              }
              if (type === 'dingtalk' || type === 'wechat') {
                return (
                  <Form.Item name="webhook" label="机器人 Webhook" rules={[{ required: true }]}>
                    <Input placeholder="https://..." />
                  </Form.Item>
                );
              }
              if (type === 'email') {
                return (
                  <>
                    <Form.Item name="to" label="收件人" rules={[{ required: true }]}>
                      <Input placeholder="email@example.com" />
                    </Form.Item>
                    <Form.Item name="smtp" label="SMTP 服务器">
                      <Input placeholder="smtp.example.com:587" />
                    </Form.Item>
                  </>
                );
              }
              return null;
            }}
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AlertConfig;


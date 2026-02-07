import React, { useEffect, useState } from 'react';
import {
  Card,
  Form,
  Input,
  InputNumber,
  Select,
  Button,
  Space,
  message,
  Alert,
  Typography,
  Tag,
  Spin,
} from 'antd';
import {
  CloudServerOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getControlPlaneConfig,
  updateControlPlaneConfig,
  testControlPlaneConnection,
} from '../../services/api';

const { TextArea } = Input;
const { Paragraph, Text } = Typography;

const ControlPlaneConfig: React.FC = () => {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const [authMethod, setAuthMethod] = useState<'password' | 'privateKey'>('password');
  const [testResult, setTestResult] = useState<'success' | 'error' | null>(null);

  const { data: config, isLoading } = useQuery({
    queryKey: ['controlPlaneConfig'],
    queryFn: () => getControlPlaneConfig().then(res => res.data),
  });

  useEffect(() => {
    if (config) {
      form.setFieldsValue({
        host: config.host,
        sshPort: config.sshPort || 22,
        sshUser: config.sshUser || 'root',
        authMethod: config.authMethod || 'password',
      });
      setAuthMethod(config.authMethod || 'password');
    }
  }, [config, form]);

  const updateMutation = useMutation({
    mutationFn: updateControlPlaneConfig,
    onSuccess: () => {
      message.success('控制面配置已保存');
      queryClient.invalidateQueries({ queryKey: ['controlPlaneConfig'] });
      setTestResult(null);
    },
    onError: (err: Error) => {
      message.error(`保存失败: ${err.message}`);
    },
  });

  const testMutation = useMutation({
    mutationFn: testControlPlaneConnection,
    onSuccess: () => {
      setTestResult('success');
      message.success('连接测试成功');
    },
    onError: (err: Error) => {
      setTestResult('error');
      message.error(`连接测试失败: ${err.message}`);
    },
  });

  const handleSave = () => {
    form.validateFields().then(values => {
      updateMutation.mutate(values);
    });
  };

  const handleTest = () => {
    // First save, then test
    form.validateFields().then(values => {
      updateMutation.mutate(values, {
        onSuccess: () => {
          testMutation.mutate();
        },
      });
    });
  };

  if (isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <Alert
        message="控制面 SSH 配置"
        description={
          <Paragraph style={{ marginBottom: 0 }}>
            配置 Kubernetes 控制面节点的 SSH 访问信息，用于在添加节点时执行
            <Text code>kubeadm token create</Text> 命令生成加入令牌。
            请确保控制面节点上已安装 kubeadm。
          </Paragraph>
        }
        type="info"
        showIcon
        style={{ marginBottom: 24 }}
      />

      <Card className="glass-card">
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            sshPort: 22,
            sshUser: 'root',
            authMethod: 'password',
          }}
        >
          <Form.Item
            name="host"
            label={
              <Space>
                <CloudServerOutlined />
                控制面节点 IP/主机名
              </Space>
            }
            rules={[{ required: true, message: '请输入控制面节点地址' }]}
          >
            <Input placeholder="如：192.168.1.100 或 k8s-master" />
          </Form.Item>

          <Space size="large" style={{ width: '100%' }}>
            <Form.Item
              name="sshPort"
              label="SSH 端口"
              rules={[{ required: true }]}
              style={{ width: 150 }}
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item
              name="sshUser"
              label="SSH 用户名"
              rules={[{ required: true }]}
              style={{ width: 200 }}
            >
              <Input placeholder="root" />
            </Form.Item>
          </Space>

          <Form.Item
            name="authMethod"
            label={
              <Space>
                <SafetyCertificateOutlined />
                认证方式
              </Space>
            }
            rules={[{ required: true }]}
          >
            <Select onChange={(value) => setAuthMethod(value)}>
              <Select.Option value="password">密码</Select.Option>
              <Select.Option value="privateKey">私钥</Select.Option>
            </Select>
          </Form.Item>

          {authMethod === 'password' ? (
            <Form.Item
              name="password"
              label="密码"
              extra={config?.hasPassword ? <Tag color="green">已配置密码</Tag> : null}
            >
              <Input.Password placeholder="留空则保留原密码" />
            </Form.Item>
          ) : (
            <Form.Item
              name="privateKey"
              label="私钥内容"
              extra={config?.hasPrivateKey ? <Tag color="green">已配置私钥</Tag> : null}
            >
              <TextArea
                rows={8}
                placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                style={{ fontFamily: 'monospace', fontSize: 12 }}
              />
            </Form.Item>
          )}

          {testResult && (
            <Alert
              message={testResult === 'success' ? '连接测试成功' : '连接测试失败'}
              type={testResult === 'success' ? 'success' : 'error'}
              showIcon
              icon={testResult === 'success' ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
              style={{ marginBottom: 16 }}
            />
          )}

          <Form.Item>
            <Space>
              <Button
                type="primary"
                onClick={handleSave}
                loading={updateMutation.isPending}
              >
                保存配置
              </Button>
              <Button
                onClick={handleTest}
                loading={testMutation.isPending || updateMutation.isPending}
              >
                测试连接
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default ControlPlaneConfig;

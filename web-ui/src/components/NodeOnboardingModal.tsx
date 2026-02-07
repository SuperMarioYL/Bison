import React, { useState } from 'react';
import {
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Alert,
  Typography,
} from 'antd';
import { useMutation } from '@tanstack/react-query';
import { startNodeOnboarding, OnboardingRequest } from '../services/api';

const { TextArea } = Input;
const { Text } = Typography;

interface NodeOnboardingModalProps {
  open: boolean;
  onClose: () => void;
  onStarted: (jobId: string) => void;
}

const NodeOnboardingModal: React.FC<NodeOnboardingModalProps> = ({
  open,
  onClose,
  onStarted,
}) => {
  const [form] = Form.useForm();
  const [authMethod, setAuthMethod] = useState<'password' | 'privateKey'>('password');

  const startMutation = useMutation({
    mutationFn: startNodeOnboarding,
    onSuccess: (response) => {
      form.resetFields();
      onStarted(response.data.id);
    },
  });

  const handleSubmit = () => {
    form.validateFields().then(values => {
      const request: OnboardingRequest = {
        nodeIP: values.nodeIP,
        sshPort: values.sshPort || 22,
        sshUsername: values.sshUsername,
        authMethod: values.authMethod,
        password: values.authMethod === 'password' ? values.password : undefined,
        privateKey: values.authMethod === 'privateKey' ? values.privateKey : undefined,
      };
      startMutation.mutate(request);
    });
  };

  const handleClose = () => {
    form.resetFields();
    setAuthMethod('password');
    onClose();
  };

  // IP address validation
  const validateIP = (_: unknown, value: string) => {
    if (!value) {
      return Promise.reject(new Error('请输入节点 IP'));
    }
    // Simple IP format validation
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
    if (!ipRegex.test(value)) {
      return Promise.reject(new Error('请输入有效的 IP 地址'));
    }
    const parts = value.split('.').map(Number);
    if (parts.some(p => p > 255)) {
      return Promise.reject(new Error('请输入有效的 IP 地址'));
    }
    return Promise.resolve();
  };

  return (
    <Modal
      title="添加裸金属节点"
      open={open}
      onOk={handleSubmit}
      onCancel={handleClose}
      okText="开始添加"
      cancelText="取消"
      confirmLoading={startMutation.isPending}
      width={500}
      destroyOnClose
    >
      <Alert
        message="前置条件"
        description={
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            <li>目标节点已安装操作系统（Ubuntu/CentOS 等）</li>
            <li>目标节点已安装 kubeadm、kubelet、kubectl</li>
            <li>目标节点网络可达，支持 SSH 连接</li>
          </ul>
        }
        type="info"
        showIcon
        style={{ marginBottom: 24 }}
      />

      {startMutation.isError && (
        <Alert
          message="启动失败"
          description={(startMutation.error as Error).message}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          sshPort: 22,
          sshUsername: 'root',
          authMethod: 'password',
        }}
      >
        <Form.Item
          name="nodeIP"
          label="节点 IP"
          rules={[{ validator: validateIP }]}
        >
          <Input placeholder="如：192.168.1.100" />
        </Form.Item>

        <Form.Item
          name="sshPort"
          label="SSH 端口"
          rules={[{ required: true, message: '请输入 SSH 端口' }]}
        >
          <InputNumber min={1} max={65535} style={{ width: 150 }} />
        </Form.Item>

        <Form.Item
          name="sshUsername"
          label="SSH 用户名"
          rules={[{ required: true, message: '请输入 SSH 用户名' }]}
        >
          <Input placeholder="root" />
        </Form.Item>

        <Form.Item
          name="authMethod"
          label="认证方式"
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
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="SSH 登录密码" />
          </Form.Item>
        ) : (
          <Form.Item
            name="privateKey"
            label="私钥内容"
            rules={[{ required: true, message: '请输入私钥' }]}
            extra={<Text type="secondary">将私钥内容粘贴到此处</Text>}
          >
            <TextArea
              rows={6}
              placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

export default NodeOnboardingModal;

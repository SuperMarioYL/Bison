import React, { useState } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Space,
  Switch,
  message,
  Alert,
  Typography,
  Spin,
  Tag,
  Table,
  Modal,
  Popconfirm,
  Tooltip,
  Collapse,
  Empty,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CodeOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getInitScripts,
  createInitScript,
  updateInitScript,
  deleteInitScript,
  toggleInitScript,
  ScriptGroup,
  Script,
  ScriptPhase,
} from '../../services/api';

const { TextArea } = Input;
const { Text, Paragraph } = Typography;
const { Panel } = Collapse;

const phaseLabels: Record<ScriptPhase, string> = {
  'pre-join': 'Pre-join',
  'post-join': 'Post-join',
};

const phaseColors: Record<ScriptPhase, string> = {
  'pre-join': 'blue',
  'post-join': 'green',
};

// Platform script editor component
interface PlatformScriptEditorProps {
  scripts: Script[];
  onChange: (scripts: Script[]) => void;
}

const PlatformScriptEditor: React.FC<PlatformScriptEditorProps> = ({
  scripts,
  onChange,
}) => {
  const [editingScript, setEditingScript] = useState<Script | null>(null);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [form] = Form.useForm();

  const osList = [
    { value: '*', label: '通用 (所有系统)' },
    { value: 'ubuntu', label: 'Ubuntu' },
    { value: 'centos', label: 'CentOS' },
    { value: 'debian', label: 'Debian' },
    { value: 'rhel', label: 'RHEL' },
    { value: 'openEuler', label: 'openEuler' },
    { value: 'rocky', label: 'Rocky Linux' },
    { value: 'almalinux', label: 'AlmaLinux' },
    { value: 'kylin', label: 'Kylin' },
    { value: 'uos', label: 'UOS (统信)' },
  ];
  const archList = [
    { value: '*', label: '通用 (所有架构)' },
    { value: 'amd64', label: 'amd64 (x86_64)' },
    { value: 'arm64', label: 'arm64 (aarch64)' },
  ];

  const handleAdd = () => {
    setEditingScript(null);
    form.resetFields();
    form.setFieldsValue({ os: '*', arch: '*', content: '#!/bin/bash\nset -e\n\n' });
    setEditModalVisible(true);
  };

  const handleEdit = (script: Script) => {
    setEditingScript(script);
    form.setFieldsValue(script);
    setEditModalVisible(true);
  };

  const handleDelete = (scriptId: string) => {
    onChange(scripts.filter(s => s.id !== scriptId));
  };

  const handleSave = () => {
    form.validateFields().then(values => {
      if (editingScript) {
        onChange(scripts.map(s => s.id === editingScript.id ? { ...s, ...values } : s));
      } else {
        const newScript: Script = {
          id: `script-${Date.now()}`,
          ...values,
        };
        onChange([...scripts, newScript]);
      }
      setEditModalVisible(false);
    });
  };

  const getPlatformLabel = (os: string, arch: string) => {
    const osLabel = os === '*' ? '通用' : os;
    const archLabel = arch === '*' ? '所有架构' : arch;
    return `${osLabel} × ${archLabel}`;
  };

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button type="dashed" icon={<PlusOutlined />} onClick={handleAdd}>
          添加平台脚本
        </Button>
      </div>

      {scripts.length === 0 ? (
        <Empty description="暂无平台脚本" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Collapse>
          {scripts.map(script => (
            <Panel
              key={script.id}
              header={
                <Space>
                  <Tag color={script.os === '*' && script.arch === '*' ? 'default' : 'blue'}>
                    {getPlatformLabel(script.os, script.arch)}
                  </Tag>
                </Space>
              }
              extra={
                <Space onClick={e => e.stopPropagation()}>
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => handleEdit(script)}
                  />
                  <Popconfirm
                    title="确定删除此平台脚本？"
                    onConfirm={() => handleDelete(script.id)}
                  >
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              }
            >
              <pre style={{
                background: '#f5f5f5',
                padding: 12,
                borderRadius: 4,
                maxHeight: 200,
                overflow: 'auto',
                fontSize: 12,
              }}>
                {script.content}
              </pre>
            </Panel>
          ))}
        </Collapse>
      )}

      <Modal
        title={editingScript ? '编辑平台脚本' : '添加平台脚本'}
        open={editModalVisible}
        onOk={handleSave}
        onCancel={() => setEditModalVisible(false)}
        width={700}
      >
        <Form form={form} layout="vertical">
          <Space style={{ width: '100%' }} size="large">
            <Form.Item name="os" label="操作系统" rules={[{ required: true }]} style={{ width: 220 }}>
              <Select
                showSearch
                optionFilterProp="label"
                options={osList}
                placeholder="选择或输入操作系统"
              />
            </Form.Item>
            <Form.Item name="arch" label="CPU 架构" rules={[{ required: true }]} style={{ width: 200 }}>
              <Select options={archList} />
            </Form.Item>
          </Space>
          <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
            操作系统名称来自节点的 /etc/os-release 文件中的 ID 字段。如果列表中没有你需要的系统，可以先添加一个节点，查看检测到的系统名称后再配置脚本。
          </Text>
          <Form.Item
            name="content"
            label="脚本内容"
            rules={[{ required: true, message: '请输入脚本内容' }]}
          >
            <TextArea
              rows={15}
              style={{ fontFamily: 'monospace', fontSize: 12 }}
              placeholder="#!/bin/bash&#10;set -e&#10;&#10;# Your script here"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// ==================== Main Component ====================

const NodeInitConfig: React.FC = () => {
  const queryClient = useQueryClient();

  const [scriptModalVisible, setScriptModalVisible] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ScriptGroup | null>(null);
  const [scriptForm] = Form.useForm();
  const [scripts, setScripts] = useState<Script[]>([]);

  const { data: scriptGroups, isLoading } = useQuery({
    queryKey: ['initScripts'],
    queryFn: () => getInitScripts().then(res => res.data.items),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleInitScript(id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['initScripts'] });
    },
    onError: (err: Error) => {
      message.error(`操作失败: ${err.message}`);
    },
  });

  const createMutation = useMutation({
    mutationFn: createInitScript,
    onSuccess: () => {
      message.success('脚本分组创建成功');
      queryClient.invalidateQueries({ queryKey: ['initScripts'] });
      setScriptModalVisible(false);
    },
    onError: (err: Error) => {
      message.error(`创建失败: ${err.message}`);
    },
  });

  const updateScriptMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ScriptGroup> }) =>
      updateInitScript(id, data),
    onSuccess: () => {
      message.success('脚本分组更新成功');
      queryClient.invalidateQueries({ queryKey: ['initScripts'] });
      setScriptModalVisible(false);
    },
    onError: (err: Error) => {
      message.error(`更新失败: ${err.message}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteInitScript,
    onSuccess: () => {
      message.success('脚本分组已删除');
      queryClient.invalidateQueries({ queryKey: ['initScripts'] });
    },
    onError: (err: Error) => {
      message.error(`删除失败: ${err.message}`);
    },
  });

  const handleAddScript = () => {
    setEditingGroup(null);
    setScripts([]);
    scriptForm.resetFields();
    scriptForm.setFieldsValue({ phase: 'pre-join', enabled: true });
    setScriptModalVisible(true);
  };

  const handleEditScript = (group: ScriptGroup) => {
    setEditingGroup(group);
    setScripts(group.scripts || []);
    scriptForm.setFieldsValue({
      name: group.name,
      description: group.description,
      phase: group.phase,
      enabled: group.enabled,
    });
    setScriptModalVisible(true);
  };

  const handleSaveScript = () => {
    scriptForm.validateFields().then(values => {
      const data = {
        ...values,
        scripts,
      };

      if (editingGroup) {
        updateScriptMutation.mutate({ id: editingGroup.id, data });
      } else {
        createMutation.mutate(data);
      }
    });
  };

  const getSupportedPlatforms = (group: ScriptGroup) => {
    if (!group.scripts || group.scripts.length === 0) return '无';

    const platforms = group.scripts.map(s => {
      if (s.os === '*' && s.arch === '*') return '通用';
      const os = s.os === '*' ? '*' : s.os;
      const arch = s.arch === '*' ? '*' : s.arch;
      return `${os}×${arch}`;
    });

    return platforms.join(', ');
  };

  const scriptColumns = [
    {
      title: '脚本名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: ScriptGroup) => (
        <Space>
          <CodeOutlined />
          <span>{name}</span>
          {record.builtin && <Tag>内置</Tag>}
        </Space>
      ),
    },
    {
      title: '执行阶段',
      dataIndex: 'phase',
      key: 'phase',
      width: 120,
      render: (phase: ScriptPhase) => (
        <Tag color={phaseColors[phase]}>{phaseLabels[phase]}</Tag>
      ),
    },
    {
      title: '支持平台',
      key: 'platforms',
      width: 250,
      render: (_: unknown, record: ScriptGroup) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {getSupportedPlatforms(record)}
        </Text>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      render: (enabled: boolean, record: ScriptGroup) => (
        <Switch
          checked={enabled}
          onChange={(checked) => toggleMutation.mutate({ id: record.id, enabled: checked })}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_: unknown, record: ScriptGroup) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => handleEditScript(record)}
            />
          </Tooltip>
          {!record.builtin && (
            <Popconfirm
              title="确定删除此脚本分组？"
              onConfirm={() => deleteMutation.mutate(record.id)}
            >
              <Tooltip title="删除">
                <Button type="text" danger icon={<DeleteOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

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
        message="节点初始化配置"
        description={
          <div>
            <Paragraph style={{ marginBottom: 8 }}>
              初始化脚本在节点加入集群时自动执行。Pre-join 脚本在 kubeadm join 之前执行（如配置私有仓库、禁用防火墙），
              Post-join 脚本在节点加入集群后执行（如安装驱动、添加标签）。
            </Paragraph>
            <Paragraph style={{ marginBottom: 8 }}>
              <strong>平台匹配：</strong>系统会自动检测节点的操作系统和 CPU 架构，按优先级匹配脚本：
              精确匹配 &gt; OS 通用 &gt; 架构通用 &gt; 全通用。
            </Paragraph>
            <Paragraph style={{ marginBottom: 0 }}>
              <strong>支持的系统：</strong>Ubuntu、CentOS、Debian、RHEL、openEuler、Rocky Linux、AlmaLinux、Kylin、UOS 等。
              系统名称来自 /etc/os-release 的 ID 字段。
            </Paragraph>
          </div>
        }
        type="info"
        showIcon
        style={{ marginBottom: 24 }}
      />

      <Card
        className="glass-card"
        title="脚本分组"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddScript}>
            新建分组
          </Button>
        }
      >
        <Table
          columns={scriptColumns}
          dataSource={scriptGroups}
          rowKey="id"
          loading={isLoading}
          pagination={false}
        />
      </Card>

      <Modal
        title={editingGroup ? '编辑脚本分组' : '新建脚本分组'}
        open={scriptModalVisible}
        onOk={handleSaveScript}
        onCancel={() => setScriptModalVisible(false)}
        width={800}
        confirmLoading={createMutation.isPending || updateScriptMutation.isPending}
      >
        <Form form={scriptForm} layout="vertical">
          <Form.Item
            name="name"
            label="分组名称"
            rules={[{ required: true, message: '请输入分组名称' }]}
          >
            <Input placeholder="如：配置私有镜像仓库" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input placeholder="简要描述此脚本的功能" />
          </Form.Item>
          <Form.Item
            name="phase"
            label="执行阶段"
            rules={[{ required: true }]}
          >
            <Select>
              <Select.Option value="pre-join">Pre-join（kubeadm join 之前）</Select.Option>
              <Select.Option value="post-join">Post-join（kubeadm join 之后）</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item label="平台脚本">
            <PlatformScriptEditor scripts={scripts} onChange={setScripts} />
          </Form.Item>

          <Alert
            message="支持的变量"
            description={
              <Text code>
                {'${NODE_IP}'}, {'${NODE_NAME}'}, {'${REGISTRY_URL}'}, {'${CONTROL_PLANE_IP}'}
              </Text>
            }
            type="info"
            style={{ marginTop: 16 }}
          />
        </Form>
      </Modal>
    </div>
  );
};

export default NodeInitConfig;

import React from 'react';
import { Form, Input, Button, Card, Typography, message, Space, Select } from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createProject, getTeams } from '../../services/api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const ProjectCreate: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const defaultTeam = searchParams.get('team') || undefined;
  const [form] = Form.useForm();

  const { data: teamsData } = useQuery({
    queryKey: ['teams'],
    queryFn: () => getTeams().then(res => res.data),
  });

  const createMutation = useMutation({
    mutationFn: createProject,
    onSuccess: () => {
      message.success('项目创建成功');
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      navigate('/projects');
    },
    onError: (error: Error) => {
      message.error(`创建失败: ${error.message}`);
    },
  });

  const onFinish = (values: {
    name: string;
    team: string;
    displayName?: string;
    description?: string;
  }) => {
    const project = {
      name: values.name,
      team: values.team,
      displayName: values.displayName || values.name,
      description: values.description,
    };

    createMutation.mutate(project);
  };

  return (
    <div>
      <Title level={2}>创建项目</Title>
      <Text type="secondary">
        项目对应 Kubernetes Namespace，属于某个团队
      </Text>

      <Card style={{ marginTop: 16 }} className="glass-card">
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{
            team: defaultTeam,
          }}
        >
          <Form.Item
            name="team"
            label="所属团队"
            rules={[{ required: true, message: '请选择所属团队' }]}
          >
            <Select placeholder="选择团队">
              {teamsData?.items?.map(team => (
                <Select.Option key={team.name} value={team.name}>
                  {team.displayName || team.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="name"
            label="项目标识"
            rules={[
              { required: true, message: '请输入项目标识' },
              { pattern: /^[a-z0-9][a-z0-9-]*[a-z0-9]$/, message: '只能包含小写字母、数字和连字符' },
            ]}
            tooltip="唯一标识，将作为 Namespace 名称"
          >
            <Input placeholder="例如: my-project" />
          </Form.Item>

          <Form.Item
            name="displayName"
            label="显示名称"
          >
            <Input placeholder="例如: 我的项目" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea rows={3} placeholder="项目描述..." />
          </Form.Item>

          <Form.Item style={{ marginTop: 24 }}>
            <Space>
              <Button type="primary" htmlType="submit" loading={createMutation.isPending}>
                创建项目
              </Button>
              <Button onClick={() => navigate('/projects')}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default ProjectCreate;


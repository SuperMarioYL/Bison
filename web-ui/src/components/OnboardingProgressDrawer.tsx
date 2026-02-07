import React, { useEffect } from 'react';
import {
  Drawer,
  Steps,
  Typography,
  Space,
  Tag,
  Alert,
  Button,
  Spin,
  Descriptions,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getOnboardingJob,
  cancelOnboardingJob,
  OnboardingJobStatus,
  SubStep,
  SubStepStatus,
} from '../services/api';

const { Text, Paragraph } = Typography;

interface OnboardingProgressDrawerProps {
  jobId: string | null;
  open: boolean;
  onClose: () => void;
}

const stepTitles = [
  '连接测试',
  '平台检测',
  '环境检查',
  'Pre-join 脚本',
  '获取 Join Token',
  '执行 kubeadm join',
  'Post-join 脚本',
  '等待节点就绪',
  '启用节点',
];

const getStepStatus = (currentStep: number, stepIndex: number, jobStatus: OnboardingJobStatus) => {
  if (jobStatus === 'failed' && currentStep === stepIndex + 1) {
    return 'error';
  }
  if (currentStep > stepIndex + 1) {
    return 'finish';
  }
  if (currentStep === stepIndex + 1) {
    return 'process';
  }
  return 'wait';
};

const getSubStepIcon = (status: SubStepStatus) => {
  switch (status) {
    case 'success':
      return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
    case 'failed':
      return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
    case 'running':
      return <LoadingOutlined style={{ color: '#1890ff' }} />;
    case 'skipped':
      return <MinusCircleOutlined style={{ color: '#d9d9d9' }} />;
    default:
      return <ClockCircleOutlined style={{ color: '#d9d9d9' }} />;
  }
};

const SubStepList: React.FC<{ subSteps: SubStep[] }> = ({ subSteps }) => {
  if (!subSteps || subSteps.length === 0) return null;

  return (
    <div style={{ marginTop: 8, marginLeft: 24 }}>
      {subSteps.map((subStep, index) => (
        <div key={index} style={{ marginBottom: 4 }}>
          <Space>
            {getSubStepIcon(subStep.status)}
            <Text type={subStep.status === 'failed' ? 'danger' : undefined}>
              {subStep.name}
            </Text>
          </Space>
          {subStep.error && (
            <div style={{ marginLeft: 22, marginTop: 4 }}>
              <Text type="danger" style={{ fontSize: 12 }}>
                {subStep.error}
              </Text>
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

const OnboardingProgressDrawer: React.FC<OnboardingProgressDrawerProps> = ({
  jobId,
  open,
  onClose,
}) => {
  const queryClient = useQueryClient();

  const { data: job, isLoading } = useQuery({
    queryKey: ['onboardingJob', jobId],
    queryFn: () => getOnboardingJob(jobId!).then(res => res.data),
    enabled: !!jobId && open,
    refetchInterval: (query) => {
      // Stop polling when job is completed
      const data = query.state.data;
      if (data && ['success', 'failed', 'cancelled'].includes(data.status)) {
        return false;
      }
      return 2000; // Poll every 2 seconds
    },
  });

  const cancelMutation = useMutation({
    mutationFn: cancelOnboardingJob,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['onboardingJob', jobId] });
    },
  });

  useEffect(() => {
    if (!open) {
      // Refresh nodes list when drawer closes
      queryClient.invalidateQueries({ queryKey: ['managedNodes'] });
    }
  }, [open, queryClient]);

  const handleCancel = () => {
    if (jobId) {
      cancelMutation.mutate(jobId);
    }
  };

  const isRunning = job?.status === 'pending' || job?.status === 'running';
  const isSuccess = job?.status === 'success';
  const isFailed = job?.status === 'failed';
  const isCancelled = job?.status === 'cancelled';

  const getStatusTag = () => {
    if (!job) return null;
    switch (job.status) {
      case 'pending':
        return <Tag color="default">等待中</Tag>;
      case 'running':
        return <Tag color="processing">执行中</Tag>;
      case 'success':
        return <Tag color="success">成功</Tag>;
      case 'failed':
        return <Tag color="error">失败</Tag>;
      case 'cancelled':
        return <Tag color="warning">已取消</Tag>;
      default:
        return null;
    }
  };

  // Build step items with sub-steps
  const getStepItems = () => {
    if (!job) return stepTitles.map(title => ({ title }));

    return stepTitles.map((title, index) => {
      const stepNumber = index + 1;
      const status = getStepStatus(job.currentStep, index, job.status);

      // Show sub-steps for script execution steps (4 and 7)
      let description = null;
      if ((stepNumber === 4 || stepNumber === 7) && job.currentStep === stepNumber && job.subSteps) {
        description = <SubStepList subSteps={job.subSteps} />;
      }

      return {
        title,
        status,
        description,
      };
    });
  };

  return (
    <Drawer
      title="节点添加进度"
      placement="right"
      width={450}
      open={open}
      onClose={onClose}
      extra={getStatusTag()}
    >
      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 50 }}>
          <Spin size="large" />
        </div>
      ) : job ? (
        <div>
          <Descriptions column={1} size="small" style={{ marginBottom: 24 }}>
            <Descriptions.Item label="节点 IP">{job.nodeIP}</Descriptions.Item>
            {job.nodeName && (
              <Descriptions.Item label="节点名称">{job.nodeName}</Descriptions.Item>
            )}
            {job.platform.os && (
              <Descriptions.Item label="平台">
                <Space>
                  <Tag color="blue">{job.platform.os} {job.platform.version}</Tag>
                  <Tag color="green">{job.platform.arch}</Tag>
                </Space>
              </Descriptions.Item>
            )}
          </Descriptions>

          <Steps
            direction="vertical"
            size="small"
            current={job.currentStep - 1}
            items={getStepItems()}
          />

          {job.stepMessage && (
            <Alert
              message="当前状态"
              description={job.stepMessage}
              type="info"
              showIcon
              style={{ marginTop: 24 }}
            />
          )}

          {isFailed && job.errorMessage && (
            <Alert
              message="错误信息"
              description={job.errorMessage}
              type="error"
              showIcon
              style={{ marginTop: 16 }}
            />
          )}

          {isSuccess && (
            <Alert
              message="节点添加成功"
              description={
                <Paragraph style={{ marginBottom: 0 }}>
                  节点 <Text strong>{job.nodeName}</Text> 已成功加入集群并启用。
                </Paragraph>
              }
              type="success"
              showIcon
              style={{ marginTop: 16 }}
            />
          )}

          {isCancelled && (
            <Alert
              message="操作已取消"
              type="warning"
              showIcon
              style={{ marginTop: 16 }}
            />
          )}

          <div style={{ marginTop: 24 }}>
            {isRunning && (
              <Button
                danger
                onClick={handleCancel}
                loading={cancelMutation.isPending}
              >
                取消操作
              </Button>
            )}
            {!isRunning && (
              <Button onClick={onClose}>
                关闭
              </Button>
            )}
          </div>
        </div>
      ) : (
        <div style={{ textAlign: 'center', padding: 50 }}>
          <Text type="secondary">无法加载任务信息</Text>
        </div>
      )}
    </Drawer>
  );
};

export default OnboardingProgressDrawer;

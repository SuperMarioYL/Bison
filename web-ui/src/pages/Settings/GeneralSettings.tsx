import React from 'react';
import { Card, Descriptions, Typography, Alert, Spin, Tag } from 'antd';
import { CloudOutlined, LineChartOutlined, DollarOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { getSettings, getFeatures } from '../../services/api';

const { Text, Paragraph, Title } = Typography;

const GeneralSettings: React.FC = () => {
  const { data: settings, isLoading: settingsLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: () => getSettings().then(res => res.data),
  });

  const { data: features, isLoading: featuresLoading } = useQuery({
    queryKey: ['features'],
    queryFn: () => getFeatures().then(res => res.data),
  });

  const isLoading = settingsLoading || featuresLoading;

  return (
    <div>
      <Alert
        message="配置信息"
        description={
          <Paragraph style={{ marginBottom: 0 }}>
            以下配置通过 Helm values 管理，如需修改请更新 Helm values 并重新部署。
            OIDC 用户映射通过 Capsule Tenant 的 owners 字段管理。
          </Paragraph>
        }
        type="info"
        showIcon
        style={{ marginBottom: 24 }}
      />

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 50 }}>
          <Spin size="large" />
        </div>
      ) : (
        <Card className="glass-card">
          <Descriptions column={1} bordered>
            <Descriptions.Item 
              label={
                <span>
                  <LineChartOutlined style={{ marginRight: 8 }} />
                  Prometheus 地址
                </span>
              }
            >
              {settings?.prometheusUrl ? (
                <Text code>{settings.prometheusUrl}</Text>
              ) : (
                <Tag color="default">未配置</Tag>
              )}
            </Descriptions.Item>
            
            <Descriptions.Item 
              label={
                <span>
                  <DollarOutlined style={{ marginRight: 8 }} />
                  OpenCost 地址
                </span>
              }
            >
              {settings?.opencostUrl ? (
                <Text code>{settings.opencostUrl}</Text>
              ) : (
                <Tag color="default">未配置</Tag>
              )}
            </Descriptions.Item>

            <Descriptions.Item 
              label={
                <span>
                  <CloudOutlined style={{ marginRight: 8 }} />
                  费用统计
                </span>
              }
            >
              {features?.costEnabled ? (
                <Tag color="success">已启用</Tag>
              ) : (
                <Tag color="default">未启用</Tag>
              )}
            </Descriptions.Item>
          </Descriptions>

          <div style={{ marginTop: 24 }}>
            <Title level={5}>如何修改配置</Title>
            <Paragraph>
              <Text code>helm upgrade bison ./deploy/charts/bison --set dependencies.prometheus.url=http://your-prometheus:9090</Text>
            </Paragraph>
          </div>
        </Card>
      )}
    </div>
  );
};

export default GeneralSettings;


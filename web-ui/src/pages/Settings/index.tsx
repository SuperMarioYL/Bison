import React from 'react';
import { Tabs, Typography } from 'antd';
import {
  SettingOutlined,
  DollarOutlined,
  BellOutlined,
  DashboardOutlined,
  CloudOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  ToolOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import BillingConfig from './BillingConfig';
import AlertConfig from './AlertConfig';
import SystemStatus from './SystemStatus';
import GeneralSettings from './GeneralSettings';
import ResourceConfig from './ResourceConfig';
import ControlPlaneConfig from './ControlPlaneConfig';
import NodeInitConfig from './NodeInitConfig';
import ConfigTransfer from './ConfigTransfer';
import { useFeatures } from '../../hooks/useFeatures';

const { Title } = Typography;

const Settings: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: features } = useFeatures();

  const getCurrentTab = () => {
    const path = location.pathname;
    if (path.includes('/settings/resources')) return 'resources';
    if (path.includes('/settings/billing')) return 'billing';
    if (path.includes('/settings/alerts')) return 'alerts';
    if (path.includes('/settings/control-plane')) return 'control-plane';
    if (path.includes('/settings/node-init')) return 'node-init';
    if (path.includes('/settings/transfer')) return 'transfer';
    if (path.includes('/settings/status')) return 'status';
    return 'general';
  };

  const items = [
    {
      key: 'general',
      label: (
        <span>
          <CloudOutlined />
          基本配置
        </span>
      ),
    },
    {
      key: 'resources',
      label: (
        <span>
          <AppstoreOutlined />
          资源配置
        </span>
      ),
    },
    ...(features?.costEnabled !== false ? [{
      key: 'billing',
      label: (
        <span>
          <DollarOutlined />
          计费配置
        </span>
      ),
    }] : []),
    ...(features?.capsuleEnabled !== false ? [{
      key: 'alerts',
      label: (
        <span>
          <BellOutlined />
          告警配置
        </span>
      ),
    }] : []),
    {
      key: 'control-plane',
      label: (
        <span>
          <CloudServerOutlined />
          控制面配置
        </span>
      ),
    },
    {
      key: 'node-init',
      label: (
        <span>
          <ToolOutlined />
          节点初始化配置
        </span>
      ),
    },
    {
      key: 'transfer',
      label: (
        <span>
          <SwapOutlined />
          配置迁移
        </span>
      ),
    },
    {
      key: 'status',
      label: (
        <span>
          <DashboardOutlined />
          系统状态
        </span>
      ),
    },
  ];

  const handleTabChange = (key: string) => {
    if (key === 'general') {
      navigate('/settings');
    } else {
      navigate(`/settings/${key}`);
    }
  };

  return (
    <div>
      <Title level={2}>
        <SettingOutlined style={{ marginRight: 8 }} />
        系统设置
      </Title>

      <Tabs
        activeKey={getCurrentTab()}
        items={items}
        onChange={handleTabChange}
      />

      <div style={{ marginTop: 16 }}>
        <Routes>
          <Route index element={<GeneralSettings />} />
          <Route path="resources" element={<ResourceConfig />} />
          <Route path="billing" element={<BillingConfig />} />
          <Route path="alerts" element={<AlertConfig />} />
          <Route path="control-plane" element={<ControlPlaneConfig />} />
          <Route path="node-init" element={<NodeInitConfig />} />
          <Route path="transfer" element={<ConfigTransfer />} />
          <Route path="status" element={<SystemStatus />} />
        </Routes>
      </div>
    </div>
  );
};

export default Settings;

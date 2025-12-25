import React from 'react';
import { Tabs, Typography } from 'antd';
import { 
  SettingOutlined, 
  DollarOutlined, 
  BellOutlined, 
  DashboardOutlined,
  CloudOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import BillingConfig from './BillingConfig';
import AlertConfig from './AlertConfig';
import SystemStatus from './SystemStatus';
import GeneralSettings from './GeneralSettings';
import ResourceConfig from './ResourceConfig';

const { Title } = Typography;

const Settings: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const getCurrentTab = () => {
    const path = location.pathname;
    if (path.includes('/settings/resources')) return 'resources';
    if (path.includes('/settings/billing')) return 'billing';
    if (path.includes('/settings/alerts')) return 'alerts';
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
    {
      key: 'billing',
      label: (
        <span>
          <DollarOutlined />
          计费配置
        </span>
      ),
    },
    {
      key: 'alerts',
      label: (
        <span>
          <BellOutlined />
          告警配置
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
          <Route path="status" element={<SystemStatus />} />
        </Routes>
      </div>
    </div>
  );
};

export default Settings;

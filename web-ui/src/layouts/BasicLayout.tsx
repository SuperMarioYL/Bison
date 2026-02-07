import React from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, Dropdown, Space, Typography, Tooltip } from 'antd';
import {
  DashboardOutlined,
  ClusterOutlined,
  UserOutlined,
  LogoutOutlined,
  SettingOutlined,
  ApartmentOutlined,
  SunOutlined,
  MoonOutlined,
  ProjectOutlined,
  BarChartOutlined,
  AuditOutlined,
} from '@ant-design/icons';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { useFeatures } from '../hooks/useFeatures';
import './BasicLayout.css';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const BasicLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { logout, username, authEnabled } = useAuth();
  const { theme, toggleTheme, isDark } = useTheme();
  const { data: features } = useFeatures();

  const handleLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  const userMenuItems = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout,
    },
  ];

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: '资源总览',
    },
    {
      key: '/cluster/nodes',
      icon: <ClusterOutlined />,
      label: '集群节点',
    },
    ...(features?.capsuleEnabled !== false ? [
      {
        key: '/teams',
        icon: <ApartmentOutlined />,
        label: '团队管理',
      },
      {
        key: '/projects',
        icon: <ProjectOutlined />,
        label: '项目管理',
      },
      {
        key: '/users',
        icon: <UserOutlined />,
        label: '用户管理',
      },
    ] : []),
    ...(features?.costEnabled !== false ? [
      {
        key: '/reports',
        icon: <BarChartOutlined />,
        label: '报表中心',
      },
    ] : []),
    {
      key: '/audit',
      icon: <AuditOutlined />,
      label: '审计日志',
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: '系统设置',
    },
  ];

  const getSelectedKey = () => {
    const path = location.pathname;
    if (path.startsWith('/teams')) return '/teams';
    if (path.startsWith('/projects')) return '/projects';
    if (path.startsWith('/users')) return '/users';
    if (path.startsWith('/cluster/nodes')) return '/cluster/nodes';
    if (path.startsWith('/reports')) return '/reports';
    if (path.startsWith('/audit')) return '/audit';
    if (path.startsWith('/settings')) return '/settings';
    return path;
  };

  return (
    <Layout className={`app-layout ${theme}`}>
      <Sider
        width={240}
        className="app-sider glass-sidebar"
      >
        <div className="sider-header">
          <div className="logo-wrapper">
            <img src="/logo.png" alt="Bison Logo" className="logo" />
          </div>
          <span className="logo-text">Bison</span>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[getSelectedKey()]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          className="app-menu"
        />
        <div className="sider-footer">
          <Text className="version-text">v3.0.0{features ? ` (${[
            features.capsuleEnabled && 'Capsule',
            features.costEnabled && 'OpenCost',
            features.prometheusEnabled && 'Prometheus',
          ].filter(Boolean).join(' + ') || '基础模式'})` : ''}</Text>
        </div>
      </Sider>
      <Layout className="main-layout">
        <Header className="app-header glass-header">
          <div className="header-title">
            集群资源调度计费平台
          </div>
          <Space size={16} className="header-actions">
            <Tooltip title={isDark ? '切换到浅色模式' : '切换到深色模式'}>
              <Button
                type="text"
                icon={isDark ? <SunOutlined /> : <MoonOutlined />}
                onClick={toggleTheme}
                className="theme-toggle-btn"
              />
            </Tooltip>
            {authEnabled && (
              <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
                <Button type="text" className="user-btn">
                  <Space>
                    <div className="user-avatar">
                      <UserOutlined />
                    </div>
                    <Text className="username">{username || '用户'}</Text>
                  </Space>
                </Button>
              </Dropdown>
            )}
          </Space>
        </Header>
        <Content className="app-content">
          <div className="content-wrapper glass-content">
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  );
};

export default BasicLayout;

import React, { useState, useEffect } from 'react';
import { Form, Input, Button, message, Typography } from 'antd';
import { UserOutlined, LockOutlined, SunOutlined, MoonOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { login, getAuthStatus } from '../../services/api';
import { useAuth } from '../../contexts/AuthContext';
import { useTheme } from '../../contexts/ThemeContext';
import './Login.css';

const { Title, Text } = Typography;

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [checkingAuth, setCheckingAuth] = useState(true);
  const navigate = useNavigate();
  const { checkAuth } = useAuth();
  const { theme, toggleTheme, isDark } = useTheme();

  useEffect(() => {
    const checkAuthStatus = async () => {
      try {
        const { data } = await getAuthStatus();
        if (!data.authEnabled) {
          navigate('/dashboard', { replace: true });
          return;
        }

        const token = localStorage.getItem('token');
        if (token) {
          navigate('/dashboard', { replace: true });
          return;
        }
      } catch (error) {
        console.error('Failed to check auth status:', error);
      } finally {
        setCheckingAuth(false);
      }
    };

    checkAuthStatus();
  }, [navigate]);

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const { data } = await login(values);
      localStorage.setItem('token', data.token);
      localStorage.setItem('username', data.username);
      localStorage.setItem('tokenExpires', String(data.expiresAt));
      message.success('登录成功');
      await checkAuth();
      navigate('/dashboard', { replace: true });
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      message.error(err.response?.data?.error || '登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
    }
  };

  if (checkingAuth) {
    return (
      <div className={`login-container ${theme}`}>
        <div className="gradient-bg" />
        <div className="gradient-orbs">
          <div className="orb orb-1" />
          <div className="orb orb-2" />
          <div className="orb orb-3" />
          <div className="orb orb-4" />
        </div>
        <div className="login-card glass">
          <Text className="loading-text">检查认证状态...</Text>
        </div>
      </div>
    );
  }

  return (
    <div className={`login-container ${theme}`}>
      {/* Animated gradient background */}
      <div className="gradient-bg" />
      
      {/* Floating gradient orbs */}
      <div className="gradient-orbs">
        <div className="orb orb-1" />
        <div className="orb orb-2" />
        <div className="orb orb-3" />
        <div className="orb orb-4" />
      </div>
      
      {/* Theme toggle button */}
      <button className="theme-toggle glass" onClick={toggleTheme} aria-label="Toggle theme">
        {isDark ? <SunOutlined /> : <MoonOutlined />}
      </button>

      {/* Login card with glassmorphism */}
      <div className="login-card glass">
        <div className="login-header">
          <div className="logo-container">
            <img src="/logo.png" alt="Bison Logo" className="logo" />
          </div>
          <Title level={2} className="login-title">Bison</Title>
          <Text className="login-subtitle">集群资源调度平台</Text>
        </div>

        <Form
          name="login"
          onFinish={onFinish}
          size="large"
          autoComplete="off"
          className="login-form"
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              prefix={<UserOutlined className="input-icon" />}
              placeholder="用户名"
              className="login-input"
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined className="input-icon" />}
              placeholder="密码"
              className="login-input"
            />
          </Form.Item>

          <Form.Item className="login-button-item">
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              className="login-button"
            >
              登 录
            </Button>
          </Form.Item>
        </Form>

        <div className="login-footer">
          <Text className="footer-text">Powered by Kubernetes & Kueue</Text>
        </div>
      </div>
    </div>
  );
};

export default Login;

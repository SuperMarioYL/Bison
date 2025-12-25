import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import App from './App';
import { AuthProvider } from './contexts/AuthContext';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import './styles/theme.css';
import './index.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 30000,
    },
  },
});

// Themed App wrapper to access theme context
const ThemedApp: React.FC = () => {
  const { isDark } = useTheme();

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: isDark ? '#0a84ff' : '#0071e3',
          borderRadius: 12,
          fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', Roboto, sans-serif",
          colorBgContainer: isDark ? 'rgba(44, 44, 46, 0.8)' : 'rgba(255, 255, 255, 0.8)',
          colorBgElevated: isDark ? 'rgba(58, 58, 60, 0.95)' : 'rgba(255, 255, 255, 0.95)',
          colorBgLayout: isDark ? '#000000' : '#f5f5f7',
          colorText: isDark ? '#f5f5f7' : '#1d1d1f',
          colorTextSecondary: isDark ? '#98989d' : '#86868b',
          colorBorder: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.08)',
          colorBorderSecondary: isDark ? 'rgba(255, 255, 255, 0.05)' : 'rgba(0, 0, 0, 0.04)',
          boxShadow: isDark 
            ? '0 4px 24px rgba(0, 0, 0, 0.4)' 
            : '0 4px 24px rgba(0, 0, 0, 0.08)',
          boxShadowSecondary: isDark
            ? '0 8px 40px rgba(0, 0, 0, 0.5)'
            : '0 8px 40px rgba(0, 0, 0, 0.12)',
        },
        components: {
          Card: {
            borderRadiusLG: 16,
            paddingLG: 24,
          },
          Button: {
            borderRadius: 10,
            controlHeight: 40,
            fontWeight: 500,
          },
          Input: {
            borderRadius: 10,
            controlHeight: 44,
          },
          Select: {
            borderRadius: 10,
            controlHeight: 44,
          },
          Table: {
            borderRadiusLG: 12,
            headerBg: isDark ? 'rgba(255, 255, 255, 0.04)' : 'rgba(0, 0, 0, 0.02)',
          },
          Modal: {
            borderRadiusLG: 20,
          },
          Menu: {
            itemBorderRadius: 10,
            subMenuItemBorderRadius: 8,
          },
          Tag: {
            borderRadiusSM: 6,
          },
          Statistic: {
            contentFontSize: 28,
          },
        },
      }}
    >
      <BrowserRouter>
        <AuthProvider>
          <App />
        </AuthProvider>
      </BrowserRouter>
    </ConfigProvider>
  );
};

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ThemedApp />
      </ThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>
);

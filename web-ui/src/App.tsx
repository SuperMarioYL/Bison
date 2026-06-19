import React, { Suspense, lazy } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Spin } from 'antd';
import BasicLayout from './layouts/BasicLayout';
import ProtectedRoute from './components/ProtectedRoute';
import { useFeatures } from './hooks/useFeatures';

// Route-level code splitting: each page (and its heavy deps such as echarts on
// the node-detail page) is only downloaded when first visited.
const Login = lazy(() => import('./pages/Login'));
const Dashboard = lazy(() => import('./pages/Dashboard'));
const ProjectList = lazy(() => import('./pages/Project/ProjectList'));
const ProjectCreate = lazy(() => import('./pages/Project/ProjectCreate'));
const ProjectDetail = lazy(() => import('./pages/Project/ProjectDetail'));
const ClusterNodes = lazy(() => import('./pages/Cluster/ClusterNodes'));
const NodeDetail = lazy(() => import('./pages/Cluster/NodeDetail'));
const TeamList = lazy(() => import('./pages/Team/TeamList'));
const TeamCreate = lazy(() => import('./pages/Team/TeamCreate'));
const TeamDetail = lazy(() => import('./pages/Team/TeamDetail'));
const UserList = lazy(() => import('./pages/User/UserList'));
const UserDetail = lazy(() => import('./pages/User/UserDetail'));
const AuditList = lazy(() => import('./pages/Audit/AuditList'));
const ReportCenter = lazy(() => import('./pages/Report/ReportCenter'));
const Settings = lazy(() => import('./pages/Settings'));

const PageFallback: React.FC = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '40vh' }}>
    <Spin size="large" />
  </div>
);

const App: React.FC = () => {
  const { data: features } = useFeatures();

  return (
    <Suspense fallback={<PageFallback />}>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={
          <ProtectedRoute>
            <BasicLayout />
          </ProtectedRoute>
        }>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          {features?.capsuleEnabled !== false && (
            <>
              <Route path="teams" element={<TeamList />} />
              <Route path="teams/create" element={<TeamCreate />} />
              <Route path="teams/:name" element={<TeamDetail />} />
              <Route path="projects" element={<ProjectList />} />
              <Route path="projects/create" element={<ProjectCreate />} />
              <Route path="projects/:name" element={<ProjectDetail />} />
              <Route path="users" element={<UserList />} />
              <Route path="users/:email" element={<UserDetail />} />
            </>
          )}
          <Route path="cluster/nodes" element={<ClusterNodes />} />
          <Route path="cluster/nodes/:name" element={<NodeDetail />} />
          {features?.costEnabled !== false && (
            <Route path="reports" element={<ReportCenter />} />
          )}
          <Route path="audit" element={<AuditList />} />
          <Route path="settings/*" element={<Settings />} />
        </Route>
      </Routes>
    </Suspense>
  );
};

export default App;

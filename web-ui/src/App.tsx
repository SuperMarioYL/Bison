import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import BasicLayout from './layouts/BasicLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import ProjectList from './pages/Project/ProjectList';
import ProjectCreate from './pages/Project/ProjectCreate';
import ProjectDetail from './pages/Project/ProjectDetail';
import ClusterNodes from './pages/Cluster/ClusterNodes';
import NodeDetail from './pages/Cluster/NodeDetail';
import TeamList from './pages/Team/TeamList';
import TeamCreate from './pages/Team/TeamCreate';
import TeamDetail from './pages/Team/TeamDetail';
import UserList from './pages/User/UserList';
import UserDetail from './pages/User/UserDetail';
import AuditList from './pages/Audit/AuditList';
import ReportCenter from './pages/Report/ReportCenter';
import Settings from './pages/Settings';
import ProtectedRoute from './components/ProtectedRoute';
import { useFeatures } from './hooks/useFeatures';

const App: React.FC = () => {
  const { data: features } = useFeatures();

  return (
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
  );
};

export default App;

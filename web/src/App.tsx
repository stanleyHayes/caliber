import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from './components/AppShell';
import { ProtectedRoute } from './components/ProtectedRoute';
import { SessionBootstrap } from './components/SessionBootstrap';
import { DashboardPage } from './pages/DashboardPage';
import { AgentPage } from './pages/AgentPage';
import { EmployerFlowPage } from './pages/EmployerFlowPage';
import { InterviewPage } from './pages/InterviewPage';
import { LoginPage } from './pages/LoginPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { RadarPage } from './pages/RadarPage';
import { RegisterPage } from './pages/RegisterPage';
import { queryClient } from './query/client';

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <SessionBootstrap>
          <Routes>
            <Route element={<AppShell />}>
              <Route element={<ProtectedRoute />}>
                <Route path="/" element={<DashboardPage />} />
                <Route path="/roles/new" element={<EmployerFlowPage />} />
                <Route path="/interview" element={<InterviewPage />} />
                <Route path="/agent" element={<AgentPage />} />
                <Route path="/radar" element={<RadarPage />} />
              </Route>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
              <Route path="/404" element={<NotFoundPage />} />
              <Route path="*" element={<Navigate to="/404" replace />} />
            </Route>
          </Routes>
        </SessionBootstrap>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

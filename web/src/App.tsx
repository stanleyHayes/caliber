import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from './components/AppShell';
import { ProtectedRoute } from './components/ProtectedRoute';
import { RouteSeo } from './components/RouteSeo';
import { SessionBootstrap } from './components/SessionBootstrap';
import { DashboardPage } from './pages/DashboardPage';
import { LandingPage } from './pages/LandingPage';
import { AgentPage } from './pages/AgentPage';
import { EmployerFlowPage } from './pages/EmployerFlowPage';
import { InterviewPage } from './pages/InterviewPage';
import { LoginPage } from './pages/LoginPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { ProfilePage } from './pages/ProfilePage';
import { RadarPage } from './pages/RadarPage';
import { RolesPage } from './pages/RolesPage';
import { RegisterPage } from './pages/RegisterPage';
import { queryClient } from './query/client';

/**
 * AppRoutes is the route tree shared between the client (CSR) and the build-time
 * prerenderer (SSR). It does not include the router or global providers so the
 * two entry points can supply the appropriate implementations.
 */
export function AppRoutes() {
  return (
    <SessionBootstrap>
      <RouteSeo />
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<LandingPage />} />
          <Route element={<ProtectedRoute />}>
            <Route path="/app" element={<DashboardPage />} />
            <Route path="/roles" element={<RolesPage />} />
            <Route path="/roles/new" element={<EmployerFlowPage />} />
            <Route path="/interview" element={<InterviewPage />} />
            <Route path="/profile" element={<ProfilePage />} />
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
  );
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </QueryClientProvider>
  );
}

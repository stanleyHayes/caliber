import { Box, Skeleton } from '@mui/material';
import { QueryClientProvider } from '@tanstack/react-query';
import { Suspense, lazy } from 'react';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from './components/AppShell';
import { ProtectedRoute } from './components/ProtectedRoute';
import { RouteSeo } from './components/RouteSeo';
import { SessionBootstrap } from './components/SessionBootstrap';
import { LandingPage } from './pages/LandingPage';
import { LoginPage } from './pages/LoginPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { RegisterPage } from './pages/RegisterPage';
import { queryClient } from './query/client';

// Public/marketing pages stay eagerly loaded so the build-time prerender
// pipeline can render them without awaiting async chunks (CAL-121).
// Authenticated app routes are lazy-loaded to reduce the initial JS bundle
// and improve LCP/INP on the public landing experience (CAL-125).
const DashboardPage = lazy(() => import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })));
const AgentPage = lazy(() => import('./pages/AgentPage').then((m) => ({ default: m.AgentPage })));
const EmployerFlowPage = lazy(() => import('./pages/EmployerFlowPage').then((m) => ({ default: m.EmployerFlowPage })));
const InterviewPage = lazy(() => import('./pages/InterviewPage').then((m) => ({ default: m.InterviewPage })));
const ProfilePage = lazy(() => import('./pages/ProfilePage').then((m) => ({ default: m.ProfilePage })));
const RadarPage = lazy(() => import('./pages/RadarPage').then((m) => ({ default: m.RadarPage })));
const RolesPage = lazy(() => import('./pages/RolesPage').then((m) => ({ default: m.RolesPage })));

function RouteFallback() {
  return (
    <Box sx={{ py: 4 }} aria-busy="true" aria-label="Loading page">
      <Skeleton variant="text" width="55%" height={40} sx={{ mb: 2 }} />
      <Skeleton variant="rectangular" height={160} sx={{ borderRadius: 2, mb: 2 }} />
      <Skeleton variant="text" width="80%" />
      <Skeleton variant="text" width="70%" />
      <Skeleton variant="text" width="90%" />
    </Box>
  );
}

/**
 * AppRoutes is the route tree shared between the client (CSR) and the build-time
 * prerenderer (SSR). It does not include the router or global providers so the
 * two entry points can supply the appropriate implementations.
 */
export function AppRoutes() {
  return (
    <SessionBootstrap>
      <RouteSeo />
      <Suspense fallback={<RouteFallback />}>
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
      </Suspense>
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

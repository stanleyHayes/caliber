import { Navigate, Outlet } from 'react-router-dom';

import { canScreenSelf } from '../lib/permissions';
import { useAuthStore } from '../stores/auth';

// Route guard for candidate-only surfaces (Flow B — the screening interview).
// Assumes an outer ProtectedRoute already ensured authentication; here we enforce
// the spec's persona model: screening interviews are taken by candidates, so a
// reviewer who lands on /interview is sent to their dashboard rather than into a
// candidate-only action that would 403. Role still loading (undefined) → render
// through and let it resolve.
export function RequireCandidate() {
  const role = useAuthStore((s) => s.user?.role);
  if (role && !canScreenSelf(role)) {
    return <Navigate to="/app" replace />;
  }
  return <Outlet />;
}

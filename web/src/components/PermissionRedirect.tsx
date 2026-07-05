import { Navigate, useInRouterContext } from 'react-router-dom';

export function PermissionRedirect({ to = '/app' }: { to?: string }) {
  const inRouter = useInRouterContext();
  return inRouter ? <Navigate to={to} replace /> : null;
}

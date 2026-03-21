import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import { hasMinRole } from '@/lib/roles';
import type { Role } from '@/types';

interface ProtectedRouteProps {
  minimumRole?: Role;
}

export function ProtectedRoute({ minimumRole = 'viewer' }: ProtectedRouteProps) {
  const { isAuthenticated, user } = useAuthStore();

  if (!isAuthenticated || !user) {
    return <Navigate to="/login" replace />;
  }

  if (!hasMinRole(user.role, minimumRole)) {
    return <Navigate to="/kb" replace />;
  }

  return <Outlet />;
}

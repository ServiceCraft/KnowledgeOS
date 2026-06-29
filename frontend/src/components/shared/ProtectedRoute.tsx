import { Navigate, Outlet } from 'react-router-dom';
import { useShallow } from 'zustand/react/shallow';
import { useAuthStore } from '@/stores/authStore';
import { hasMinRole } from '@/lib/roles';
import { LoadingState } from '@/components/shared/LoadingState';
import type { Role } from '@/types';

interface ProtectedRouteProps {
  minimumRole?: Role;
}

export function ProtectedRoute({ minimumRole = 'viewer' }: ProtectedRouteProps) {
  const { hasHydrated, isAuthenticated, user } = useAuthStore(
    useShallow((s) => ({
      hasHydrated: s._hasHydrated,
      isAuthenticated: s.isAuthenticated,
      user: s.user,
    }))
  );

  if (!hasHydrated) {
    return <LoadingState />;
  }

  if (!isAuthenticated || !user) {
    return <Navigate to="/login" replace />;
  }

  if (!hasMinRole(user.role, minimumRole)) {
    return <Navigate to="/kb" replace />;
  }

  return <Outlet />;
}

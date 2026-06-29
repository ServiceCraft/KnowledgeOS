import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useShallow } from 'zustand/react/shallow';
import { useAuthStore } from '@/stores/authStore';
import { needsCompanySelection } from '@/lib/tenantContext';
import { LoadingState } from '@/components/shared/LoadingState';

export function CompanyContextRoute() {
  const location = useLocation();
  const { hasHydrated, user, selectedCompanyId } = useAuthStore(
    useShallow((s) => ({
      hasHydrated: s._hasHydrated,
      user: s.user,
      selectedCompanyId: s.selectedCompanyId,
    }))
  );

  if (!hasHydrated) {
    return <LoadingState />;
  }

  if (needsCompanySelection(user, selectedCompanyId)) {
    if (user?.role === 'superadmin') {
      return <Navigate to="/admin/companies" replace state={{ from: location.pathname }} />;
    }
    return <Navigate to="/select-company" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}

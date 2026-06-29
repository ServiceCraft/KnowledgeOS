import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';

export function CompanyContextRoute() {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const selectedCompanyId = useAuthStore((s) => s.selectedCompanyId);

  if (user?.role === 'superadmin' && !selectedCompanyId) {
    return <Navigate to="/admin/companies" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}

import { useAuthStore } from '@/stores/authStore';
import { canWrite, canManageUsers, isSuperadmin } from '@/lib/roles';

// Centralized client-side permission checks. The backend is the source of truth
// (it enforces 403s); these only drive what the UI shows.
export function usePermissions() {
  const user = useAuthStore((s) => s.user);
  const role = user?.role;
  return {
    role,
    userId: user?.id,
    canWrite: canWrite(role),
    canManageUsers: canManageUsers(role),
    isSuperadmin: isSuperadmin(role),
  };
}

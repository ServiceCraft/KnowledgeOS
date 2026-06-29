import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/authStore';

export function useSelectCompanyContext() {
  const qc = useQueryClient();
  const setSelectedCompany = useAuthStore((s) => s.setSelectedCompany);

  return (companyId: string, companyName: string) => {
    setSelectedCompany(companyId, companyName);
    qc.clear();
  };
}

import { useQuery } from '@tanstack/react-query';
import { authApi } from '@/api/auth';
import { queryKeys } from '@/lib/queryKeys';

export function useAccessibleCompanies(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.auth.companies,
    queryFn: () => authApi.listCompanies(),
    enabled: options?.enabled ?? true,
  });
}

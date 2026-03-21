import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminApi, type CompanyFilter, type CreateCompanyRequest } from '@/api/admin';
import { queryKeys } from '@/lib/queryKeys';
import type { Company } from '@/types';

export function useCompaniesList(filters?: CompanyFilter) {
  return useQuery({
    queryKey: queryKeys.companies.list(filters),
    queryFn: () => adminApi.listCompanies(filters),
  });
}

export function useCreateCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateCompanyRequest) => adminApi.createCompany(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.companies.all }),
  });
}

export function useUpdateCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Company> }) =>
      adminApi.updateCompany(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.companies.all }),
  });
}

export function useDeleteCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: adminApi.deleteCompany,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.companies.all }),
  });
}

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminApi, type CompanyFilter, type CreateCompanyWithAdminRequest } from '@/api/admin';
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
    mutationFn: async (data: CreateCompanyWithAdminRequest) => {
      const company = await adminApi.createCompany({ name: data.name, tier: data.tier });
      await adminApi.createCompanyAdmin(company.id, {
        email: data.admin_email,
        password: data.admin_password,
      });
      return company;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.companies.all }),
  });
}

export function useCompanyDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.companies.detail(id),
    queryFn: () => adminApi.getCompany(id),
    enabled: !!id,
  });
}

export function useCreateCompanyAdmin(companyId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { email: string; password: string }) =>
      adminApi.createCompanyAdmin(companyId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.companies.detail(companyId) }),
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

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { pricingApi, type PricingFilter } from '@/api/pricing';
import { queryKeys } from '@/lib/queryKeys';

export function usePricingList(filters?: PricingFilter) {
  return useQuery({
    queryKey: queryKeys.pricing.list(filters),
    queryFn: () => pricingApi.list(filters),
  });
}

export function usePricingDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.pricing.detail(id),
    queryFn: () => pricingApi.getById(id),
    enabled: !!id,
  });
}

export function useCreatePricingNode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: pricingApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.pricing.all }),
  });
}

export function useUpdatePricingNode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Parameters<typeof pricingApi.update>[1]> }) =>
      pricingApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.pricing.all }),
  });
}

export function useDeletePricingNode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: pricingApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.pricing.all }),
  });
}

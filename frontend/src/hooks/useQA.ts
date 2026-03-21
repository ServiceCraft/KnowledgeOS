import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { qaApi, type QAFilter } from '@/api/qa';
import { queryKeys } from '@/lib/queryKeys';

export function useQAList(filters?: QAFilter) {
  return useQuery({
    queryKey: queryKeys.qa.list(filters),
    queryFn: () => qaApi.list(filters),
    placeholderData: keepPreviousData,
  });
}

export function useQADetail(id: string) {
  return useQuery({
    queryKey: queryKeys.qa.detail(id),
    queryFn: () => qaApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateQA() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: qaApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.qa.all }),
  });
}

export function useUpdateQA() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Parameters<typeof qaApi.update>[1]> }) =>
      qaApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.qa.all }),
  });
}

export function useDeleteQA() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: qaApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.qa.all }),
  });
}

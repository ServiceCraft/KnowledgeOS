import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ragApi, type RagSearchRequest } from '@/api/rag';
import { queryKeys } from '@/lib/queryKeys';

export function useRagIndexStatus() {
  return useQuery({
    queryKey: queryKeys.rag.status,
    queryFn: ragApi.indexStatus,
    refetchInterval: 10_000,
  });
}

export function useRagReindex() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ragApi.reindex,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.rag.status }),
  });
}

export function useRagSearch() {
  return useMutation({
    mutationFn: (data: RagSearchRequest) => ragApi.search(data),
  });
}

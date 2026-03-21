import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { linksApi } from '@/api/links';
import { queryKeys } from '@/lib/queryKeys';
import type { EntityLink } from '@/types';

export function useLinksList(sourceType: string, sourceId: string) {
  return useQuery({
    queryKey: queryKeys.links.list(sourceType, sourceId),
    queryFn: () => linksApi.list(sourceType, sourceId),
    enabled: !!sourceType && !!sourceId,
  });
}

export function useCreateLink(sourceType: string, sourceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<EntityLink>) => linksApi.create(sourceType, sourceId, data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.links.list(sourceType, sourceId) }),
  });
}

export function useDeleteLink(sourceType: string, sourceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (linkId: string) => linksApi.delete(sourceType, sourceId, linkId),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.links.list(sourceType, sourceId) }),
  });
}

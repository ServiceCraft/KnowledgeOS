import { useQuery } from '@tanstack/react-query';
import { syncApi } from '@/api/sync';
import { queryKeys } from '@/lib/queryKeys';

export function useSyncStatus() {
  return useQuery({
    queryKey: queryKeys.sync.status,
    queryFn: () => syncApi.getStatus(),
    refetchInterval: 60_000,
  });
}

import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { searchApi, type SearchParams } from '@/api/search';
import { queryKeys } from '@/lib/queryKeys';

export function useSearch(params: SearchParams) {
  return useQuery({
    queryKey: queryKeys.search.results(params),
    queryFn: () => searchApi.search(params),
    enabled: !!params.query && params.query.length >= 2,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
}

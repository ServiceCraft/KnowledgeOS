import { useQuery } from '@tanstack/react-query';
import { callsApi } from '@/api/calls';
import { queryKeys } from '@/lib/queryKeys';

export function useQACallMentions(qaId: string) {
  return useQuery({
    queryKey: queryKeys.calls.mentionsForQA(qaId),
    queryFn: () => callsApi.listMentionsForQA(qaId, { limit: 100 }),
    enabled: !!qaId,
  });
}

export function useCallDetail(callId: string | null) {
  return useQuery({
    queryKey: callId ? queryKeys.calls.detail(callId) : ['calls', 'noop'],
    queryFn: () => callsApi.getById(callId!),
    enabled: !!callId,
  });
}

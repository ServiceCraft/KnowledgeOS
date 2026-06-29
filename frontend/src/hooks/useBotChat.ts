import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { botChatApi, type CreateChatSessionRequest } from '@/api/botChat';
import { queryKeys } from '@/lib/queryKeys';

export function useChatSessions(page = 1) {
  return useQuery({
    queryKey: queryKeys.botChat.sessions(page),
    queryFn: () => botChatApi.listSessions({ page, limit: 30 }),
  });
}

export function useChatSession(id?: string, refetchInterval: number | false = false) {
  return useQuery({
    queryKey: queryKeys.botChat.detail(id ?? ''),
    queryFn: () => botChatApi.getSession(id!),
    enabled: !!id,
    refetchInterval,
  });
}

export function useCreateChatSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data?: CreateChatSessionRequest) => botChatApi.createSession(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.botChat.all }),
  });
}

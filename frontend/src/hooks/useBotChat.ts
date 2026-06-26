import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { botChatApi, type CreateChatSessionRequest, type SendChatMessageRequest } from '@/api/botChat';
import { queryKeys } from '@/lib/queryKeys';

export function useChatSessions(page = 1) {
  return useQuery({
    queryKey: queryKeys.botChat.sessions(page),
    queryFn: () => botChatApi.listSessions({ page, limit: 30 }),
  });
}

export function useChatSession(id?: string) {
  return useQuery({
    queryKey: queryKeys.botChat.detail(id ?? ''),
    queryFn: () => botChatApi.getSession(id!),
    enabled: !!id,
  });
}

export function useCreateChatSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data?: CreateChatSessionRequest) => botChatApi.createSession(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.botChat.all }),
  });
}

export function useSendChatMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, data }: { sessionId: string; data: SendChatMessageRequest }) =>
      botChatApi.sendMessage(sessionId, data),
    onSuccess: (result) => {
      qc.invalidateQueries({ queryKey: queryKeys.botChat.all });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(result.session.id) });
    },
  });
}

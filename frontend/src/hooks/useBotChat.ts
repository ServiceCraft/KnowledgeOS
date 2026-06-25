import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { botChatApi, type CreateChatSessionRequest, type SendChatMessageRequest } from '@/api/botChat';
import { queryKeys } from '@/lib/queryKeys';

export function useChatSessions() {
  return useQuery({
    queryKey: queryKeys.botChat.sessions,
    queryFn: botChatApi.listSessions,
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
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.botChat.sessions }),
  });
}

export function useSendChatMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, data }: { sessionId: string; data: SendChatMessageRequest }) =>
      botChatApi.sendMessage(sessionId, data),
    onSuccess: (result) => {
      qc.invalidateQueries({ queryKey: queryKeys.botChat.sessions });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(result.session.id) });
    },
  });
}

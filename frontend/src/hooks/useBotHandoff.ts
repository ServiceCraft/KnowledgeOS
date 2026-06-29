import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { botHandoffApi, type HandoffSessionParams, type SendOperatorMessageRequest } from '@/api/botHandoff';
import { queryKeys } from '@/lib/queryKeys';

const QUEUE_POLL_MS = 5000;
const METRICS_POLL_MS = 10000;
const SESSION_POLL_MS = 5000;

export function useHandoffSessions(params?: HandoffSessionParams) {
  return useQuery({
    queryKey: queryKeys.handoff.queue(params),
    queryFn: () => botHandoffApi.listSessions(params),
    refetchInterval: QUEUE_POLL_MS,
  });
}

export function useHandoffMetrics() {
  return useQuery({
    queryKey: queryKeys.handoff.metrics,
    queryFn: botHandoffApi.metrics,
    refetchInterval: METRICS_POLL_MS,
  });
}

export function useHandoffSession(id?: string) {
  return useQuery({
    queryKey: queryKeys.handoff.session(id ?? ''),
    queryFn: () => botHandoffApi.getSession(id!),
    enabled: !!id,
    refetchInterval: id ? SESSION_POLL_MS : false,
  });
}

export function useClaimHandoffSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => botHandoffApi.claim(id),
    onSuccess: (session) => {
      qc.invalidateQueries({ queryKey: queryKeys.handoff.all });
      qc.invalidateQueries({ queryKey: queryKeys.handoff.session(session.id) });
    },
  });
}

export function useEscalateHandoffSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason = 'manual' }: { id: string; reason?: string }) => botHandoffApi.escalate(id, reason),
    onSuccess: (session) => {
      qc.invalidateQueries({ queryKey: queryKeys.handoff.all });
      qc.invalidateQueries({ queryKey: queryKeys.handoff.session(session.id) });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(session.id) });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.all });
    },
  });
}

export function useSendOperatorMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ sessionId, data }: { sessionId: string; data: SendOperatorMessageRequest }) =>
      botHandoffApi.sendMessage(sessionId, data),
    onSuccess: (message) => {
      qc.invalidateQueries({ queryKey: queryKeys.handoff.all });
      qc.invalidateQueries({ queryKey: queryKeys.handoff.session(message.session_id) });
    },
  });
}

export function useReleaseHandoffSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => botHandoffApi.release(id),
    onSuccess: (session) => {
      qc.invalidateQueries({ queryKey: queryKeys.handoff.all });
      qc.invalidateQueries({ queryKey: queryKeys.handoff.session(session.id) });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(session.id) });
    },
  });
}

export function useCloseHandoffSession() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => botHandoffApi.close(id),
    onSuccess: (session) => {
      qc.invalidateQueries({ queryKey: queryKeys.handoff.all });
      qc.invalidateQueries({ queryKey: queryKeys.handoff.session(session.id) });
    },
  });
}

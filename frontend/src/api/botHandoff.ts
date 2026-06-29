import client from './client';
import { normalizeChatMessage, normalizeChatSessionWithMessages } from './chatNormalize';
import type { ChatChannel, ChatMessage, ChatSession, ChatSessionMetrics, ChatSessionWithMessages, ChatState } from '@/types';

interface ListResponse<T> {
  data: T[];
  total?: number;
}

interface DataResponse<T> {
  data: T;
}

export interface HandoffSessionParams {
  page?: number;
  limit?: number;
  state?: ChatState;
  operator_id?: string;
  channel?: ChatChannel;
  active?: boolean;
}

export interface SendOperatorMessageRequest {
  content: string;
}

const base = '/admin/bot/handoff';

export const botHandoffApi = {
  listSessions: (params?: HandoffSessionParams) =>
    client.get<ListResponse<ChatSession>>(`${base}/sessions`, { params }).then((r) => r.data),

  metrics: () =>
    client.get<DataResponse<ChatSessionMetrics>>(`${base}/sessions/metrics`).then((r) => r.data.data),

  getSession: (id: string) =>
    client
      .get<DataResponse<ChatSessionWithMessages>>(`${base}/sessions/${id}`)
      .then((r) => normalizeChatSessionWithMessages(r.data.data)),

  claim: (id: string) =>
    client.post<DataResponse<ChatSession>>(`${base}/sessions/${id}/claim`).then((r) => r.data.data),

  sendMessage: (id: string, data: SendOperatorMessageRequest) =>
    client
      .post<DataResponse<ChatMessage>>(`${base}/sessions/${id}/messages`, data)
      .then((r) => normalizeChatMessage(r.data.data)),

  release: (id: string) =>
    client.post<DataResponse<ChatSession>>(`${base}/sessions/${id}/release`).then((r) => r.data.data),

  close: (id: string) =>
    client.post<DataResponse<ChatSession>>(`${base}/sessions/${id}/close`).then((r) => r.data.data),

  escalate: (id: string, reason = 'manual') =>
    client.post<DataResponse<ChatSession>>(`${base}/sessions/${id}/escalate`, { reason }).then((r) => r.data.data),
};

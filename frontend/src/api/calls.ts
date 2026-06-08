import client from './client';
import type { Call, CallMention, PaginatedResponse } from '@/types';

export const callsApi = {
  getById: (id: string) =>
    client.get(`/calls/${id}`).then((r) => r.data.data as Call),
  listMentionsForQA: (qaId: string, params?: { page?: number; limit?: number }) =>
    client
      .get<PaginatedResponse<CallMention>>(`/qa/${qaId}/mentions`, { params })
      .then((r) => r.data),
};

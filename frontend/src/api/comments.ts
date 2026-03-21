import client from './client';
import type { Comment, PaginatedResponse } from '@/types';

export interface CommentFilter {
  page?: number;
  limit?: number;
}

export const commentsApi = {
  list: (entityType: string, entityId: string, params?: CommentFilter) =>
    client
      .get<PaginatedResponse<Comment>>(`/${entityType}/${entityId}/comments`, { params })
      .then((r) => r.data),
  create: (entityType: string, entityId: string, data: { body: string }) =>
    client
      .post(`/${entityType}/${entityId}/comments`, data)
      .then((r) => r.data.data as Comment),
  update: (entityType: string, entityId: string, commentId: string, data: { body: string }) =>
    client
      .patch(`/${entityType}/${entityId}/comments/${commentId}`, data)
      .then((r) => r.data.data as Comment),
  delete: (entityType: string, entityId: string, commentId: string) =>
    client.delete(`/${entityType}/${entityId}/comments/${commentId}`),
};

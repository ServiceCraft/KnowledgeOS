import client from './client';
import type { Theme, PaginatedResponse, QAPair } from '@/types';

export interface ThemeFilter {
  query?: string;
  page?: number;
  limit?: number;
}

export const themesApi = {
  list: (params?: ThemeFilter) =>
    client.get<PaginatedResponse<Theme>>('/themes', { params }).then((r) => r.data),
  getById: (id: string) =>
    client.get(`/themes/${id}`).then((r) => r.data.data as Theme),
  create: (data: Partial<Theme>) =>
    client.post(`/themes`, data).then((r) => r.data.data as Theme),
  update: (id: string, data: Partial<Theme>) =>
    client.patch(`/themes/${id}`, data).then((r) => r.data.data as Theme),
  delete: (id: string) =>
    client.delete(`/themes/${id}`),
  listQA: (id: string, params?: { page?: number; limit?: number }) =>
    client.get<PaginatedResponse<QAPair>>(`/themes/${id}/qa`, { params }).then((r) => r.data),
};

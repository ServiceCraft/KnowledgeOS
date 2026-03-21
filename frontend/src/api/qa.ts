import client from './client';
import type { QAPair, PaginatedResponse } from '@/types';

export interface QAFilter {
  theme_id?: string;
  is_faq?: boolean;
  query?: string;
  page?: number;
  limit?: number;
}

export const qaApi = {
  list: (params?: QAFilter) =>
    client.get<PaginatedResponse<QAPair>>('/qa', { params }).then((r) => r.data),
  getById: (id: string) =>
    client.get(`/qa/${id}`).then((r) => r.data.data as QAPair),
  create: (data: Partial<QAPair>) =>
    client.post(`/qa`, data).then((r) => r.data.data as QAPair),
  update: (id: string, data: Partial<QAPair>) =>
    client.patch(`/qa/${id}`, data).then((r) => r.data.data as QAPair),
  delete: (id: string) =>
    client.delete(`/qa/${id}`),
};

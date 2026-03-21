import client from './client';
import type { Article, PaginatedResponse } from '@/types';

export interface ArticleFilter {
  query?: string;
  page?: number;
  limit?: number;
}

export const articlesApi = {
  list: (params?: ArticleFilter) =>
    client.get<PaginatedResponse<Article>>('/articles', { params }).then((r) => r.data),
  getById: (id: string) =>
    client.get(`/articles/${id}`).then((r) => r.data.data as Article),
  create: (data: Partial<Article>) =>
    client.post(`/articles`, data).then((r) => r.data.data as Article),
  update: (id: string, data: Partial<Article>) =>
    client.patch(`/articles/${id}`, data).then((r) => r.data.data as Article),
  delete: (id: string) =>
    client.delete(`/articles/${id}`),
};

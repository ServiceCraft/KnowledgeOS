import client from './client';
import type { EntityLink, PaginatedResponse } from '@/types';

export interface LinkFilter {
  page?: number;
  limit?: number;
}

export const linksApi = {
  list: (sourceType: string, sourceId: string, params?: LinkFilter) =>
    client
      .get<PaginatedResponse<EntityLink>>(`/${sourceType}/${sourceId}/links`, { params })
      .then((r) => r.data),
  create: (sourceType: string, sourceId: string, data: Partial<EntityLink>) =>
    client
      .post(`/${sourceType}/${sourceId}/links`, data)
      .then((r) => r.data.data as EntityLink),
  delete: (sourceType: string, sourceId: string, linkId: string) =>
    client.delete(`/${sourceType}/${sourceId}/links/${linkId}`),
};

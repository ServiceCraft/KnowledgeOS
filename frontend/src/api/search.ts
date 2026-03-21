import client from './client';
import type { SearchResponse } from '@/types';

export interface SearchParams {
  query: string;
  types?: string[];
  theme_id?: string;
  page?: number;
  limit?: number;
}

export const searchApi = {
  search: (params: SearchParams) =>
    client.get<SearchResponse>('/search', { params }).then((r) => r.data),
};

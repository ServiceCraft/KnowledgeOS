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
  search: (params: SearchParams) => {
    const { types, ...rest } = params;
    return client
      .get<SearchResponse>('/search', {
        params: {
          ...rest,
          ...(types?.length ? { types: types.join(',') } : {}),
        },
      })
      .then((r) => r.data);
  },
};

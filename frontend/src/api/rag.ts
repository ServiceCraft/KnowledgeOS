import client from './client';
import type { RagIndexStatus, RagSearchResult } from '@/types';

export interface RagSearchRequest {
  query: string;
  types?: string[];
  theme_id?: string;
  vector_top_k?: number;
  hybrid_top_k?: number;
  rewrite?: boolean;
}

const base = '/admin/bot/rag';

export const ragApi = {
  reindex: () =>
    client.post(`${base}/reindex`).then((r) => r.data.data as { status: string }),
  indexStatus: () =>
    client.get(`${base}/index-status`).then((r) => r.data.data as RagIndexStatus),
  search: (data: RagSearchRequest) =>
    client.post(`${base}/search`, data).then((r) => r.data.data as RagSearchResult),
};

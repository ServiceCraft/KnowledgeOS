import type { QAPair, Article, Theme, PricingNode, Call, QAPairCallMention, Company, User, SyncStatus } from './models';

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page?: number;
  limit?: number;
}

export interface SearchResult {
  entity_type: string;
  entity_id: string;
  title: string;
  snippet: string;
  rank: number;
}

export interface SearchResponse {
  data: SearchResult[];
  total: number;
}

export interface ExportData {
  themes: Theme[];
  qa_pairs: QAPair[];
  pricing_nodes: PricingNode[];
  articles: Article[];
  calls?: Call[];
  qa_pair_call_mentions?: QAPairCallMention[];
}

export interface ImportResult {
  imported: number;
  skipped: number;
  errors: string[];
}

export type { QAPair, Article, Theme, PricingNode, Call, QAPairCallMention, Company, User, SyncStatus };

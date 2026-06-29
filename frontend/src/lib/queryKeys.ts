import type { QAFilter } from '@/api/qa';
import type { ThemeFilter } from '@/api/themes';
import type { PricingFilter } from '@/api/pricing';
import type { ArticleFilter } from '@/api/articles';
import type { SearchParams } from '@/api/search';
import type { CompanyFilter } from '@/api/admin';
import type { UserFilter } from '@/api/users';
import { tenantScopeKey } from '@/lib/tenantContext';

const tenant = () => ['tenant', tenantScopeKey()] as const;

export const queryKeys = {
  auth: {
    companies: ['auth', 'companies'] as const,
  },
  qa: {
    get all() { return [...tenant(), 'qa'] as const; },
    list: (filters?: QAFilter) => [...tenant(), 'qa', 'list', filters] as const,
    detail: (id: string) => [...tenant(), 'qa', id] as const,
  },
  themes: {
    get all() { return [...tenant(), 'themes'] as const; },
    list: (filters?: ThemeFilter) => [...tenant(), 'themes', 'list', filters] as const,
    detail: (id: string) => [...tenant(), 'themes', id] as const,
    qa: (id: string, page?: number) => [...tenant(), 'themes', id, 'qa', page] as const,
  },
  pricing: {
    get all() { return [...tenant(), 'pricing'] as const; },
    list: (filters?: PricingFilter) => [...tenant(), 'pricing', 'list', filters] as const,
    detail: (id: string) => [...tenant(), 'pricing', id] as const,
  },
  articles: {
    get all() { return [...tenant(), 'articles'] as const; },
    list: (filters?: ArticleFilter) => [...tenant(), 'articles', 'list', filters] as const,
    detail: (id: string) => [...tenant(), 'articles', id] as const,
  },
  comments: {
    get all() { return [...tenant(), 'comments'] as const; },
    list: (entityType: string, entityId: string) =>
      [...tenant(), 'comments', entityType, entityId] as const,
  },
  links: {
    get all() { return [...tenant(), 'links'] as const; },
    list: (sourceType: string, sourceId: string) =>
      [...tenant(), 'links', sourceType, sourceId] as const,
  },
  search: {
    get all() { return [...tenant(), 'search'] as const; },
    results: (params: SearchParams) => [...tenant(), 'search', params] as const,
  },
  sync: {
    get status() { return [...tenant(), 'sync', 'status'] as const; },
  },
  users: {
    get all() { return [...tenant(), 'users'] as const; },
    list: (filters?: UserFilter) => [...tenant(), 'users', 'list', filters] as const,
    detail: (id: string) => [...tenant(), 'users', id] as const,
  },
  companies: {
    all: ['companies'] as const,
    list: (filters?: CompanyFilter) => ['companies', 'list', filters] as const,
    detail: (id: string) => ['companies', id] as const,
  },
  calls: {
    get all() { return [...tenant(), 'calls'] as const; },
    detail: (id: string) => [...tenant(), 'calls', id] as const,
    mentionsForQA: (qaId: string) => [...tenant(), 'calls', 'mentions', 'qa', qaId] as const,
  },
  botChat: {
    get all() { return [...tenant(), 'botChat'] as const; },
    sessions: (page?: number) => [...tenant(), 'botChat', 'sessions', page] as const,
    detail: (id: string) => [...tenant(), 'botChat', 'sessions', id] as const,
  },
  handoff: {
    get all() { return [...tenant(), 'handoff'] as const; },
    queue: (params?: {
      page?: number;
      limit?: number;
      state?: string;
      operator_id?: string;
      channel?: string;
      active?: boolean;
    }) =>
      [
        ...tenant(),
        'handoff',
        'queue',
        params?.page ?? 1,
        params?.limit ?? 30,
        params?.state ?? '',
        params?.operator_id ?? '',
        params?.channel ?? '',
        params?.active ?? false,
      ] as const,
    session: (id: string) => [...tenant(), 'handoff', 'sessions', id] as const,
    get metrics() { return [...tenant(), 'handoff', 'metrics'] as const; },
    mySessions: (operatorId?: string) => [...tenant(), 'handoff', 'my', operatorId] as const,
  },
  botAdmin: {
    get settings() { return [...tenant(), 'botAdmin', 'settings'] as const; },
    get secrets() { return [...tenant(), 'botAdmin', 'secrets'] as const; },
    get channels() { return [...tenant(), 'botAdmin', 'channels'] as const; },
  },
  rag: {
    get status() { return [...tenant(), 'rag', 'status'] as const; },
  },
};

export function isTenantScopedQueryKey(queryKey: readonly unknown[]): boolean {
  return queryKey[0] === 'tenant';
}

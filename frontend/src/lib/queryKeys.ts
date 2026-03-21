import type { QAFilter } from '@/api/qa';
import type { ThemeFilter } from '@/api/themes';
import type { PricingFilter } from '@/api/pricing';
import type { ArticleFilter } from '@/api/articles';
import type { SearchParams } from '@/api/search';
import type { CompanyFilter } from '@/api/admin';
import type { UserFilter } from '@/api/users';

export const queryKeys = {
  qa: {
    all: ['qa'] as const,
    list: (filters?: QAFilter) => ['qa', 'list', filters] as const,
    detail: (id: string) => ['qa', id] as const,
  },
  themes: {
    all: ['themes'] as const,
    list: (filters?: ThemeFilter) => ['themes', 'list', filters] as const,
    detail: (id: string) => ['themes', id] as const,
  },
  pricing: {
    all: ['pricing'] as const,
    list: (filters?: PricingFilter) => ['pricing', 'list', filters] as const,
    detail: (id: string) => ['pricing', id] as const,
  },
  articles: {
    all: ['articles'] as const,
    list: (filters?: ArticleFilter) => ['articles', 'list', filters] as const,
    detail: (id: string) => ['articles', id] as const,
  },
  comments: {
    all: ['comments'] as const,
    list: (entityType: string, entityId: string) =>
      ['comments', entityType, entityId] as const,
  },
  links: {
    all: ['links'] as const,
    list: (sourceType: string, sourceId: string) =>
      ['links', sourceType, sourceId] as const,
  },
  search: {
    all: ['search'] as const,
    results: (params: SearchParams) => ['search', params] as const,
  },
  sync: {
    status: ['sync', 'status'] as const,
  },
  users: {
    all: ['users'] as const,
    list: (filters?: UserFilter) => ['users', 'list', filters] as const,
    detail: (id: string) => ['users', id] as const,
  },
  companies: {
    all: ['companies'] as const,
    list: (filters?: CompanyFilter) => ['companies', 'list', filters] as const,
    detail: (id: string) => ['companies', id] as const,
  },
};

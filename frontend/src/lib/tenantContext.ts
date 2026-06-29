import type { AuthUser } from '@/types/auth';

export const AUTH_STORAGE_KEY = 'auth-storage';

export interface AuthSnapshot {
  user: AuthUser | null;
  tokens: { access_token: string; refresh_token: string } | null;
  selectedCompanyId: string | null;
  selectedCompanyName: string | null;
}

export function getAuthSnapshot(): AuthSnapshot {
  try {
    const raw = localStorage.getItem(AUTH_STORAGE_KEY);
    if (!raw) {
      return { user: null, tokens: null, selectedCompanyId: null, selectedCompanyName: null };
    }
    const state = JSON.parse(raw)?.state;
    return {
      user: state?.user ?? null,
      tokens: state?.tokens ?? null,
      selectedCompanyId: state?.selectedCompanyId ?? null,
      selectedCompanyName: state?.selectedCompanyName ?? null,
    };
  } catch {
    return { user: null, tokens: null, selectedCompanyId: null, selectedCompanyName: null };
  }
}

export function needsCompanySelection(
  user: AuthUser | null | undefined,
  selectedCompanyId: string | null | undefined
): boolean {
  if (!user) return false;
  const companyIds = user.company_ids ?? [];
  return (
    (user.role === 'superadmin' && !selectedCompanyId) ||
    (user.role !== 'superadmin' && companyIds.length > 1 && !selectedCompanyId)
  );
}

export function isTenantReady(
  user: AuthUser | null | undefined,
  selectedCompanyId: string | null | undefined
): boolean {
  return !needsCompanySelection(user, selectedCompanyId);
}

export function getPostAuthPath(user: AuthUser): string {
  if (user.role === 'superadmin') return '/admin/companies';
  if ((user.company_ids?.length ?? 0) > 1) return '/select-company';
  return '/kb';
}

export function buildAuthHeaders(): Record<string, string> {
  const { tokens, selectedCompanyId } = getAuthSnapshot();
  const headers: Record<string, string> = {};
  if (tokens?.access_token) {
    headers.Authorization = `Bearer ${tokens.access_token}`;
  }
  if (selectedCompanyId) {
    headers['X-Company-ID'] = selectedCompanyId;
  }
  return headers;
}

export function tenantScopeKey(): string {
  return getAuthSnapshot().selectedCompanyId || 'no-company';
}

export function isTenantScopedApiPath(url?: string): boolean {
  if (!url) return false;
  const path = url.startsWith('/api/v1') ? url.slice('/api/v1'.length) || '/' : url;
  if (path.startsWith('/auth/')) return false;
  if (path === '/admin/companies' || path.startsWith('/admin/companies/')) return false;
  return true;
}

export function tenantContextError(url?: string): Error | null {
  const { user, selectedCompanyId } = getAuthSnapshot();
  if (!isTenantScopedApiPath(url)) return null;
  if (!needsCompanySelection(user, selectedCompanyId)) return null;
  return new Error('company selection required');
}

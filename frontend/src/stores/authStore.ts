import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthUser } from '@/types/auth';

interface AuthState {
  user: AuthUser | null;
  tokens: { access_token: string; refresh_token: string } | null;
  isAuthenticated: boolean;
  selectedCompanyId: string | null;
  selectedCompanyName: string | null;
  _hasHydrated: boolean;
  login: (user: AuthUser, tokens: { access_token: string; refresh_token: string }) => void;
  logout: () => void;
  setTokens: (tokens: { access_token: string; refresh_token: string }) => void;
  setUser: (user: AuthUser) => void;
  setSelectedCompany: (companyId: string | null, companyName?: string | null) => void;
  setHasHydrated: (value: boolean) => void;
}

function initialCompanyId(user: AuthUser): string | null {
  if (user.role === 'superadmin') return null;
  if (user.company_ids?.length === 1) return user.company_ids[0];
  return null;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      tokens: null,
      isAuthenticated: false,
      selectedCompanyId: null,
      selectedCompanyName: null,
      _hasHydrated: false,
      login: (user, tokens) =>
        set({
          user,
          tokens,
          isAuthenticated: true,
          selectedCompanyId: initialCompanyId(user),
          selectedCompanyName: null,
        }),
      logout: () =>
        set({
          user: null,
          tokens: null,
          isAuthenticated: false,
          selectedCompanyId: null,
          selectedCompanyName: null,
        }),
      setTokens: (tokens) => set({ tokens }),
      setUser: (user) => set({ user }),
      setSelectedCompany: (companyId, companyName = null) =>
        set({ selectedCompanyId: companyId, selectedCompanyName: companyName }),
      setHasHydrated: (value) => set({ _hasHydrated: value }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        tokens: state.tokens,
        isAuthenticated: state.isAuthenticated,
        selectedCompanyId: state.selectedCompanyId,
        selectedCompanyName: state.selectedCompanyName,
      }),
    }
  )
);

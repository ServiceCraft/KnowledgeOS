import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthUser } from '@/types/auth';

interface AuthState {
  user: AuthUser | null;
  tokens: { access_token: string; refresh_token: string } | null;
  isAuthenticated: boolean;
  selectedCompanyId: string | null;
  selectedCompanyName: string | null;
  login: (user: AuthUser, tokens: { access_token: string; refresh_token: string }) => void;
  logout: () => void;
  setTokens: (tokens: { access_token: string; refresh_token: string }) => void;
  setUser: (user: AuthUser) => void;
  setSelectedCompany: (companyId: string | null, companyName?: string | null) => void;
  setSelectedCompanyId: (companyId: string | null) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      tokens: null,
      isAuthenticated: false,
      selectedCompanyId: null,
      selectedCompanyName: null,
      login: (user, tokens) =>
        set({
          user,
          tokens,
          isAuthenticated: true,
          selectedCompanyId: user.role === 'superadmin' ? null : user.company_id ?? null,
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
      setSelectedCompanyId: (companyId) => set({ selectedCompanyId: companyId, selectedCompanyName: null }),
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

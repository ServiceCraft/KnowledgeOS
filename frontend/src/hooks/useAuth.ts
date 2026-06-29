import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { authApi } from '@/api/auth';
import { getPostAuthPath } from '@/lib/tenantContext';
import { useAuthStore } from '@/stores/authStore';
import type { LoginRequest } from '@/types/auth';

export function useLogin() {
  const login = useAuthStore((s) => s.login);
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (data: LoginRequest) => authApi.login(data),
    onSuccess: (data) => {
      login(data.user, {
        access_token: data.access_token,
        refresh_token: data.refresh_token,
      });
      navigate(getPostAuthPath(data.user));
    },
  });
}

export function useLogout() {
  const logout = useAuthStore((s) => s.logout);
  const tokens = useAuthStore((s) => s.tokens);
  const navigate = useNavigate();
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () => authApi.logout(tokens?.refresh_token ?? ''),
    onSettled: () => {
      logout();
      qc.clear();
      navigate('/login');
    },
  });
}

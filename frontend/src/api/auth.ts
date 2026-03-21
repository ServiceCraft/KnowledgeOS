import client from './client';
import type { LoginRequest, LoginResponse } from '@/types/auth';

export const authApi = {
  login: (data: LoginRequest) =>
    client.post<{ data: LoginResponse }>('/auth/login', data).then((r) => r.data.data),
  refresh: (refreshToken: string) =>
    client.post<{ data: LoginResponse }>('/auth/refresh', { refresh_token: refreshToken }).then((r) => r.data.data),
  logout: () =>
    client.post('/auth/logout'),
};

import client from './client';
import type { LoginRequest, LoginResponse } from '@/types/auth';
import type { Company } from '@/types';

export const authApi = {
  login: (data: LoginRequest) =>
    client.post<{ data: LoginResponse }>('/auth/login', data).then((r) => r.data.data),
  refresh: (refreshToken: string) =>
    client.post<{ data: LoginResponse }>('/auth/refresh', { refresh_token: refreshToken }).then((r) => r.data.data),
  logout: (refreshToken: string) =>
    client.post('/auth/logout', { refresh_token: refreshToken }),
  listCompanies: () =>
    client.get<{ data: Company[] }>('/auth/companies').then((r) => r.data.data),
};

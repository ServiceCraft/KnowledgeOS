import client from './client';
import type { User, PaginatedResponse, Role } from '@/types';

export interface UserFilter {
  page?: number;
  limit?: number;
  q?: string;
  role?: Role;
  sort?: string; // 'created_at' | '-created_at'
}

export interface CreateUserRequest {
  email: string;
  password: string;
  role: Role;
  is_active?: boolean;
}

export interface UpdateUserRequest {
  email?: string;
  password?: string;
  role?: Role;
  is_active?: boolean;
}

export const usersApi = {
  list: (params?: UserFilter) =>
    client.get<PaginatedResponse<User>>('/admin/users', { params }).then((r) => r.data),
  create: (data: CreateUserRequest) =>
    client.post('/admin/users', data).then((r) => r.data.data as User),
  update: (id: string, data: UpdateUserRequest) =>
    client.patch(`/admin/users/${id}`, data).then((r) => r.data.data as User),
  delete: (id: string) =>
    client.delete(`/admin/users/${id}`),
};

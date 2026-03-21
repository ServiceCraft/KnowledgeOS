import client from './client';
import type { User, PaginatedResponse, Role } from '@/types';

export interface UserFilter {
  page?: number;
  limit?: number;
}

export interface CreateUserRequest {
  email: string;
  password: string;
  role: Role;
}

export interface UpdateUserRequest {
  email?: string;
  password?: string;
  role?: Role;
}

export const usersApi = {
  list: (params?: UserFilter) =>
    client.get<PaginatedResponse<User>>('/users', { params }).then((r) => r.data),
  getById: (id: string) =>
    client.get<User>(`/users/${id}`).then((r) => r.data),
  create: (data: CreateUserRequest) =>
    client.post<User>('/users', data).then((r) => r.data),
  update: (id: string, data: UpdateUserRequest) =>
    client.patch<User>(`/users/${id}`, data).then((r) => r.data),
  delete: (id: string) =>
    client.delete(`/users/${id}`),
};

import client from './client';
import type { Company, PaginatedResponse, User } from '@/types';

export interface CompanyFilter {
  page?: number;
  limit?: number;
}

export interface CreateCompanyRequest {
  name: string;
  tier: string;
}

export interface CreateCompanyAdminRequest {
  email: string;
  password: string;
}

export interface CreateCompanyWithAdminRequest extends CreateCompanyRequest {
  admin_email: string;
  admin_password: string;
}

export const adminApi = {
  listCompanies: (params?: CompanyFilter) =>
    client.get<PaginatedResponse<Company>>('/admin/companies', { params }).then((r) => r.data),
  getCompany: (id: string) =>
    client.get(`/admin/companies/${id}`).then((r) => r.data.data as Company),
  createCompany: (data: CreateCompanyRequest) =>
    client.post('/admin/companies', data).then((r) => r.data.data as Company),
  createCompanyAdmin: (companyId: string, data: CreateCompanyAdminRequest) =>
    client.post(`/admin/companies/${companyId}/admin`, data).then((r) => r.data.data as User),
  updateCompany: (id: string, data: Partial<Company>) =>
    client.patch(`/admin/companies/${id}`, data).then((r) => r.data.data as Company),
  deleteCompany: (id: string) =>
    client.delete(`/admin/companies/${id}`),
};

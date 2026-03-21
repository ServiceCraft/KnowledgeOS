import client from './client';
import type { PricingNode, PaginatedResponse } from '@/types';

export interface PricingFilter {
  parent_id?: string;
  node_type?: string;
  page?: number;
  limit?: number;
}

export const pricingApi = {
  list: (params?: PricingFilter) =>
    client.get<PaginatedResponse<PricingNode>>('/pricing', { params }).then((r) => r.data),
  getById: (id: string) =>
    client.get(`/pricing/${id}`).then((r) => r.data.data as PricingNode),
  create: (data: Partial<PricingNode>) =>
    client.post(`/pricing`, data).then((r) => r.data.data as PricingNode),
  update: (id: string, data: Partial<PricingNode>) =>
    client.patch(`/pricing/${id}`, data).then((r) => r.data.data as PricingNode),
  delete: (id: string) =>
    client.delete(`/pricing/${id}`),
};

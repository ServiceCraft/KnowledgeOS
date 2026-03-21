export interface Company {
  id: string;
  created_at: string;
  updated_at: string;
  name: string;
  tier: string;
}

export interface User {
  id: string;
  created_at: string;
  updated_at: string;
  company_id?: string;
  email: string;
  role: Role;
}

export type Role = 'superadmin' | 'admin' | 'editor' | 'viewer';

export interface Theme {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  name: string;
  description: string;
}

export interface QAPair {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  theme_id?: string;
  question: string;
  answer: string;
  is_faq: boolean;
  is_locked: boolean;
}

export interface PricingNode {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  parent_id?: string;
  node_type: string;
  name: string;
  price?: number;
}

export interface Article {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  title: string;
  body: string;
}

export interface Comment {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  entity_type: string;
  entity_id: string;
  body: string;
  author_id?: string;
}

export interface EntityLink {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  source_type: string;
  source_id: string;
  target_type?: string;
  target_id?: string;
  url?: string;
  label?: string;
}

export interface SyncStatus {
  company_id: string;
  last_sync_at?: string;
  last_sync_result?: string;
  last_error?: string;
  subscription_active: boolean;
}

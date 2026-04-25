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

export type AIStatus = 'pending' | 'accepted' | 'rejected' | 'edited';

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
  frequency: number;
  ai_answer?: string;
  ai_status?: AIStatus;
  ai_reviewed_by?: string;
  ai_reviewed_at?: string;
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

export interface Call {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  external_id?: string;
  title: string;
  occurred_at?: string;
  duration_sec?: number;
  audio_url?: string;
  transcript: string;
}

export interface CallMention {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  sync_version: number;
  sync_origin: string;
  created_by?: string;
  updated_by?: string;
  deleted_at?: string;
  qa_pair_id: string;
  call_id: string;
  snippet: string;
  start_offset: number;
  end_offset: number;
  start_sec?: number;
  end_sec?: number;
  confidence?: number;
  call_title: string;
  call_occurred_at?: string;
}

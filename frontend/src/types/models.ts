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
  company_ids?: string[];
  email: string;
  role: Role;
  is_active: boolean;
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

export interface QAPairCallMention {
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
  start_offset?: number;
  end_offset?: number;
  start_sec?: number;
  end_sec?: number;
  confidence?: number;
}

export interface CallMention extends QAPairCallMention {
  call_title: string;
  call_occurred_at?: string;
}

export type ChatRole = 'user' | 'assistant' | 'tool' | 'operator';
export type ChatChannel = 'playground' | 'api' | 'telegram' | 'max' | 'vk';
export type ChatState = 'bot' | 'waiting_operator' | 'operator' | 'closed';
export type GuardrailAction = 'answer' | 'refuse' | 'escalate';

export interface ChatSession {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  channel: ChatChannel;
  external_chat_id?: string;
  state: ChatState;
  operator_id?: string;
  title: string;
  last_message_at?: string;
}

export interface ChatSource {
  source_id: string;
  entity_type: string;
  entity_id: string;
  chunk_idx: number;
  title: string;
  content: string;
  snippet?: string;
  score: number;
}

export interface ChatToolCall {
  id?: string;
  name: string;
  arguments?: unknown;
}

export interface ChatMessage {
  id: string;
  created_at: string;
  updated_at: string;
  company_id: string;
  session_id: string;
  role: ChatRole;
  content: string;
  tool_call_id?: string;
  tool_calls: ChatToolCall[];
  sources: ChatSource[];
  guardrail_action?: GuardrailAction;
  confidence_score?: number | null;
  refusal_reason?: string;
  cited_source_ids?: string[];
  tokens_prompt: number;
  tokens_completion: number;
}

export interface ChatSessionWithMessages {
  session: ChatSession;
  messages: ChatMessage[];
}

export interface ChatSessionMetrics {
  total: number;
  bot: number;
  waiting_operator: number;
  operator: number;
  closed: number;
  by_channel: Partial<Record<ChatChannel, number>>;
}

export interface ChatExchange {
  session: ChatSession;
  user: ChatMessage;
  message: ChatMessage;
  sources: ChatSource[];
}

export type BotProvider = 'yandex';
export type BotModelTier = 'lite' | 'pro';
export type SecretKind = 'llm' | 'telegram' | 'max' | 'vk' | 'yclients';

export interface BotSettings {
  company_id: string;
  enabled: boolean;
  provider: BotProvider;
  model_tier: BotModelTier;
  model?: string;
  temperature: number;
  max_tokens: number;
  persona_name: string;
  persona_tone: string;
  persona_rules: string;
  enabled_modules: Record<string, unknown>;
  min_retrieval_score: number;
  min_confidence: number;
  allowed_theme_ids: string[];
  escalate_on_low_confidence: boolean;
  require_citations: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateBotSettingsRequest {
  enabled?: boolean;
  provider?: BotProvider;
  model_tier?: BotModelTier;
  model?: string;
  temperature?: number;
  max_tokens?: number;
  persona_name?: string;
  persona_tone?: string;
  persona_rules?: string;
  enabled_modules?: Record<string, unknown>;
  min_retrieval_score?: number;
  min_confidence?: number;
  allowed_theme_ids?: string[];
  escalate_on_low_confidence?: boolean;
  require_citations?: boolean;
}

export interface TenantSecretStatus {
  kind: SecretKind;
  is_set: boolean;
  metadata: Record<string, unknown>;
  updated_at?: string;
}

export interface EditableTenantSecret {
  kind: SecretKind;
  is_set: boolean;
  value: string;
  metadata: Record<string, unknown>;
}

export interface ChannelStatus {
  channel: Extract<ChatChannel, 'telegram' | 'max' | 'vk'>;
  secret_kind: Extract<SecretKind, 'telegram' | 'max' | 'vk'>;
  configured: boolean;
  enabled: boolean;
  bot_enabled: boolean;
  webhook_url: string;
  webhook_configured: boolean;
  webhook_error?: string;
  metadata: Record<string, unknown>;
  updated_at?: string;
}

export interface RagIndexStatus {
  embeddings: number;
  jobs: Record<string, number>;
}

export interface RagCandidate {
  source_id: string;
  entity_type: string;
  entity_id: string;
  chunk_idx: number;
  theme_id?: string;
  title: string;
  content: string;
  snippet?: string;
  score: number;
}

export interface RagSearchResult {
  query: string;
  rewritten_query?: string;
  results: RagCandidate[];
}


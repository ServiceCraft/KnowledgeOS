import client, { tenantContextError } from './client';
import { normalizeChatExchange, normalizeChatMessage, normalizeChatSessionWithMessages } from './chatNormalize';
import type { ChatExchange, ChatMessage, ChatSession, ChatSessionWithMessages, ChatSource } from '@/types';

export interface CreateChatSessionRequest {
  channel?: 'playground' | 'api';
  title?: string;
}

export interface SendChatMessageRequest {
  content: string;
}

export interface ChatStreamEvent {
  type: 'sources' | 'delta' | 'usage' | 'message' | 'error' | 'done';
  delta?: string;
  message?: ChatMessage;
  sources?: ChatSource[];
  usage?: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
  error?: string;
}

interface ListResponse<T> {
  data: T[];
  total?: number;
}

interface DataResponse<T> {
  data: T;
}

const base = '/admin/bot/chat';

export const botChatApi = {
  createSession: (data: CreateChatSessionRequest = {}) =>
    client.post<DataResponse<ChatSession>>(`${base}/sessions`, data).then((r) => r.data.data),

  listSessions: (params?: { page?: number; limit?: number }) =>
    client.get<ListResponse<ChatSession>>(`${base}/sessions`, { params }).then((r) => r.data),

  getSession: (id: string) =>
    client
      .get<DataResponse<ChatSessionWithMessages>>(`${base}/sessions/${id}`)
      .then((r) => normalizeChatSessionWithMessages(r.data.data)),

  sendMessage: (sessionId: string, data: SendChatMessageRequest) =>
    client
      .post<DataResponse<ChatExchange>>(`${base}/sessions/${sessionId}/messages`, data)
      .then((r) => normalizeChatExchange(r.data.data)),

  streamMessage: async (
    sessionId: string,
    data: SendChatMessageRequest,
    onEvent: (event: ChatStreamEvent) => void,
    signal?: AbortSignal
  ) => {
    const url = `${base}/sessions/${sessionId}/messages/stream`;
    const tenantError = tenantContextError(url);
    if (tenantError) {
      throw tenantError;
    }
    const response = await fetch(`/api/v1${url}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders(),
      },
      body: JSON.stringify(data),
      signal,
    });
    if (!response.ok || !response.body) {
      const text = await response.text();
      throw new Error(errorFromBody(text) || `HTTP ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n\n');
      buffer = parts.pop() ?? '';
      for (const part of parts) {
        const event = parseSSE(part);
        if (event) onEvent(normalizeChatStreamEvent(event));
      }
    }
    if (buffer.trim()) {
      const event = parseSSE(buffer);
      if (event) onEvent(normalizeChatStreamEvent(event));
    }
  },
};

function normalizeChatStreamEvent(event: ChatStreamEvent): ChatStreamEvent {
  return {
    ...event,
    message: event.message ? normalizeChatMessage(event.message) : undefined,
    sources: Array.isArray(event.sources) ? event.sources : [],
  };
}

function authHeaders(): Record<string, string> {
  try {
    const raw = localStorage.getItem('auth-storage');
    const state = raw ? JSON.parse(raw)?.state : null;
    const token = state?.tokens?.access_token;
    const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
    if (state?.user?.role === 'superadmin' && state?.selectedCompanyId) {
      headers['X-Company-ID'] = state.selectedCompanyId;
    }
    return headers;
  } catch {
    return {};
  }
}

function parseSSE(raw: string): ChatStreamEvent | null {
  const dataLines = raw
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart());
  if (dataLines.length === 0) return null;
  try {
    return JSON.parse(dataLines.join('\n')) as ChatStreamEvent;
  } catch {
    return null;
  }
}

function errorFromBody(text: string) {
  try {
    return JSON.parse(text)?.error;
  } catch {
    return text;
  }
}

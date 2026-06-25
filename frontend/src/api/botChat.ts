import client from './client';
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

  listSessions: () =>
    client.get<ListResponse<ChatSession>>(`${base}/sessions`).then((r) => r.data),

  getSession: (id: string) =>
    client.get<DataResponse<ChatSessionWithMessages>>(`${base}/sessions/${id}`).then((r) => r.data.data),

  sendMessage: (sessionId: string, data: SendChatMessageRequest) =>
    client.post<DataResponse<ChatExchange>>(`${base}/sessions/${sessionId}/messages`, data).then((r) => r.data.data),

  streamMessage: async (
    sessionId: string,
    data: SendChatMessageRequest,
    onEvent: (event: ChatStreamEvent) => void,
    signal?: AbortSignal
  ) => {
    const response = await fetch(`/api/v1${base}/sessions/${sessionId}/messages/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeader(),
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
        if (event) onEvent(event);
      }
    }
    if (buffer.trim()) {
      const event = parseSSE(buffer);
      if (event) onEvent(event);
    }
  },
};

function authHeader(): Record<string, string> {
  try {
    const raw = localStorage.getItem('auth-storage');
    const token = raw ? JSON.parse(raw)?.state?.tokens?.access_token : null;
    return token ? { Authorization: `Bearer ${token}` } : {};
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

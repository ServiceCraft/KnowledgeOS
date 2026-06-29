import type { ChatExchange, ChatMessage, ChatSessionWithMessages } from '@/types';

export function normalizeChatMessage(message: ChatMessage): ChatMessage {
  return {
    ...message,
    tool_calls: Array.isArray(message.tool_calls) ? message.tool_calls : [],
    sources: Array.isArray(message.sources) ? message.sources : [],
    cited_source_ids: Array.isArray(message.cited_source_ids) ? message.cited_source_ids : [],
  };
}

export function normalizeChatSessionWithMessages(data: ChatSessionWithMessages): ChatSessionWithMessages {
  return {
    ...data,
    messages: Array.isArray(data.messages) ? data.messages.map(normalizeChatMessage) : [],
  };
}

export function normalizeChatExchange(data: ChatExchange): ChatExchange {
  return {
    ...data,
    user: normalizeChatMessage(data.user),
    message: normalizeChatMessage(data.message),
    sources: Array.isArray(data.sources) ? data.sources : [],
  };
}

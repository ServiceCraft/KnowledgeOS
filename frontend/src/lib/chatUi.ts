import type { ChatMessage, ChatSource, ChatState } from '@/types';

export const STATE_LABELS: Record<ChatState, string> = {
  bot: 'Бот',
  waiting_operator: 'Ожидает',
  operator: 'У оператора',
  closed: 'Закрыт',
};

export function collectSources(messages: ChatMessage[]): ChatSource[] {
  const seen = new Set<string>();
  const out: ChatSource[] = [];
  for (const message of messages) {
    for (const source of message.sources ?? []) {
      if (seen.has(source.source_id)) continue;
      seen.add(source.source_id);
      out.push(source);
    }
  }
  return out;
}

export function guardrailReasonLabel(reason: string): string {
  switch (reason) {
    case 'explicit_request':
      return 'клиент попросил оператора';
    case 'no_context':
      return 'нет данных в базе знаний';
    case 'low_confidence':
      return 'низкая уверенность';
    case 'missing_citation':
      return 'нет подтверждающего источника';
    case 'fabricated_citation':
      return 'выдуманный источник';
    case 'prompt_leak':
      return 'попытка раскрыть инструкции';
    default:
      return reason;
  }
}

export function mapChatStreamError(raw?: string): string {
  const text = (raw ?? '').toLowerCase();
  if (text.includes('chat session is closed') || text.includes('closed')) {
    return 'Диалог закрыт. Создайте новую сессию.';
  }
  if (text.includes('not handled by bot') || text.includes('handled by operator')) {
    return 'Диалог у оператора. Откройте очередь handoff.';
  }
  return raw ?? 'Ошибка генерации ответа';
}

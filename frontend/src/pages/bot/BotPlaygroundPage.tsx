import { useEffect, useMemo, useRef, useState } from 'react';
import { Bot, Loader2, MessageSquare, Plus, Send, ShieldAlert, ShieldCheck, Square, User } from 'lucide-react';
import { botChatApi, type ChatStreamEvent } from '@/api/botChat';
import { useChatSession, useChatSessions, useCreateChatSession } from '@/hooks/useBotChat';
import type { ChatMessage, ChatSource } from '@/types';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Textarea } from '@/components/ui/textarea';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { cn } from '@/lib/utils';
import { queryKeys } from '@/lib/queryKeys';
import { useQueryClient } from '@tanstack/react-query';

export function BotPlaygroundPage() {
  const qc = useQueryClient();
  const sessionsQuery = useChatSessions();
  const createSession = useCreateChatSession();
  const [selectedSessionId, setSelectedSessionId] = useState<string>();
  const [draft, setDraft] = useState('');
  const [localMessages, setLocalMessages] = useState<ChatMessage[]>([]);
  const [streamingText, setStreamingText] = useState('');
  const [streamSources, setStreamSources] = useState<ChatSource[]>([]);
  const [error, setError] = useState<string>();
  const [isStreaming, setIsStreaming] = useState(false);
  const [lastUsage, setLastUsage] = useState<ChatStreamEvent['usage']>();
  const abortRef = useRef<AbortController | undefined>(undefined);

  const sessionQuery = useChatSession(selectedSessionId);
  const sessions = useMemo(() => sessionsQuery.data?.data ?? [], [sessionsQuery.data?.data]);

  useEffect(() => {
    if (!selectedSessionId && sessions.length > 0) {
      setSelectedSessionId(sessions[0].id);
    }
  }, [selectedSessionId, sessions]);

  useEffect(() => {
    if (sessionQuery.data?.messages) {
      setLocalMessages(sessionQuery.data.messages);
    }
  }, [sessionQuery.data?.messages]);

  const visibleSources = useMemo(() => {
    if (streamSources.length > 0) return streamSources;
    const lastAssistant = [...localMessages].reverse().find((m) => m.role === 'assistant');
    return lastAssistant?.sources ?? [];
  }, [localMessages, streamSources]);

  async function ensureSession() {
    if (selectedSessionId) return selectedSessionId;
    const created = await createSession.mutateAsync({ channel: 'playground' });
    setSelectedSessionId(created.id);
    return created.id;
  }

  async function startNewSession() {
    setError(undefined);
    const created = await createSession.mutateAsync({ channel: 'playground' });
    setSelectedSessionId(created.id);
    setLocalMessages([]);
    setStreamingText('');
    setStreamSources([]);
  }

  function stopStreaming() {
    abortRef.current?.abort();
  }

  async function sendMessage() {
    const content = draft.trim();
    if (!content || isStreaming) return;
    setDraft('');
    setError(undefined);
    setStreamingText('');
    setStreamSources([]);
    setLastUsage(undefined);
    const sessionId = await ensureSession();
    const optimistic: ChatMessage = {
      id: `local-${Date.now()}`,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      company_id: '',
      session_id: sessionId,
      role: 'user',
      content,
      tool_calls: [],
      sources: [],
      tokens_prompt: 0,
      tokens_completion: 0,
    };
    setLocalMessages((items) => [...items, optimistic]);
    setIsStreaming(true);
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      await botChatApi.streamMessage(
        sessionId,
        { content },
        (event) => {
          if (event.type === 'sources' && event.sources) {
            setStreamSources(event.sources);
          }
          if (event.type === 'delta' && event.delta) {
            setStreamingText((current) => current + event.delta);
          }
          if (event.type === 'usage' && event.usage) {
            setLastUsage(event.usage);
          }
          if (event.type === 'message' && event.message) {
            setLocalMessages((items) => [...items.filter((item) => item.id !== event.message!.id), event.message!]);
            setStreamingText('');
            setStreamSources(event.message.sources ?? []);
          }
          if (event.type === 'error') {
            setError(event.error ?? 'Ошибка генерации ответа');
          }
        },
        controller.signal
      );
      qc.invalidateQueries({ queryKey: queryKeys.botChat.sessions });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(sessionId) });
    } catch (err) {
      if (controller.signal.aborted) {
        setError(undefined);
      } else {
        setError(err instanceof Error ? err.message : 'Ошибка генерации ответа');
      }
    } finally {
      setIsStreaming(false);
      abortRef.current = undefined;
    }
  }

  return (
    <div className="flex h-[calc(100vh-7rem)] gap-4">
      <Card className="w-72 shrink-0">
        <CardHeader>
          <CardTitle>Плейграунд</CardTitle>
          <CardDescription>Внутренние тестовые диалоги</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button className="w-full" onClick={startNewSession} disabled={createSession.isPending}>
            {createSession.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            Новая сессия
          </Button>
          <ScrollArea className="h-[calc(100vh-17rem)]">
            <div className="space-y-2 pr-2">
              {sessions.map((session) => (
                <button
                  key={session.id}
                  type="button"
                  onClick={() => setSelectedSessionId(session.id)}
                  className={cn(
                    'w-full rounded-lg border p-3 text-left text-sm transition hover:bg-muted',
                    selectedSessionId === session.id && 'border-primary bg-muted'
                  )}
                >
                  <div className="flex items-center gap-2">
                    <MessageSquare className="h-4 w-4 shrink-0" />
                    <span className="truncate font-medium">{session.title || 'Новая сессия'}</span>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {session.last_message_at ? new Date(session.last_message_at).toLocaleString() : 'Без сообщений'}
                  </p>
                </button>
              ))}
              {sessions.length === 0 && (
                <p className="px-1 text-sm text-muted-foreground">Создайте первую сессию для теста бота.</p>
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      <Card className="min-w-0 flex-1">
        <CardHeader>
          <CardTitle>Чат с ботом</CardTitle>
          <CardDescription>Ответ строится через RAG и сохраняется в истории с источниками.</CardDescription>
        </CardHeader>
        <CardContent className="flex h-full min-h-0 flex-col gap-3">
          {error && (
            <Alert variant="destructive">
              <AlertTitle>Бот не ответил</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <ScrollArea className="min-h-0 flex-1 rounded-lg border bg-muted/20 p-4">
            <div className="space-y-4 pr-3">
              {localMessages.map((message) => (
                <MessageBubble key={message.id} message={message} />
              ))}
              {streamingText && (
                <div className="flex items-start gap-3">
                  <Bot className="mt-1 h-5 w-5 text-primary" />
                  <div className="max-w-[80%] rounded-lg bg-card p-3 text-sm ring-1 ring-border">
                    <p className="whitespace-pre-wrap">{streamingText}</p>
                  </div>
                </div>
              )}
              {localMessages.length === 0 && !streamingText && (
                <div className="flex h-48 flex-col items-center justify-center text-center text-muted-foreground">
                  <Bot className="mb-3 h-8 w-8" />
                  <p className="text-sm">Задайте вопрос по базе знаний, статьям или прайсу.</p>
                </div>
              )}
            </div>
          </ScrollArea>

          {visibleSources.length > 0 && (
            <div className="rounded-lg border p-3">
              <div className="mb-2 flex items-center justify-between text-sm font-medium">
                <span>Источники ответа</span>
                {lastUsage && (
                  <span className="text-xs font-normal text-muted-foreground">
                    Токены: {lastUsage.prompt_tokens}+{lastUsage.completion_tokens}={lastUsage.total_tokens}
                  </span>
                )}
              </div>
              <div className="space-y-2">
                {visibleSources.map((source) => (
                  <SourceRow key={source.source_id} source={source} />
                ))}
              </div>
            </div>
          )}

          <div className="flex items-end gap-2">
            <Textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault();
                  sendMessage();
                }
              }}
              placeholder="Введите сообщение. Ctrl/⌘ + Enter отправляет."
              disabled={isStreaming}
              className="min-h-20"
            />
            {isStreaming ? (
              <Button variant="outline" onClick={stopStreaming}>
                <Square className="h-4 w-4" />
                Стоп
              </Button>
            ) : (
              <Button onClick={sendMessage} disabled={!draft.trim()}>
                <Send className="h-4 w-4" />
                Отправить
              </Button>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user';
  if (message.role === 'tool') return null;
  const Icon = isUser ? User : Bot;
  return (
    <div className={cn('flex items-start gap-3', isUser && 'flex-row-reverse')}>
      <Icon className={cn('mt-1 h-5 w-5', isUser ? 'text-muted-foreground' : 'text-primary')} />
      <div className={cn('max-w-[80%] rounded-lg p-3 text-sm ring-1', isUser ? 'bg-primary text-primary-foreground ring-primary' : 'bg-card ring-border')}>
        <p className="whitespace-pre-wrap">{message.content}</p>
        {!isUser && <GuardrailVerdict message={message} />}
      </div>
    </div>
  );
}

function GuardrailVerdict({ message }: { message: ChatMessage }) {
  const action = message.guardrail_action;
  if (!action || action === 'answer') {
    if (typeof message.confidence_score !== 'number') return null;
    return (
      <div className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
        <ShieldCheck className="h-3.5 w-3.5 text-emerald-600" />
        <span>Уверенность {Math.round(message.confidence_score * 100)}%</span>
      </div>
    );
  }
  const label = action === 'escalate' ? 'Эскалация оператору' : 'Отказ';
  return (
    <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
      <Badge variant={action === 'escalate' ? 'default' : 'secondary'} className="gap-1">
        <ShieldAlert className="h-3.5 w-3.5" />
        {label}
      </Badge>
      {message.refusal_reason && <span className="text-muted-foreground">{guardrailReasonLabel(message.refusal_reason)}</span>}
    </div>
  );
}

function guardrailReasonLabel(reason: string): string {
  switch (reason) {
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

function SourceRow({ source }: { source: ChatSource }) {
  return (
    <div className="rounded-md border bg-card/40 p-2 text-xs">
      <div className="flex items-center gap-2">
        <Badge variant="outline" className="shrink-0 uppercase">
          {source.entity_type}
        </Badge>
        <span className="truncate font-medium">{source.title || source.source_id}</span>
        {typeof source.score === 'number' && source.score > 0 && (
          <span className="ml-auto shrink-0 text-muted-foreground">{source.score.toFixed(3)}</span>
        )}
      </div>
      {source.snippet && <p className="mt-1 line-clamp-2 text-muted-foreground">{source.snippet}</p>}
    </div>
  );
}

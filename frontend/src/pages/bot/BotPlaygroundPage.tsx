import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Bot,
  Headphones,
  Loader2,
  MessageSquare,
  Plus,
  Send,
  ShieldAlert,
  ShieldCheck,
  Square,
  Trash2,
  User,
} from 'lucide-react';
import { botChatApi } from '@/api/botChat';
import {
  useChatSession,
  useChatSessions,
  useCreateChatSession,
  useDeleteChatSession,
} from '@/hooks/useBotChat';
import { useEscalateHandoffSession } from '@/hooks/useBotHandoff';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import type { ChatMessage, ChatSession } from '@/types';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { formatMessageBody } from '@/lib/chatSources';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/queryKeys';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { cn } from '@/lib/utils';
import { guardrailReasonLabel, mapChatStreamError, STATE_LABELS } from '@/lib/chatUi';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ChatAnswerContent } from '@/components/chat/ChatCitations';
import { BotTypingIndicator } from '@/components/chat/BotTypingIndicator';
import { ChatComposer } from '@/components/chat/ChatComposer';
import { MessageText } from '@/components/chat/MessageText';
import { toast } from 'sonner';

export function BotPlaygroundPage() {
  const qc = useQueryClient();
  const [sessionsPage, setSessionsPage] = useState(1);
  const sessionsQuery = useChatSessions(sessionsPage);
  const createSession = useCreateChatSession();
  const deleteSession = useDeleteChatSession();
  const escalateSession = useEscalateHandoffSession();
  const [selectedSessionId, setSelectedSessionId] = useState<string>();
  const [sessionToDelete, setSessionToDelete] = useState<ChatSession>();
  const [draft, setDraft] = useState('');
  const [localMessages, setLocalMessages] = useState<ChatMessage[]>([]);
  const [error, setError] = useState<string>();
  const [isStreaming, setIsStreaming] = useState(false);
  const abortRef = useRef<AbortController | undefined>(undefined);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const loadedSessionRef = useRef<string | undefined>(undefined);
  const pendingInitialScrollRef = useRef(false);

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'auto', force = false) => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    if (!force && distanceFromBottom > 120 && behavior === 'auto') return;
    container.scrollTo({ top: container.scrollHeight, behavior });
  }, []);

  const [sessionPollMs, setSessionPollMs] = useState<number | false>(false);
  const sessionQuery = useChatSession(selectedSessionId, sessionPollMs);
  const selectedSession = sessionQuery.data?.session;
  const sessions = useMemo(() => sessionsQuery.data?.data ?? [], [sessionsQuery.data?.data]);
  const sessionsTotal = sessionsQuery.data?.total ?? 0;
  const sessionsTotalPages = Math.ceil(sessionsTotal / 30) || 1;
  const showTypingIndicator = isStreaming && localMessages[localMessages.length - 1]?.role === 'user';
  const canChat = selectedSession?.state === 'bot';
  const sessionState = selectedSession?.state;
  const composerPlaceholder = !selectedSession
    ? 'Выберите или создайте сессию'
    : canChat
      ? 'Введите сообщение. Ctrl/⌘ + Enter отправляет.'
      : sessionState === 'closed'
        ? 'Диалог закрыт — создайте новую сессию'
        : 'Диалог у оператора — откройте Handoff';

  useEffect(() => {
    if (!selectedSessionId && sessions.length > 0) {
      setSelectedSessionId(sessions[0].id);
    }
  }, [selectedSessionId, sessions]);

  useEffect(() => {
    setLocalMessages([]);
    loadedSessionRef.current = undefined;
    pendingInitialScrollRef.current = true;
  }, [selectedSessionId]);

  useEffect(() => {
    const data = sessionQuery.data;
    if (!data || data.session.id !== selectedSessionId) return;
    if (isStreaming) return;

    const isInitialLoad = loadedSessionRef.current !== selectedSessionId;
    if (isInitialLoad) {
      loadedSessionRef.current = selectedSessionId;
      pendingInitialScrollRef.current = true;
    }
    setLocalMessages(data.messages);
  }, [sessionQuery.data, selectedSessionId, isStreaming]);

  useEffect(() => {
    if (isStreaming || !selectedSessionId) {
      setSessionPollMs(false);
      return;
    }
    const messages = sessionQuery.data?.messages;
    if (!messages?.length) {
      setSessionPollMs(false);
      return;
    }
    setSessionPollMs(messages[messages.length - 1]?.role === 'user' ? 2000 : false);
  }, [isStreaming, selectedSessionId, sessionQuery.data?.messages]);

  useEffect(() => {
    if (localMessages.length === 0 && !showTypingIndicator) return;
    const force = pendingInitialScrollRef.current;
    const run = () => scrollToBottom('auto', force);
    requestAnimationFrame(() => {
      run();
      requestAnimationFrame(() => {
        run();
        pendingInitialScrollRef.current = false;
      });
    });
  }, [localMessages, showTypingIndicator, scrollToBottom]);

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
  }

  function stopStreaming() {
    abortRef.current?.abort();
  }

  function escalateSelectedSession() {
    if (!selectedSession || selectedSession.state !== 'bot') return;
    escalateSession.mutate(
      { id: selectedSession.id, reason: 'manual_playground' },
      {
        onSuccess: () => {
          toast.success('Диалог передан оператору');
          qc.invalidateQueries({ queryKey: queryKeys.botChat.sessions(sessionsPage) });
          if (selectedSession) {
            qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(selectedSession.id) });
          }
        },
        onError: () => toast.error('Не удалось передать диалог оператору'),
      }
    );
  }

  async function sendMessage() {
    const content = draft.trim();
    if (!content || isStreaming || !canChat) return;
    setDraft('');
    setError(undefined);
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
          if (event.type === 'message' && event.message) {
            setLocalMessages((items) => [...items, event.message!]);
          }
          if (event.type === 'error') {
            setError(mapChatStreamError(event.error));
          }
        },
        controller.signal
      );
      qc.invalidateQueries({ queryKey: queryKeys.botChat.sessions(sessionsPage) });
      qc.invalidateQueries({ queryKey: queryKeys.botChat.detail(sessionId) });
    } catch (err) {
      if (controller.signal.aborted) {
        setError(undefined);
      } else {
        setError(mapChatStreamError(err instanceof Error ? err.message : 'Ошибка генерации ответа'));
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
            {sessionsQuery.isLoading ? (
              <LoadingState />
            ) : sessionsQuery.isError ? (
              <ErrorState message="Не удалось загрузить сессии." />
            ) : (
            <div className="space-y-2 pr-2">
              {sessions.map((session) => (
                <div
                  key={session.id}
                  className={cn(
                    'group relative rounded-lg border transition hover:bg-muted',
                    selectedSessionId === session.id && 'border-primary bg-muted'
                  )}
                >
                  <button
                    type="button"
                    onClick={() => setSelectedSessionId(session.id)}
                    className="w-full rounded-lg p-3 pr-9 text-left text-sm"
                  >
                    <div className="flex items-center gap-2">
                      <MessageSquare className="h-4 w-4 shrink-0" />
                      <span className="truncate font-medium">{session.title || 'Новая сессия'}</span>
                      {session.state !== 'bot' && (
                        <Badge variant="secondary" className="ml-auto shrink-0 text-[10px]">
                          {STATE_LABELS[session.state]}
                        </Badge>
                      )}
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {session.last_message_at ? new Date(session.last_message_at).toLocaleString() : 'Без сообщений'}
                    </p>
                  </button>
                  <button
                    type="button"
                    aria-label="Удалить чат"
                    title="Удалить чат"
                    onClick={() => setSessionToDelete(session)}
                    className="absolute right-1.5 top-2.5 rounded p-1 text-muted-foreground opacity-0 transition hover:bg-destructive/10 hover:text-destructive focus:opacity-100 group-hover:opacity-100"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
              {sessions.length === 0 && (
                <p className="px-1 text-sm text-muted-foreground">Создайте первую сессию для теста бота.</p>
              )}
              {sessionsTotalPages > 1 && (
                <div className="flex items-center justify-between pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={sessionsPage <= 1}
                    onClick={() => setSessionsPage((p) => p - 1)}
                  >
                    Назад
                  </Button>
                  <span className="text-xs text-muted-foreground">
                    {sessionsPage} / {sessionsTotalPages}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={sessionsPage >= sessionsTotalPages}
                    onClick={() => setSessionsPage((p) => p + 1)}
                  >
                    Далее
                  </Button>
                </div>
              )}
            </div>
            )}
          </ScrollArea>
        </CardContent>
      </Card>

      <Card className="min-w-0 flex-1">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle>Чат с ботом</CardTitle>
              <CardDescription>Ответ строится через RAG и сохраняется в истории с источниками.</CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={escalateSelectedSession}
              disabled={!selectedSession || selectedSession.state !== 'bot' || isStreaming || escalateSession.isPending}
            >
              <Headphones className="h-4 w-4" />
              Передать оператору
            </Button>
          </div>
        </CardHeader>
        <CardContent className="flex h-full min-h-0 flex-col gap-3">
          {sessionState === 'closed' && (
            <Alert>
              <AlertTitle>Диалог закрыт</AlertTitle>
              <AlertDescription>
                Эта сессия завершена. Нажмите «Новая сессия», чтобы продолжить тестирование бота.
              </AlertDescription>
            </Alert>
          )}
          {(sessionState === 'waiting_operator' || sessionState === 'operator') && (
            <Alert>
              <AlertTitle>Диалог у оператора</AlertTitle>
              <AlertDescription>
                Бот не отвечает, пока сессия в handoff. Откройте{' '}
                <Link to="/bot/handoff" className="font-medium underline underline-offset-2">
                  очередь операторов
                </Link>
                .
              </AlertDescription>
            </Alert>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertTitle>Бот не ответил</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div
            ref={scrollContainerRef}
            className="min-h-0 flex-1 overflow-y-auto rounded-lg border bg-muted/20 p-4"
          >
            <div className="space-y-4 pr-3">
              {localMessages.map((message) => (
                <MessageBubble key={message.id} message={message} />
              ))}
              {showTypingIndicator && <BotTypingIndicator />}
              {localMessages.length === 0 && !isStreaming && (
                <div className="flex h-48 flex-col items-center justify-center text-center text-muted-foreground">
                  <Bot className="mb-3 h-8 w-8" />
                  <p className="text-sm">Задайте вопрос по базе знаний, статьям или прайсу.</p>
                </div>
              )}
            </div>
          </div>

          <div className="flex items-end gap-2">
            <ChatComposer
              value={draft}
              onChange={setDraft}
              onSubmit={sendMessage}
              disabled={isStreaming || !canChat}
              placeholder={composerPlaceholder}
              className="flex-1"
            />
            {isStreaming ? (
              <Button variant="outline" onClick={stopStreaming}>
                <Square className="h-4 w-4" />
                Стоп
              </Button>
            ) : (
              <Button onClick={sendMessage} disabled={!draft.trim() || !canChat}>
                <Send className="h-4 w-4" />
                Отправить
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={!!sessionToDelete}
        onOpenChange={(open) => !open && setSessionToDelete(undefined)}
        title="Удалить чат?"
        description="Чат и все его сообщения будут удалены безвозвратно."
        confirmLabel="Удалить"
        loading={deleteSession.isPending}
        onConfirm={async () => {
          if (!sessionToDelete) return;
          try {
            await deleteSession.mutateAsync(sessionToDelete.id);
            if (selectedSessionId === sessionToDelete.id) {
              setSelectedSessionId(undefined);
              setLocalMessages([]);
            }
            toast.success('Чат удалён');
          } catch {
            toast.error('Не удалось удалить чат');
          } finally {
            setSessionToDelete(undefined);
          }
        }}
      />
    </div>
  );
}

const MessageBubble = memo(function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user';
  if (message.role === 'tool') return null;
  const Icon = isUser ? User : Bot;
  const body = isUser ? message.content.trim() : formatMessageBody(message.content);
  return (
    <div className={cn('flex select-none items-start gap-3', isUser && 'flex-row-reverse')}>
      <Icon className={cn('mt-1 h-5 w-5 shrink-0', isUser ? 'text-muted-foreground' : 'text-primary')} />
      <div className={cn('max-w-[80%] rounded-lg p-3 text-sm ring-1', isUser ? 'bg-primary text-primary-foreground ring-primary' : 'bg-card ring-border')}>
        {isUser ? (
          <MessageText text={body} />
        ) : (
          <ChatAnswerContent message={message} />
        )}
        {!isUser && <GuardrailVerdict message={message} />}
      </div>
    </div>
  );
});
function GuardrailVerdict({ message }: { message: ChatMessage }) {
  const action = message.guardrail_action;
  if (!action || action === 'answer') {
    if (typeof message.confidence_score !== 'number') return null;
    return (
      <div className="mt-2 flex select-none items-center gap-1.5 text-xs text-muted-foreground">
        <ShieldCheck className="h-3.5 w-3.5 text-emerald-600" />
        <span>Уверенность {Math.round(message.confidence_score * 100)}%</span>
      </div>
    );
  }
  const label = action === 'escalate' ? 'Эскалация оператору' : 'Отказ';
  return (
    <div className="mt-2 flex select-none flex-wrap items-center gap-2 text-xs">
      <Badge variant={action === 'escalate' ? 'default' : 'secondary'} className="gap-1">
        <ShieldAlert className="h-3.5 w-3.5" />
        {label}
      </Badge>
      {message.refusal_reason && <span className="text-muted-foreground">{guardrailReasonLabel(message.refusal_reason)}</span>}
    </div>
  );
}


import { useMemo, useState } from 'react';
import { Bot, CheckCircle2, Headphones, MessageSquare, RotateCcw, Send, User } from 'lucide-react';
import type { ChatChannel, ChatMessage, ChatSession, ChatState } from '@/types';
import { guardrailReasonLabel, STATE_LABELS } from '@/lib/chatUi';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { apiErrorLabel } from '@/lib/apiError';
import {
  useClaimHandoffSession,
  useCloseHandoffSession,
  useHandoffSession,
  useHandoffSessions,
  useReleaseHandoffSession,
  useSendOperatorMessage,
} from '@/hooks/useBotHandoff';
import { useAuthStore } from '@/stores/authStore';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ChatComposer } from '@/components/chat/ChatComposer';
import { ChatAnswerContent } from '@/components/chat/ChatCitations';
import { MessageText } from '@/components/chat/MessageText';
import { cn } from '@/lib/utils';
import { toast } from 'sonner';

const FILTERS: Array<{ value: 'waiting' | 'mine' | 'active'; label: string }> = [
  { value: 'waiting', label: 'Ожидают' },
  { value: 'mine', label: 'Мои' },
  { value: 'active', label: 'Все активные' },
];

const CHANNEL_FILTERS: Array<{ value: 'all' | ChatChannel; label: string }> = [
  { value: 'all', label: 'Все каналы' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'max', label: 'MAX' },
  { value: 'vk', label: 'VK' },
  { value: 'playground', label: 'Плейграунд' },
  { value: 'api', label: 'API' },
];

export function HandoffPage() {
  const user = useAuthStore((s) => s.user);
  const [filter, setFilter] = useState<'waiting' | 'mine' | 'active'>('waiting');
  const [channelFilter, setChannelFilter] = useState<'all' | ChatChannel>('all');
  const [selectedSessionId, setSelectedSessionId] = useState<string>();
  const [draft, setDraft] = useState('');
  const [confirmClose, setConfirmClose] = useState(false);

  const params = useMemo(() => {
    const channel = channelFilter === 'all' ? undefined : channelFilter;
    if (filter === 'waiting') return { state: 'waiting_operator' as ChatState, channel, page: 1, limit: 50 };
    if (filter === 'mine') return { state: 'operator' as ChatState, operator_id: user?.id, channel, page: 1, limit: 50 };
    return { active: true, channel, page: 1, limit: 50 };
  }, [channelFilter, filter, user?.id]);

  const sessionsQuery = useHandoffSessions(params);
  const sessions = useMemo(() => sessionsQuery.data?.data ?? [], [sessionsQuery.data?.data]);
  const effectiveSelectedSessionId = selectedSessionId && sessions.some((s) => s.id === selectedSessionId)
    ? selectedSessionId
    : sessions[0]?.id;
  const sessionQuery = useHandoffSession(effectiveSelectedSessionId);
  const selected = sessionQuery.data?.session;
  const messages = useMemo(() => sessionQuery.data?.messages ?? [], [sessionQuery.data?.messages]);

  const claim = useClaimHandoffSession();
  const send = useSendOperatorMessage();
  const release = useReleaseHandoffSession();
  const close = useCloseHandoffSession();

  const canManage = user?.role === 'admin' || user?.role === 'superadmin';
  const isAssignedOperator = !!selected?.operator_id && selected.operator_id === user?.id;
  const canSend = selected?.state === 'operator' && (canManage || isAssignedOperator);
  const canRelease = selected?.state === 'operator' && (canManage || isAssignedOperator);
  const canClose = !!selected && selected.state !== 'closed' && (canManage || (selected.state === 'operator' && isAssignedOperator));
  const canClaim = selected?.state === 'waiting_operator';
  const composerPlaceholder = selected
    ? canSend
      ? 'Ответ оператора. Ctrl/⌘+Enter для отправки'
      : canClaim
        ? 'Сначала нажмите «Взять диалог»'
        : selected.state === 'operator'
          ? 'Диалог уже назначен оператору'
          : 'Диалог закрыт или не в handoff'
    : 'Выберите диалог';

  const claimSelected = () => {
    if (!selected || !canClaim) return;
    const claimedId = selected.id;
    claim.mutate(claimedId, {
      onSuccess: () => {
        // Взятый диалог выпадает из фильтра «Ожидают» — переключаемся на «Мои»
        // и сохраняем выбор, чтобы оператор сразу мог ответить клиенту.
        setFilter('mine');
        setSelectedSessionId(claimedId);
        toast.success('Диалог назначен на вас');
      },
      onError: () => toast.error('Не удалось взять диалог'),
    });
  };

  const sendMessage = () => {
    const content = draft.trim();
    if (!selected || !content || !canSend) return;
    send.mutate(
      { sessionId: selected.id, data: { content } },
      {
        onSuccess: () => {
          setDraft('');
          toast.success('Сообщение отправлено');
        },
        onError: () => toast.error('Не удалось отправить сообщение'),
      }
    );
  };

  return (
    <div className="flex h-[calc(100vh-7rem)] gap-4">
      <Card className="w-80 shrink-0">
        <CardHeader>
          <CardTitle>Диалоги / Эскалации</CardTitle>
          <CardDescription>Очередь обращений, которые требуют оператора</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap gap-2">
            {FILTERS.map((item) => (
              <Button
                key={item.value}
                type="button"
                size="sm"
                variant={filter === item.value ? 'default' : 'outline'}
                onClick={() => {
                  setFilter(item.value);
                  setSelectedSessionId(undefined);
                }}
              >
                {item.label}
              </Button>
            ))}
          </div>
          <Select
            value={channelFilter}
            onValueChange={(value) => {
              setChannelFilter(value as 'all' | ChatChannel);
              setSelectedSessionId(undefined);
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CHANNEL_FILTERS.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="h-[calc(100vh-18rem)] space-y-2 overflow-y-auto pr-1">
            {sessions.map((session) => (
              <SessionRow
                key={session.id}
                session={session}
                active={session.id === effectiveSelectedSessionId}
                onClick={() => setSelectedSessionId(session.id)}
              />
            ))}
            {sessionsQuery.isError ? (
              <ErrorState message={apiErrorLabel(sessionsQuery.error, 'Не удалось загрузить очередь диалогов.')} />
            ) : sessionsQuery.isLoading ? (
              <LoadingState />
            ) : sessions.length === 0 ? (
              <p className="px-1 text-sm text-muted-foreground">Нет диалогов для выбранного фильтра.</p>
            ) : null}
          </div>
        </CardContent>
      </Card>

      <Card className="min-w-0 flex-1">
        <CardHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <CardTitle>{selected?.title || 'Выберите диалог'}</CardTitle>
              <CardDescription>
                {selected ? `${selected.channel} · ${STATE_LABELS[selected.state]}` : 'История и ответы оператора появятся здесь'}
              </CardDescription>
            </div>
            {selected && (
              <div className="flex flex-wrap justify-end gap-2">
                {canClaim && (
                  <Button
                    size="sm"
                    onClick={claimSelected}
                    disabled={claim.isPending}
                  >
                    <Headphones className="h-4 w-4" />
                    Взять диалог
                  </Button>
                )}
                {canRelease && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => release.mutate(selected.id, {
                      onSuccess: () => toast.success('Диалог возвращён боту'),
                      onError: () => toast.error('Не удалось вернуть диалог боту'),
                    })}
                    disabled={release.isPending}
                  >
                    <RotateCcw className="h-4 w-4" />
                    Вернуть боту
                  </Button>
                )}
                {canClose && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setConfirmClose(true)}
                    disabled={close.isPending}
                  >
                    <CheckCircle2 className="h-4 w-4" />
                    Завершить диалог
                  </Button>
                )}
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent className="flex h-full min-h-0 flex-col gap-3">
          {selected?.state === 'waiting_operator' && (
            <Alert>
              <Headphones className="h-4 w-4" />
              <AlertTitle>Диалог ожидает оператора</AlertTitle>
              <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
                <span>Возьмите обращение, чтобы ответить клиенту от лица оператора.</span>
                <Button size="sm" onClick={claimSelected} disabled={claim.isPending}>
                  <Headphones className="h-4 w-4" />
                  Взять диалог
                </Button>
              </AlertDescription>
            </Alert>
          )}

          <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border bg-muted/20 p-4">
            <div className="space-y-4 pr-3">
              {messages.map((message) => (
                <MessageBubble key={message.id} message={message} />
              ))}
              {selected && messages.length === 0 && (
                <p className="py-12 text-center text-sm text-muted-foreground">История диалога пуста.</p>
              )}
              {!selected && (
                <div className="flex h-48 flex-col items-center justify-center text-center text-muted-foreground">
                  <MessageSquare className="mb-3 h-8 w-8" />
                  <p className="text-sm">Выберите обращение из очереди.</p>
                </div>
              )}
            </div>
          </div>

          <div className="flex items-end gap-2">
            <ChatComposer
              value={draft}
              onChange={setDraft}
              onSubmit={sendMessage}
              disabled={!canSend || send.isPending}
              placeholder={composerPlaceholder}
            />
            <Button onClick={sendMessage} disabled={!draft.trim() || !canSend || send.isPending}>
              <Send className="h-4 w-4" />
              Отправить
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={confirmClose}
        onOpenChange={setConfirmClose}
        title="Завершить диалог?"
        description="Диалог будет помечен как решённый и возвращён боту. Продолжить?"
        confirmLabel="Завершить диалог"
        loadingLabel="Завершение..."
        destructive={false}
        loading={close.isPending}
        onConfirm={() => {
          if (!selected) return;
          close.mutate(selected.id, {
            onSuccess: () => {
              toast.success('Диалог завершён');
              setConfirmClose(false);
            },
            onError: () => toast.error('Не удалось завершить диалог'),
          });
        }}
      />
    </div>
  );
}

function SessionRow({ session, active, onClick }: { session: ChatSession; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full rounded-lg border p-3 text-left text-sm transition hover:bg-muted',
        active && 'border-primary bg-muted'
      )}
    >
      <div className="flex items-center gap-2">
        <MessageSquare className="h-4 w-4 shrink-0" />
        <span className="truncate font-medium">{session.title || 'Новая сессия'}</span>
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <Badge variant={session.state === 'waiting_operator' ? 'destructive' : 'outline'}>
          {STATE_LABELS[session.state]}
        </Badge>
        <Badge variant="secondary">{session.channel}</Badge>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        {session.last_message_at ? new Date(session.last_message_at).toLocaleString() : 'Без сообщений'}
      </p>
    </button>
  );
}

function MessageBubble({ message }: { message: ChatMessage }) {
  if (message.role === 'tool') return null;
  const isUser = message.role === 'user';
  const isOperator = message.role === 'operator';

  return (
    <div className={cn('flex gap-3', isUser && 'justify-end')}>
      {!isUser && (
        <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
          {isOperator ? <Headphones className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
        </div>
      )}
      <div
        className={cn(
          'max-w-[78%] rounded-lg border bg-card p-3 text-sm shadow-sm',
          isUser && 'bg-primary text-primary-foreground',
          isOperator && 'border-primary/40'
        )}
      >
        <div className="mb-1 flex items-center gap-2 text-xs opacity-70">
          {isUser ? <User className="h-3 w-3" /> : isOperator ? <Headphones className="h-3 w-3" /> : <Bot className="h-3 w-3" />}
          <span>{isUser ? 'Клиент' : isOperator ? 'Оператор' : 'Бот'}</span>
          <span>{new Date(message.created_at).toLocaleTimeString()}</span>
        </div>
        {message.role === 'assistant' ? <ChatAnswerContent message={message} /> : <MessageText text={message.content} />}
        {message.guardrail_action === 'escalate' && (
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Badge variant="destructive">Эскалация</Badge>
            {message.refusal_reason && (
              <span className="text-xs opacity-70">{guardrailReasonLabel(message.refusal_reason)}</span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

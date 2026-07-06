import { type ReactNode, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { toast } from 'sonner';
import { Copy, RefreshCw } from 'lucide-react';
import { botAdminApi } from '@/api/botAdmin';
import type { ChannelStatus, SecretKind, TenantSecretStatus } from '@/types';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  useBotSettings,
  useUpdateBotSettings,
  useBotSecrets,
  useChannelStatus,
  useSetBotSecret,
  useDeleteBotSecret,
  useRegisterChannelWebhook,
  useChannelSubscriptions,
} from '@/hooks/useBotAdmin';
import {
  SECRET_LABELS,
  CHANNEL_LABELS,
  CHANNEL_METADATA_FIELDS,
  SERVICE_SECRET_HINTS,
  CHANNEL_HINTS,
  CHANNEL_SECRET_KINDS,
  SERVICE_SECRET_KINDS,
  MASKED_SECRET_VALUE,
  isSensitiveMetadataField,
  canGenerateWebhookSecret,
  generateWebhookSecret,
  buildChannelFormFromSecret,
} from '@/components/bot-admin/constants';
import { FieldHint } from '@/components/bot-admin/FieldHint';
import { RevealableSecretInput } from '@/components/bot-admin/RevealableSecretInput';
import { channelModules } from '@/components/bot-admin/moduleHelpers';

export function ChannelsAndSecretsTab() {
  const { data: secretStatuses, isLoading: secretsLoading, isError: secretsError } = useBotSecrets();
  const { data: statuses, isLoading: channelsLoading, isError: channelsError } = useChannelStatus();
  const { data: settings } = useBotSettings();
  const updateSettings = useUpdateBotSettings();
  const setSecret = useSetBotSecret();
  const deleteSecret = useDeleteBotSecret();
  const registerWebhook = useRegisterChannelWebhook();
  const subscriptions = useChannelSubscriptions();

  const [editingService, setEditingService] = useState<TenantSecretStatus | null>(null);
  const [editingChannel, setEditingChannel] = useState<ChannelStatus | null>(null);
  const [serviceValue, setServiceValue] = useState('');
  const [token, setToken] = useState('');
  const [metadata, setMetadata] = useState<Record<string, string>>({});
  const [channelEditorLoading, setChannelEditorLoading] = useState(false);
  const [deleteKind, setDeleteKind] = useState<SecretKind | null>(null);
  const [subscriptionsChannel, setSubscriptionsChannel] = useState<ChannelStatus['channel'] | null>(null);
  const [subscriptionsData, setSubscriptionsData] = useState<unknown>(null);

  if (secretsLoading || channelsLoading) return <LoadingState />;
  if (secretsError || channelsError) return <ErrorState message="Не удалось загрузить каналы и секреты." />;

  const serviceSecrets = (secretStatuses ?? []).filter((s) => SERVICE_SECRET_KINDS.includes(s.kind));
  const modules = channelModules(settings?.enabled_modules);

  const openChannelEditor = (status: ChannelStatus) => {
    setEditingChannel(status);
    setToken('');
    setMetadata({});
    setChannelEditorLoading(true);
    botAdminApi
      .getSecretForEdit(status.secret_kind)
      .then((secret) => {
        const form = buildChannelFormFromSecret(status.channel, secret);
        setToken(form.token);
        setMetadata(form.metadata);
      })
      .catch(() => {
        toast.error('Не удалось загрузить текущие значения канала');
        setEditingChannel(null);
      })
      .finally(() => setChannelEditorLoading(false));
  };

  const saveChannelSecret = () => {
    if (!editingChannel || !token.trim()) return;
    const cleanMetadata = Object.fromEntries(
      Object.entries(metadata)
        .map(([key, value]) => [key, value.trim()])
        // Drop empty values and the masked sentinel so we never overwrite a real
        // stored secret with the display-only placeholder.
        .filter(([, value]) => value && value !== MASKED_SECRET_VALUE)
    );
    setSecret.mutate(
      { kind: editingChannel.secret_kind, data: { value: token.trim(), metadata: cleanMetadata } },
      {
        onSuccess: () => {
          setEditingChannel(null);
          setToken('');
          setMetadata({});
          toast.success('Канал сохранён');
        },
        onError: () => toast.error('Не удалось сохранить канал'),
      }
    );
  };

  const saveServiceSecret = () => {
    if (!editingService || !serviceValue.trim()) return;
    setSecret.mutate(
      { kind: editingService.kind, data: { value: serviceValue } },
      {
        onSuccess: () => {
          setEditingService(null);
          setServiceValue('');
          toast.success('Секрет сохранён');
        },
        onError: () => toast.error('Не удалось сохранить секрет'),
      }
    );
  };

  const handleDelete = () => {
    if (!deleteKind) return;
    deleteSecret.mutate(deleteKind, {
      onSuccess: () => {
        setDeleteKind(null);
        toast.success('Секрет удалён');
      },
      onError: () => toast.error('Не удалось удалить секрет'),
    });
  };

  const toggleChannel = (channel: ChannelStatus['channel'], enabled: boolean) => {
    if (!settings) return;
    const current = channelModules(settings.enabled_modules);
    const enabled_modules = {
      ...(settings.enabled_modules ?? {}),
      channels: { ...current, [channel]: enabled },
    };
    updateSettings.mutate(
      { enabled_modules },
      {
        onSuccess: () => toast.success('Настройки каналов сохранены'),
        onError: () => toast.error('Не удалось сохранить настройки каналов'),
      }
    );
  };

  const handleRegisterWebhook = (channel: ChannelStatus['channel']) => {
    registerWebhook.mutate(channel, {
      onSuccess: (res) =>
        res.registered
          ? toast.success('Webhook зарегистрирован')
          : toast.warning('Webhook не зарегистрирован: заполните обязательные поля канала'),
      onError: () => toast.error('Не удалось зарегистрировать webhook'),
    });
  };

  const handleViewSubscriptions = (channel: ChannelStatus['channel']) => {
    setSubscriptionsChannel(channel);
    setSubscriptionsData(null);
    subscriptions.mutate(channel, {
      onSuccess: (data) => setSubscriptionsData(data),
      onError: () => {
        toast.error('Не удалось получить подписки');
        setSubscriptionsChannel(null);
      },
    });
  };

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="space-y-3">
        <div>
          <h2 className="text-base font-semibold">Сервисные секреты</h2>
          <p className="text-sm text-muted-foreground">
            Ключи для LLM и интеграций, не привязанные к мессенджерам.
          </p>
        </div>
        {serviceSecrets.map((s) => (
          <Card key={s.kind}>
            <CardContent className="pt-6 flex items-center justify-between gap-4">
              <div>
                <FieldHint
                  label={SECRET_LABELS[s.kind] ?? s.kind}
                  hint={SERVICE_SECRET_HINTS[s.kind] ?? 'Секрет для интеграции.'}
                />
                <div className="flex items-center gap-2 mt-1">
                  <Badge variant={s.is_set ? 'secondary' : 'outline'}>
                    {s.is_set ? 'Настроен' : 'Не задан'}
                  </Badge>
                  {s.updated_at && (
                    <span className="text-xs text-muted-foreground">
                      {new Date(s.updated_at).toLocaleString()}
                    </span>
                  )}
                </div>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => { setEditingService(s); setServiceValue(''); }}>
                  {s.is_set ? 'Обновить' : 'Задать'}
                </Button>
                {s.is_set && (
                  <Button variant="destructive" size="sm" onClick={() => setDeleteKind(s.kind)}>
                    Удалить
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="space-y-3">
        <div>
          <h2 className="text-base font-semibold">Каналы связи</h2>
          <p className="text-sm text-muted-foreground">
            Токены ботов, webhook и параметры Telegram, MAX и VK в одном месте.
          </p>
        </div>
        {(statuses ?? []).map((status) => (
          <Card key={status.channel}>
            <CardHeader>
              <div className="flex items-center justify-between gap-4">
                <div>
                  <CardTitle className="text-base">{CHANNEL_LABELS[status.channel]}</CardTitle>
                  <CardDescription>Токен, webhook и дополнительные параметры канала</CardDescription>
                </div>
                <Badge variant={status.configured ? 'secondary' : 'outline'}>
                  {status.configured ? 'Настроен' : 'Не задан'}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-3">
                <FieldHint
                  label="Состояние канала"
                  htmlFor={`channel-${status.channel}`}
                  hint={CHANNEL_HINTS.enabled}
                />
                <ChannelEnabledButton
                  id={`channel-${status.channel}`}
                  enabled={modules[status.channel]}
                  disabled={!settings || updateSettings.isPending}
                  onToggle={() => toggleChannel(status.channel, !modules[status.channel])}
                />
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <FieldHint label="Webhook URL" hint={CHANNEL_HINTS.webhook} />
                  {status.channel === 'telegram' ? (
                    <WebhookStatusBadge status={status} />
                  ) : (
                    <span className="text-xs text-muted-foreground">Проверка доступна для Telegram</span>
                  )}
                </div>
                <div className="flex gap-2">
                  <Input value={status.webhook_url} readOnly />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label="Скопировать Webhook URL"
                    onClick={() => {
                      navigator.clipboard?.writeText(status.webhook_url);
                      toast.success('Webhook URL скопирован');
                    }}
                  >
                    <Copy className="h-4 w-4" aria-hidden="true" />
                    <span className="sr-only">Скопировать Webhook URL</span>
                  </Button>
                </div>
              </div>

              <div className="flex flex-wrap justify-end gap-2">
                {WEBHOOK_ACTION_CHANNELS.includes(status.channel) && (
                  <>
                    <Button
                      variant="outline"
                      disabled={!status.configured || registerWebhook.isPending}
                      onClick={() => handleRegisterWebhook(status.channel)}
                    >
                      Зарегистрировать вебхук
                    </Button>
                    <SubscriptionsButton
                      configured={status.configured}
                      loading={subscriptions.isPending && subscriptionsChannel === status.channel}
                      onClick={() => handleViewSubscriptions(status.channel)}
                    />
                  </>
                )}
                <Button variant="outline" onClick={() => openChannelEditor(status)}>
                  {status.configured ? 'Обновить токен и параметры' : 'Настроить'}
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Dialog open={!!editingService} onOpenChange={(open) => !open && setEditingService(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingService ? SECRET_LABELS[editingService.kind] : 'Секрет'}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <FieldHint
              label={<RequiredLabel>Значение</RequiredLabel>}
              htmlFor="service-secret-value"
              hint={editingService ? (SERVICE_SECRET_HINTS[editingService.kind] ?? '') : ''}
            />
            <Input
              id="service-secret-value"
              type="password"
              value={serviceValue}
              onChange={(e) => setServiceValue(e.target.value)}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingService(null)}>Отмена</Button>
            <Button onClick={saveServiceSecret} disabled={!serviceValue.trim() || setSecret.isPending}>
              Сохранить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={!!editingChannel}
        onOpenChange={(open) => {
          if (!open) {
            setEditingChannel(null);
            setToken('');
            setMetadata({});
            setChannelEditorLoading(false);
          }
        }}
      >
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingChannel ? CHANNEL_LABELS[editingChannel.channel] : 'Канал'}</DialogTitle>
          </DialogHeader>
          {channelEditorLoading ? (
            <p className="text-sm text-muted-foreground">Загрузка текущих значений…</p>
          ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <FieldHint
                label={<RequiredLabel>Токен бота</RequiredLabel>}
                htmlFor="channel-token"
                hint={editingChannel ? CHANNEL_HINTS.token[editingChannel.channel] : ''}
              />
              <RevealableSecretInput
                id="channel-token"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={token ? undefined : 'Токен бота'}
              />
            </div>
            {editingChannel &&
              CHANNEL_METADATA_FIELDS[editingChannel.channel].map((field) => {
                const showGenerate = canGenerateWebhookSecret(editingChannel.channel, field.key);
                const isPassword = isSensitiveMetadataField(field.key);
                const fieldValue = metadata[field.key] ?? '';
                return (
                  <div className="space-y-2" key={field.key}>
                    <FieldHint
                      label={field.required ? <RequiredLabel>{field.label}</RequiredLabel> : field.label}
                      htmlFor={`metadata-${field.key}`}
                      hint={field.hint}
                    />
                    <div className="flex gap-2">
                      <RevealableSecretInput
                        id={`metadata-${field.key}`}
                        secret={isPassword}
                        value={fieldValue}
                        placeholder={fieldValue ? undefined : field.placeholder}
                        onChange={(e) => setMetadata((m) => ({ ...m, [field.key]: e.target.value }))}
                      />
                      {showGenerate && (
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          aria-label="Сгенерировать webhook secret"
                          title="Сгенерировать webhook secret"
                          onClick={() =>
                            setMetadata((m) => ({ ...m, [field.key]: generateWebhookSecret() }))
                          }
                        >
                          <RefreshCw className="h-4 w-4" aria-hidden="true" />
                        </Button>
                      )}
                    </div>
                  </div>
                );
              })}
          </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingChannel(null)}>Отмена</Button>
            <Button
              onClick={saveChannelSecret}
              disabled={channelEditorLoading || !token.trim() || setSecret.isPending}
            >
              Сохранить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!subscriptionsChannel} onOpenChange={(open) => !open && setSubscriptionsChannel(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              Подписки {subscriptionsChannel ? CHANNEL_LABELS[subscriptionsChannel] : ''}
            </DialogTitle>
          </DialogHeader>
          {subscriptions.isPending ? (
            <p className="text-sm text-muted-foreground">Загрузка подписок…</p>
          ) : (
            <SubscriptionsView channel={subscriptionsChannel} data={subscriptionsData} />
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setSubscriptionsChannel(null)}>Закрыть</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteKind}
        onOpenChange={(open) => !open && setDeleteKind(null)}
        title="Удалить секрет"
        description={
          deleteKind && CHANNEL_SECRET_KINDS.includes(deleteKind)
            ? 'Секрет будет удалён. Канал перестанет принимать и отправлять сообщения.'
            : 'Секрет будет удалён. Связанная интеграция может перестать работать.'
        }
        onConfirm={handleDelete}
        loading={deleteSecret.isPending}
      />
    </div>
  );
}

const WEBHOOK_ACTION_CHANNELS: ChannelStatus['channel'][] = ['telegram', 'max'];

type ParsedSubscription = {
  key: string;
  createdAt: string;
  updateTypes: string[];
};

function SubscriptionsView({ channel, data }: { channel: ChannelStatus['channel'] | null; data: unknown }) {
  const normalizedData = normalizeSubscriptionPayload(data);
  const rows = parseSubscriptions(normalizedData);
  const showCreatedAt = channel !== 'telegram';

  if (!normalizedData) {
    return <p className="text-sm text-muted-foreground">Нет данных</p>;
  }

  if (rows.length === 0) {
    return (
      <div className="space-y-3">
        <p className="text-sm text-muted-foreground">Подписки не найдены или формат ответа не распознан.</p>
        <pre className="max-h-[40vh] overflow-auto rounded-md bg-muted p-3 font-mono text-xs whitespace-pre-wrap break-all">
          {JSON.stringify(normalizedData, null, 2)}
        </pre>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {rows.map((row, index) => (
        <div key={row.key} className="rounded-md border p-3">
          <div className="mb-2 text-sm font-medium">Подписка {index + 1}</div>
          <div className="grid gap-2 text-sm sm:grid-cols-[140px_1fr]">
            {showCreatedAt && (
              <>
                <span className="text-muted-foreground">Дата создания</span>
                <span>{row.createdAt}</span>
              </>
            )}
            <span className="text-muted-foreground">События</span>
            <div className="flex flex-wrap gap-1.5">
              {row.updateTypes.length > 0 ? (
                row.updateTypes.map((type) => (
                  <Badge key={type} variant="outline" className="font-mono text-xs">
                    {type}
                  </Badge>
                ))
              ) : (
                <span className="text-muted-foreground">Не указаны</span>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function parseSubscriptions(data: unknown): ParsedSubscription[] {
  const items = extractSubscriptionItems(normalizeSubscriptionPayload(data));
  return items.map((item, index) => ({
    key: subscriptionKey(item, index),
    createdAt: formatSubscriptionDate(readField(item, [
      'time',
      'created_at',
      'createdAt',
      'created',
      'create_time',
      'createTime',
      'created_time',
      'createdTime',
      'creation_time',
      'creationTime',
      'created_timestamp',
      'createdTimestamp',
      'created_ts',
      'createdTs',
      'timestamp',
      'ts',
    ])),
    updateTypes: formatUpdateTypes(readField(item, [
      'update_types',
      'updateTypes',
      'allowed_updates',
      'allowedUpdates',
      'types',
    ])),
  }));
}

function normalizeSubscriptionPayload(data: unknown): unknown {
  if (typeof data !== 'string') {
    return data;
  }
  const trimmed = data.trim();
  if (!trimmed) {
    return data;
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    return data;
  }
}

function extractSubscriptionItems(data: unknown): Record<string, unknown>[] {
  if (Array.isArray(data)) {
    return data.filter(isRecord);
  }
  if (!isRecord(data)) {
    return [];
  }
  for (const key of ['subscriptions', 'items', 'data', 'result']) {
    const value = data[key];
    if (Array.isArray(value)) {
      return value.filter(isRecord);
    }
  }
  if (isRecord(data.result)) {
    return [data.result];
  }
  return [data];
}

function subscriptionKey(item: Record<string, unknown>, index: number) {
  const id = readField(item, ['id', 'subscription_id', 'subscriptionId', 'url']);
  return typeof id === 'string' && id.trim() ? id : String(index);
}

function readField(item: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    if (item[key] != null) {
      return item[key];
    }
  }
  return undefined;
}

function formatSubscriptionDate(value: unknown) {
  if (value == null || value === '') {
    return 'Не указана';
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (/^\d+$/.test(trimmed)) {
      return formatTimestamp(Number(trimmed));
    }
  }
  if (typeof value === 'number') {
    return formatTimestamp(value);
  }
  const date =
    value instanceof Date
      ? value
      : new Date(String(value));
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return date.toLocaleString();
}

function formatTimestamp(value: number) {
  const date = new Date(value < 10_000_000_000 ? value * 1000 : value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return date.toLocaleString();
}

function formatUpdateTypes(value: unknown) {
  if (Array.isArray(value)) {
    return value.map(String).map((item) => item.trim()).filter(Boolean);
  }
  if (typeof value === 'string') {
    return value.split(',').map((item) => item.trim()).filter(Boolean);
  }
  return [];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function SubscriptionsButton({
  configured,
  loading,
  onClick,
}: {
  configured: boolean;
  loading: boolean;
  onClick: () => void;
}) {
  const button = (
    <Button variant="outline" disabled={!configured || loading} onClick={onClick}>
      Посмотреть подписки
    </Button>
  );

  if (configured) {
    return button;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="inline-flex" tabIndex={0} aria-label="Не настроен токен бота">
            {button}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-xs p-3 text-xs leading-relaxed whitespace-normal">
        Не настроен токен бота
      </TooltipContent>
    </Tooltip>
  );
}

function RequiredLabel({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span>{children}</span>
      <span className="text-[10px] font-medium uppercase tracking-wide text-destructive">обязательно</span>
    </span>
  );
}

function ChannelEnabledButton({
  id,
  enabled,
  disabled,
  onToggle,
}: {
  id: string;
  enabled: boolean;
  disabled: boolean;
  onToggle: () => void;
}) {
  return (
    <Button
      id={id}
      type="button"
      size="sm"
      variant={enabled ? 'default' : 'outline'}
      aria-pressed={enabled}
      disabled={disabled}
      onClick={onToggle}
      className={
        enabled
          ? 'border-emerald-600 bg-emerald-600 text-white hover:bg-emerald-700'
          : 'border-destructive/60 text-destructive hover:bg-destructive/10 hover:text-destructive'
      }
    >
      {enabled ? 'Канал включён' : 'Канал выключен'}
    </Button>
  );
}

function WebhookStatusBadge({ status }: { status: ChannelStatus }) {
  const badge = (
    <Badge
      variant={status.webhook_configured ? 'secondary' : 'outline'}
      className={status.webhook_error ? 'cursor-help' : undefined}
    >
      {status.webhook_configured ? 'Настроен' : 'Не настроен'}
    </Badge>
  );

  if (!status.webhook_error) {
    return badge;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="inline-flex" tabIndex={0} aria-label="Ошибка проверки webhook">
            {badge}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-sm p-3 text-xs leading-relaxed whitespace-normal">
        {status.webhook_error}
      </TooltipContent>
    </Tooltip>
  );
}


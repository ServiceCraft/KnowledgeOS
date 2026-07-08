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
} from '@/hooks/useBotAdmin';
import {
  SECRET_LABELS,
  CHANNEL_LABELS,
  CHANNEL_METADATA_FIELDS,
  SERVICE_SECRET_HINTS,
  SERVICE_SECRET_METADATA_FIELDS,
  CHANNEL_HINTS,
  CHANNEL_SECRET_KINDS,
  SERVICE_SECRET_KINDS,
  MASKED_SECRET_VALUE,
  isSensitiveMetadataField,
  canGenerateWebhookSecret,
  generateWebhookSecret,
  buildChannelFormFromSecret,
  parseMetadataRecord,
  metadataFieldString,
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

  const [editingService, setEditingService] = useState<TenantSecretStatus | null>(null);
  const [editingChannel, setEditingChannel] = useState<ChannelStatus | null>(null);
  const [serviceValue, setServiceValue] = useState('');
  const [serviceMetadata, setServiceMetadata] = useState<Record<string, string>>({});
  const [serviceEditorLoading, setServiceEditorLoading] = useState(false);
  const [token, setToken] = useState('');
  const [metadata, setMetadata] = useState<Record<string, string>>({});
  const [channelEditorLoading, setChannelEditorLoading] = useState(false);
  const [deleteKind, setDeleteKind] = useState<SecretKind | null>(null);

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

  const openServiceEditor = (status: TenantSecretStatus) => {
    setEditingService(status);
    setServiceValue('');
    setServiceMetadata({});
    const fields = SERVICE_SECRET_METADATA_FIELDS[status.kind];
    // Only secrets with extra config fields need the stored values prefilled;
    // value-only secrets (llm) keep the blank re-entry behaviour.
    if (!fields || !status.is_set) return;
    setServiceEditorLoading(true);
    botAdminApi
      .getSecretForEdit(status.kind)
      .then((secret) => {
        setServiceValue(secret.value);
        const meta = parseMetadataRecord(secret.metadata);
        setServiceMetadata(
          Object.fromEntries(fields.map((f) => [f.key, metadataFieldString(meta[f.key])]))
        );
      })
      .catch(() => {
        toast.error('Не удалось загрузить текущие значения интеграции');
        setEditingService(null);
      })
      .finally(() => setServiceEditorLoading(false));
  };

  const serviceMetadataValid = () => {
    if (!editingService) return false;
    const fields = SERVICE_SECRET_METADATA_FIELDS[editingService.kind] ?? [];
    return fields.every((f) => !f.required || (serviceMetadata[f.key] ?? '').trim());
  };

  const saveServiceSecret = () => {
    if (!editingService || !serviceValue.trim() || !serviceMetadataValid()) return;
    const fields = SERVICE_SECRET_METADATA_FIELDS[editingService.kind];
    const cleanMetadata = fields
      ? Object.fromEntries(
          Object.entries(serviceMetadata)
            .map(([key, value]) => [key, value.trim()])
            .filter(([, value]) => value && value !== MASKED_SECRET_VALUE)
        )
      : undefined;
    setSecret.mutate(
      {
        kind: editingService.kind,
        data: cleanMetadata ? { value: serviceValue, metadata: cleanMetadata } : { value: serviceValue },
      },
      {
        onSuccess: () => {
          setEditingService(null);
          setServiceValue('');
          setServiceMetadata({});
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
          : toast.warning(res.reason ?? 'Webhook не зарегистрирован: заполните обязательные поля канала'),
      onError: () => toast.error('Не удалось зарегистрировать webhook'),
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
                <Button variant="outline" size="sm" onClick={() => openServiceEditor(s)}>
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
                <div className="flex flex-col items-end gap-1.5">
                  <ChannelSetupStatusBadge label="Параметры" ok={status.configured} />
                  {WEBHOOK_CHECK_CHANNELS.includes(status.channel) && (
                    <ChannelSetupStatusBadge
                      label="Webhook"
                      ok={status.webhook_configured}
                      error={status.webhook_error}
                    />
                  )}
                </div>
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
                <FieldHint label="Webhook URL" hint={CHANNEL_HINTS.webhook} />
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
                {WEBHOOK_CHECK_CHANNELS.includes(status.channel) &&
                  status.configured &&
                  !status.webhook_configured && (
                    <Button
                      variant="outline"
                      disabled={registerWebhook.isPending}
                      onClick={() => handleRegisterWebhook(status.channel)}
                    >
                      Зарегистрировать вебхук
                    </Button>
                  )}
                <Button variant="outline" onClick={() => openChannelEditor(status)}>
                  {status.configured ? 'Параметры' : 'Настроить'}
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Dialog
        open={!!editingService}
        onOpenChange={(open) => {
          if (!open) {
            setEditingService(null);
            setServiceValue('');
            setServiceMetadata({});
            setServiceEditorLoading(false);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingService ? SECRET_LABELS[editingService.kind] : 'Секрет'}
            </DialogTitle>
          </DialogHeader>
          {serviceEditorLoading ? (
            <p className="text-sm text-muted-foreground">Загрузка текущих значений…</p>
          ) : (
            <div className="space-y-4">
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
              {editingService &&
                (SERVICE_SECRET_METADATA_FIELDS[editingService.kind] ?? []).map((field) => (
                  <div className="space-y-2" key={field.key}>
                    <FieldHint
                      label={field.required ? <RequiredLabel>{field.label}</RequiredLabel> : field.label}
                      htmlFor={`service-metadata-${field.key}`}
                      hint={field.hint}
                    />
                    <Input
                      id={`service-metadata-${field.key}`}
                      value={serviceMetadata[field.key] ?? ''}
                      placeholder={field.placeholder}
                      onChange={(e) =>
                        setServiceMetadata((m) => ({ ...m, [field.key]: e.target.value }))
                      }
                    />
                  </div>
                ))}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingService(null)}>Отмена</Button>
            <Button
              onClick={saveServiceSecret}
              disabled={serviceEditorLoading || !serviceValue.trim() || !serviceMetadataValid() || setSecret.isPending}
            >
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

const WEBHOOK_CHECK_CHANNELS: ChannelStatus['channel'][] = ['telegram', 'max'];

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
      {enabled ? 'Включен' : 'Выключен'}
    </Button>
  );
}

function ChannelSetupStatusBadge({
  label,
  ok,
  error,
}: {
  label: string;
  ok: boolean;
  error?: string;
}) {
  const badge = (
    <Badge
      variant="outline"
      className={
        ok
          ? 'border-emerald-600/60 bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-400'
          : 'border-destructive/60 bg-destructive/10 text-destructive'
      }
    >
      {label}
    </Badge>
  );

  if (!error) {
    return badge;
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="inline-flex cursor-help" tabIndex={0} aria-label={`${label}: ${error}`}>
            {badge}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-sm p-3 text-xs leading-relaxed whitespace-normal">
        {error}
      </TooltipContent>
    </Tooltip>
  );
}


import { useState } from 'react';
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
import type { ChannelStatus, SecretKind, TenantSecretStatus } from '@/types';
import {
  useBotSettings,
  useUpdateBotSettings,
  useBotSecrets,
  useChannelStatus,
  useSetBotSecret,
  useDeleteBotSecret,
} from '@/hooks/useBotAdmin';
import {
  SECRET_LABELS,
  CHANNEL_LABELS,
  CHANNEL_METADATA_FIELDS,
  SERVICE_SECRET_HINTS,
  CHANNEL_HINTS,
  CHANNEL_SECRET_KINDS,
  SERVICE_SECRET_KINDS,
} from '@/components/bot-admin/constants';
import { FieldHint } from '@/components/bot-admin/FieldHint';
import { channelModules } from '@/components/bot-admin/moduleHelpers';

export function ChannelsAndSecretsTab() {
  const { data: secretStatuses, isLoading: secretsLoading, isError: secretsError } = useBotSecrets();
  const { data: statuses, isLoading: channelsLoading, isError: channelsError } = useChannelStatus();
  const { data: settings } = useBotSettings();
  const updateSettings = useUpdateBotSettings();
  const setSecret = useSetBotSecret();
  const deleteSecret = useDeleteBotSecret();

  const [editingService, setEditingService] = useState<TenantSecretStatus | null>(null);
  const [editingChannel, setEditingChannel] = useState<ChannelStatus | null>(null);
  const [serviceValue, setServiceValue] = useState('');
  const [token, setToken] = useState('');
  const [metadata, setMetadata] = useState<Record<string, string>>({});
  const [deleteKind, setDeleteKind] = useState<SecretKind | null>(null);

  if (secretsLoading || channelsLoading) return <LoadingState />;
  if (secretsError || channelsError) return <ErrorState message="Не удалось загрузить каналы и секреты." />;

  const serviceSecrets = (secretStatuses ?? []).filter((s) => SERVICE_SECRET_KINDS.includes(s.kind));
  const modules = channelModules(settings?.enabled_modules);

  const openChannelEditor = (status: ChannelStatus) => {
    const next: Record<string, string> = {};
    for (const field of CHANNEL_METADATA_FIELDS[status.channel]) {
      const value = status.metadata?.[field.key];
      next[field.key] = typeof value === 'string' ? value : value == null ? '' : String(value);
    }
    setEditingChannel(status);
    setToken('');
    setMetadata(next);
  };

  const saveChannelSecret = () => {
    if (!editingChannel || !token.trim()) return;
    const cleanMetadata = Object.fromEntries(
      Object.entries(metadata)
        .map(([key, value]) => [key, value.trim()])
        .filter(([, value]) => value)
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
                  label="Канал включён"
                  htmlFor={`channel-${status.channel}`}
                  hint={CHANNEL_HINTS.enabled}
                />
                <input
                  id={`channel-${status.channel}`}
                  type="checkbox"
                  checked={modules[status.channel]}
                  disabled={!settings || updateSettings.isPending}
                  onChange={(e) => toggleChannel(status.channel, e.target.checked)}
                />
                {!status.bot_enabled && (
                  <span className="text-xs text-muted-foreground">Бот выключен в основных настройках</span>
                )}
              </div>

              <div className="space-y-2">
                <FieldHint label="Webhook URL" hint={CHANNEL_HINTS.webhook} />
                <div className="flex gap-2">
                  <Input value={status.webhook_url} readOnly />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      navigator.clipboard?.writeText(status.webhook_url);
                      toast.success('Webhook URL скопирован');
                    }}
                  >
                    Скопировать
                  </Button>
                </div>
              </div>

              <div className="flex justify-end">
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
              label="Значение"
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

      <Dialog open={!!editingChannel} onOpenChange={(open) => !open && setEditingChannel(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editingChannel ? CHANNEL_LABELS[editingChannel.channel] : 'Канал'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <FieldHint
                label="Токен бота"
                htmlFor="channel-token"
                hint={editingChannel ? CHANNEL_HINTS.token[editingChannel.channel] : ''}
              />
              <Input
                id="channel-token"
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Введите токен заново при сохранении"
                autoFocus
              />
            </div>
            {editingChannel &&
              CHANNEL_METADATA_FIELDS[editingChannel.channel].map((field) => (
                <div className="space-y-2" key={field.key}>
                  <FieldHint label={field.label} htmlFor={`metadata-${field.key}`} hint={field.hint} />
                  <Input
                    id={`metadata-${field.key}`}
                    value={metadata[field.key] ?? ''}
                    placeholder={field.placeholder}
                    onChange={(e) => setMetadata((m) => ({ ...m, [field.key]: e.target.value }))}
                  />
                </div>
              ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingChannel(null)}>Отмена</Button>
            <Button onClick={saveChannelSecret} disabled={!token.trim() || setSecret.isPending}>
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

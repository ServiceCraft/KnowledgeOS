import { useEffect, useState, type ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { NumberInput } from '@/components/ui/number-input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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
import {
  useBotSettings,
  useUpdateBotSettings,
  useBotSecrets,
  useChannelStatus,
  useSetBotSecret,
  useDeleteBotSecret,
} from '@/hooks/useBotAdmin';
import { useRagIndexStatus, useRagReindex, useRagSearch } from '@/hooks/useRag';
import type { BotSettings, ChannelStatus, SecretKind, TenantSecretStatus } from '@/types';
import { toast } from 'sonner';
import { HelpCircle, Loader2, RefreshCw } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

const BOOL_LABELS = { true: 'Да', false: 'Нет' } as const;

const MODEL_TIER_LABELS: Record<BotSettings['model_tier'], string> = {
  lite: 'Лайт',
  pro: 'Про',
};

const SECRET_LABELS: Record<SecretKind, string> = {
  llm: 'API языковой модели',
  telegram: 'Telegram',
  max: 'MAX',
  vk: 'ВКонтакте',
  bitrix24: 'Битрикс24',
};

const CHANNEL_LABELS: Record<ChannelStatus['channel'], string> = {
  telegram: 'Telegram',
  max: 'MAX',
  vk: 'ВКонтакте',
};

const CHANNEL_METADATA_FIELDS: Record<
  ChannelStatus['channel'],
  Array<{ key: string; label: string; placeholder?: string; hint: string }>
> = {
  telegram: [
    {
      key: 'webhook_secret',
      label: 'Webhook secret token',
      placeholder: 'Секрет для X-Telegram-Bot-Api-Secret-Token',
      hint: 'Произвольная строка для проверки входящих webhook. Задайте её здесь и укажите тот же secret при регистрации webhook в Bot API (setWebhook). Telegram присылает её в заголовке X-Telegram-Bot-Api-Secret-Token.',
    },
    {
      key: 'handoff_notification_chat_id',
      label: 'Handoff notification chat ID',
      placeholder: 'ID группы для уведомлений операторам',
      hint: 'Числовой chat_id Telegram-группы операторов. Куда отправлять уведомления о новых эскалациях. Добавьте бота в группу и получите id через @userinfobot или getUpdates.',
    },
  ],
  max: [
    {
      key: 'webhook_secret',
      label: 'Webhook secret',
      placeholder: 'Секрет для X-Max-Bot-Api-Secret',
      hint: 'Секрет для проверки webhook MAX. Должен совпадать с тем, что указан при регистрации webhook у провайдера MAX.',
    },
    {
      key: 'api_base',
      label: 'API base URL',
      placeholder: 'https://botapi.max.ru',
      hint: 'Базовый URL Bot API MAX. Обычно https://botapi.max.ru — уточните в документации вашего провайдера MAX.',
    },
    {
      key: 'webhook_registration_url',
      label: 'Webhook registration URL',
      placeholder: 'Endpoint MAX для автоматической регистрации',
      hint: 'URL для автоматической регистрации webhook после сохранения секрета. Если пусто — webhook нужно зарегистрировать вручную в кабинете MAX.',
    },
  ],
  vk: [
    {
      key: 'secret',
      label: 'Callback secret',
      placeholder: 'Секрет Callback API',
      hint: 'Строка «Секретный ключ» из настроек Callback API сообщества VK → Управление → Работа с API.',
    },
    {
      key: 'confirmation_token',
      label: 'Confirmation token',
      placeholder: 'Строка подтверждения VK',
      hint: 'Строка подтверждения, которую VK показывает при первичной настройке Callback API. Сервер должен вернуть её на confirmation-запрос.',
    },
    {
      key: 'group_id',
      label: 'Group ID',
      placeholder: 'ID сообщества',
      hint: 'Числовой ID сообщества VK (без минуса). Находится в адресе группы или в настройках сообщества.',
    },
    {
      key: 'webhook_registration_url',
      label: 'Webhook registration URL',
      placeholder: 'Endpoint VK/прокси для автоматической регистрации',
      hint: 'Опциональный URL для автоматической регистрации webhook. Если не задан — укажите Webhook URL вручную в Callback API VK.',
    },
  ],
};

const SERVICE_SECRET_HINTS: Partial<Record<SecretKind, string>> = {
  llm: 'IAM-токен или API-ключ сервисного аккаунта Yandex Cloud для YandexGPT. Где взять: console.cloud.yandex.ru → Сервисные аккаунты → Создать API-ключ или получить IAM-токен. Без этого секрета бот не сможет генерировать ответы.',
  bitrix24: 'Webhook URL или токен входящего webhook Bitrix24 для интеграции CRM. Где взять: Bitrix24 → Приложения → Webhooks → Входящий webhook. Нужен только если включён модуль bitrix24.',
};

const CHANNEL_HINTS = {
  enabled:
    'Разрешает боту отвечать в этом канале. Требует включённого бота в основных настройках и сохранённого токена.',
  webhook:
    'Публичный HTTPS-URL для приёма сообщений. Укажите его в кабинете Telegram / MAX / VK (Callback API). Должен быть доступен из интернета.',
  token: {
    telegram: 'Токен от @BotFather в Telegram: /newbot → скопируйте строку вида 123456789:AA....',
    max: 'Токен бота MAX из личного кабинета разработчика MAX Bot API.',
    vk: 'Ключ доступа сообщества VK с правами на сообщения. VK → Сообщество → Управление → Работа с API → Ключи доступа.',
  } satisfies Record<ChannelStatus['channel'], string>,
};

const SETTING_HINTS = {
  enabled: 'Главный выключатель бота для компании. Если выключен — ни playground, ни каналы не отвечают.',
  model_tier: 'Лайт — быстрее и дешевле; Про — точнее на сложных вопросах. Влияет на модель YandexGPT по умолчанию.',
  model: 'Переопределение ID модели YandexGPT (например gpt://folder_id/yandexgpt/latest). Пусто — модель по тарифу.',
  temperature: 'Креативность ответов: 0 — строго по фактам, выше — вариативнее. Для support-бота обычно 0.1–0.3.',
  max_tokens: 'Максимальная длина одного ответа в токенах. Больше — длиннее ответы, но дороже запросы.',
  handoff_enabled:
    'Включает передачу диалога оператору: очередь в /bot/handoff, эскалация при пустой базе и по запросу клиента.',
  handoff_text:
    'Сообщение клиенту при эскалации. Показывается вместо ответа бота, когда диалог передаётся оператору.',
  persona_name: 'Имя бота в системном промпте, например «Администратор» или имя вашей компании.',
  persona_tone: 'Стиль общения: friendly, professional и т.д. — попадает в инструкцию модели.',
  persona_rules: 'Дополнительные правила поведения бота: что можно/нельзя отвечать, формат ответов.',
  min_retrieval:
    'Минимальный score RAG-поиска (0–1). Фрагменты ниже порога отбрасываются — бот «не видит» слабые совпадения.',
  min_confidence:
    'Минимальная уверенность ответа (0–1). Ниже порога — отказ или эскалация, если включена.',
  escalate:
    'При низкой уверенности или отсутствии цитат переводит диалог оператору. Пустая база эскалирует и без этой галочки, если handoff включён.',
  citations: 'Требует ссылку на source_id из базы знаний в каждом ответе. Без цитаты — отказ или эскалация.',
};

function FieldHint({
  label,
  htmlFor,
  hint,
  className,
}: {
  label: ReactNode;
  htmlFor?: string;
  hint: string;
  className?: string;
}) {
  return (
    <div className={cn('flex items-center gap-1.5', className)}>
      {htmlFor ? <Label htmlFor={htmlFor}>{label}</Label> : <span className="text-sm font-medium">{label}</span>}
      <Tooltip>
        <TooltipTrigger
          render={
            <button
              type="button"
              className="inline-flex shrink-0 text-muted-foreground transition hover:text-foreground"
              aria-label="Подсказка"
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </button>
          }
        />
        <TooltipContent side="top" className="max-w-sm p-3 text-xs leading-relaxed whitespace-normal">
          {hint}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

function channelModules(enabledModules?: Record<string, unknown>) {
  const raw = enabledModules?.channels;
  if (typeof raw === 'boolean') {
    return { telegram: raw, max: raw, vk: raw };
  }
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    const data = raw as Record<string, unknown>;
    return {
      telegram: data.telegram === true,
      max: data.max === true,
      vk: data.vk === true,
    };
  }
  return { telegram: false, max: false, vk: false };
}

function handoffModule(enabledModules?: Record<string, unknown>) {
  return enabledModules?.handoff === true;
}

function handoffFallbackText(enabledModules?: Record<string, unknown>) {
  const value = enabledModules?.handoff_fallback_text;
  return typeof value === 'string' ? value : '';
}

function SettingsTab() {
  const { data, isLoading, isError } = useBotSettings();
  const update = useUpdateBotSettings();
  const [form, setForm] = useState<Partial<BotSettings>>({});

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  if (isLoading) return <LoadingState />;
  if (isError || !data) return <ErrorState message="Не удалось загрузить настройки бота." />;

  const save = () => {
    update.mutate(
      {
        enabled: form.enabled,
        provider: form.provider,
        model_tier: form.model_tier,
        model: form.model || undefined,
        temperature: form.temperature,
        max_tokens: form.max_tokens,
        persona_name: form.persona_name,
        persona_tone: form.persona_tone,
        persona_rules: form.persona_rules,
        enabled_modules: form.enabled_modules,
        min_retrieval_score: form.min_retrieval_score,
        min_confidence: form.min_confidence,
        escalate_on_low_confidence: form.escalate_on_low_confidence,
        require_citations: form.require_citations,
        allowed_theme_ids: Array.isArray(form.allowed_theme_ids) ? form.allowed_theme_ids : [],
      },
      {
        onSuccess: () => toast.success('Настройки сохранены'),
        onError: () => toast.error('Не удалось сохранить настройки'),
      }
    );
  };

  const setHandoffEnabled = (enabled: boolean) => {
    setForm((f) => ({
      ...f,
      enabled_modules: {
        ...(f.enabled_modules ?? {}),
        handoff: enabled,
      },
    }));
  };

  const setHandoffText = (text: string) => {
    setForm((f) => ({
      ...f,
      enabled_modules: {
        ...(f.enabled_modules ?? {}),
        handoff_fallback_text: text,
      },
    }));
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Основные</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <FieldHint label="Бот включён" hint={SETTING_HINTS.enabled} />
            <Select
              value={form.enabled ? 'true' : 'false'}
              onValueChange={(v) => setForm((f) => ({ ...f, enabled: v === 'true' }))}
            >
              <SelectTrigger className="w-32">
                <SelectValue>
                  {form.enabled ? BOOL_LABELS.true : BOOL_LABELS.false}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="true">{BOOL_LABELS.true}</SelectItem>
                <SelectItem value="false">{BOOL_LABELS.false}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <FieldHint label="Тариф модели" hint={SETTING_HINTS.model_tier} />
              <Select
                value={form.model_tier ?? 'lite'}
                onValueChange={(v) => setForm((f) => ({ ...f, model_tier: v as BotSettings['model_tier'] }))}
              >
                <SelectTrigger>
                  <SelectValue>
                    {MODEL_TIER_LABELS[form.model_tier ?? 'lite']}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="lite">{MODEL_TIER_LABELS.lite}</SelectItem>
                  <SelectItem value="pro">{MODEL_TIER_LABELS.pro}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <FieldHint label="Модель" htmlFor="model" hint={SETTING_HINTS.model} />
              <Input
                id="model"
                value={form.model ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, model: e.target.value }))}
                placeholder="По умолчанию"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <FieldHint label="Температура" htmlFor="temperature" hint={SETTING_HINTS.temperature} />
              <NumberInput
                id="temperature"
                step={0.1}
                min={0}
                max={2}
                value={form.temperature ?? 0.2}
                onChange={(temperature) => setForm((f) => ({ ...f, temperature }))}
              />
            </div>
            <div className="space-y-2">
              <FieldHint label="Макс. токенов" htmlFor="max_tokens" hint={SETTING_HINTS.max_tokens} />
              <NumberInput
                id="max_tokens"
                min={1}
                max={8192}
                value={form.max_tokens ?? 1024}
                onChange={(max_tokens) => setForm((f) => ({ ...f, max_tokens }))}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Handoff оператору</CardTitle>
          <CardDescription>Очередь эскалаций и ручной перехват диалогов оператором</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <FieldHint label="Модуль handoff включён" htmlFor="handoff-enabled" hint={SETTING_HINTS.handoff_enabled} />
            <input
              id="handoff-enabled"
              type="checkbox"
              checked={handoffModule(form.enabled_modules)}
              onChange={(e) => setHandoffEnabled(e.target.checked)}
            />
          </div>
          <div className="space-y-2">
            <FieldHint label="Текст при передаче оператору" htmlFor="handoff-text" hint={SETTING_HINTS.handoff_text} />
            <Textarea
              id="handoff-text"
              value={handoffFallbackText(form.enabled_modules)}
              onChange={(e) => setHandoffText(e.target.value)}
              placeholder="Передаю ваш вопрос специалисту — он свяжется с вами по этому обращению."
            />
            <p className="text-xs text-muted-foreground">
              Handoff включается только при сохранении настроек. Уведомления в Telegram-группу задаются в параметрах Telegram-канала.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Персона</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <FieldHint label="Имя" htmlFor="persona_name" hint={SETTING_HINTS.persona_name} />
            <Input
              id="persona_name"
              value={form.persona_name ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, persona_name: e.target.value }))}
            />
          </div>
          <div className="space-y-2">
            <FieldHint label="Тон" htmlFor="persona_tone" hint={SETTING_HINTS.persona_tone} />
            <Input
              id="persona_tone"
              value={form.persona_tone ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, persona_tone: e.target.value }))}
            />
          </div>
          <div className="space-y-2">
            <FieldHint label="Правила" htmlFor="persona_rules" hint={SETTING_HINTS.persona_rules} />
            <Textarea
              id="persona_rules"
              value={form.persona_rules ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, persona_rules: e.target.value }))}
              rows={4}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Ограничения</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <FieldHint label="Мин. оценка поиска" htmlFor="min_retrieval" hint={SETTING_HINTS.min_retrieval} />
              <NumberInput
                id="min_retrieval"
                step={0.01}
                min={0}
                max={1}
                value={form.min_retrieval_score ?? 0}
                onChange={(min_retrieval_score) => setForm((f) => ({ ...f, min_retrieval_score }))}
              />
            </div>
            <div className="space-y-2">
              <FieldHint label="Мин. уверенность" htmlFor="min_confidence" hint={SETTING_HINTS.min_confidence} />
              <NumberInput
                id="min_confidence"
                step={0.01}
                min={0}
                max={1}
                value={form.min_confidence ?? 0}
                onChange={(min_confidence) => setForm((f) => ({ ...f, min_confidence }))}
              />
            </div>
          </div>
          <div className="flex flex-wrap gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.escalate_on_low_confidence ?? false}
                onChange={(e) => setForm((f) => ({ ...f, escalate_on_low_confidence: e.target.checked }))}
              />
              <FieldHint label="Эскалация при низкой уверенности" hint={SETTING_HINTS.escalate} />
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.require_citations ?? false}
                onChange={(e) => setForm((f) => ({ ...f, require_citations: e.target.checked }))}
              />
              <FieldHint label="Требовать цитирование" hint={SETTING_HINTS.citations} />
            </label>
          </div>
        </CardContent>
      </Card>

      <Button onClick={save} disabled={update.isPending}>
        {update.isPending ? 'Сохранение...' : 'Сохранить настройки'}
      </Button>
    </div>
  );
}

const CHANNEL_SECRET_KINDS: SecretKind[] = ['telegram', 'max', 'vk'];
const SERVICE_SECRET_KINDS: SecretKind[] = ['llm', 'bitrix24'];

function ChannelsAndSecretsTab() {
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

function RagTab() {
  const { data: status, isLoading } = useRagIndexStatus();
  const reindex = useRagReindex();
  const search = useRagSearch();
  const [query, setQuery] = useState('');

  const runSearch = () => {
    if (!query.trim()) return;
    search.mutate({ query: query.trim(), hybrid_top_k: 10 });
  };

  return (
    <div className="space-y-6 max-w-3xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Индексация</CardTitle>
          <CardDescription>Статус векторного индекса базы знаний</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <LoadingState />
          ) : status ? (
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">Чанков в индексе</p>
                <p className="text-2xl font-semibold">{status.embeddings}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Задачи в очереди</p>
                <div className="mt-1 space-y-1">
                  {Object.entries(status.jobs ?? {}).map(([k, v]) => (
                    <p key={k}>{k}: {v}</p>
                  ))}
                </div>
              </div>
            </div>
          ) : null}
          <Button
            variant="outline"
            onClick={() => reindex.mutate(undefined, {
              onSuccess: () => toast.success('Переиндексация поставлена в очередь'),
              onError: () => toast.error('Не удалось запустить переиндексацию'),
            })}
            disabled={reindex.isPending}
          >
            {reindex.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <RefreshCw className="h-4 w-4 mr-2" />}
            Переиндексировать
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Отладочный поиск</CardTitle>
          <CardDescription>RAG-поиск по индексу (для администраторов)</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Запрос..."
              onKeyDown={(e) => e.key === 'Enter' && runSearch()}
            />
            <Button onClick={runSearch} disabled={search.isPending || !query.trim()}>
              {search.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Искать'}
            </Button>
          </div>
          {search.data?.results && search.data.results.length > 0 && (
            <div className="space-y-2">
              {search.data.results.map((r) => (
                <div key={r.source_id} className="border rounded-md p-3 text-sm">
                  <div className="flex items-center gap-2 mb-1">
                    <Badge variant="outline">{r.entity_type}</Badge>
                    <span className="font-medium">{r.title}</span>
                    <span className="text-muted-foreground ml-auto">оценка {r.score.toFixed(3)}</span>
                  </div>
                  <p className="text-muted-foreground line-clamp-2">{r.snippet || r.content}</p>
                </div>
              ))}
            </div>
          )}
          {search.data?.results?.length === 0 && (
            <p className="text-sm text-muted-foreground">Ничего не найдено.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export function BotAdminPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Администрирование бота</h1>
      <Tabs defaultValue="settings">
        <TabsList>
          <TabsTrigger value="settings">Настройки</TabsTrigger>
          <TabsTrigger value="channels">Каналы и секреты</TabsTrigger>
          <TabsTrigger value="rag">База знаний</TabsTrigger>
        </TabsList>
        <TabsContent value="settings" className="mt-4">
          <SettingsTab />
        </TabsContent>
        <TabsContent value="channels" className="mt-4">
          <ChannelsAndSecretsTab />
        </TabsContent>
        <TabsContent value="rag" className="mt-4">
          <RagTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

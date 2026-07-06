import type { BotSettings, ChannelStatus, SecretKind } from '@/types';

export const BOOL_LABELS = { true: 'Да', false: 'Нет' } as const;

export const MODEL_TIER_LABELS: Record<BotSettings['model_tier'], string> = {
  lite: 'Лайт',
  pro: 'Про',
};

export const SECRET_LABELS: Record<SecretKind, string> = {
  llm: 'Yandex API',
  telegram: 'Telegram',
  max: 'MAX',
  vk: 'ВКонтакте',
  bitrix24: 'Битрикс24',
};

export const CHANNEL_LABELS: Record<ChannelStatus['channel'], string> = {
  telegram: 'Telegram',
  max: 'MAX',
  vk: 'ВКонтакте',
};

export const CHANNEL_METADATA_FIELDS: Record<
  ChannelStatus['channel'],
  Array<{ key: string; label: string; placeholder?: string; hint: string; required?: boolean }>
> = {
  telegram: [
    {
      key: 'webhook_secret',
      label: 'Webhook secret token',
      placeholder: 'Секрет для X-Telegram-Bot-Api-Secret-Token',
      hint: 'Произвольная строка для проверки входящих webhook. Задайте её здесь и укажите тот же secret при регистрации webhook в Bot API (setWebhook). Telegram присылает её в заголовке X-Telegram-Bot-Api-Secret-Token.',
      required: true,
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
      required: true,
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
      required: true,
    },
    {
      key: 'confirmation_token',
      label: 'Confirmation token',
      placeholder: 'Строка подтверждения VK',
      hint: 'Строка подтверждения, которую VK показывает при первичной настройке Callback API. Сервер должен вернуть её на confirmation-запрос.',
      required: true,
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

export const SERVICE_SECRET_HINTS: Partial<Record<SecretKind, string>> = {
  llm: 'IAM-токен или API-ключ сервисного аккаунта Yandex Cloud для YandexGPT. Где взять: console.cloud.yandex.ru → Сервисные аккаунты → Создать API-ключ или получить IAM-токен. Без этого секрета бот не сможет генерировать ответы.',
  bitrix24: 'Webhook URL или токен входящего webhook Bitrix24 для интеграции CRM. Где взять: Bitrix24 → Приложения → Webhooks → Входящий webhook. Нужен только если включён модуль bitrix24.',
};

export const CHANNEL_HINTS = {
  enabled:
    'Разрешает боту отвечать в этом канале. Требует включённого бота в основных настройках и сохранённого токена.',
  webhook:
    'Публичный HTTPS-URL для приёма сообщений. Для Telegram сервер также сверяет текущий webhook в Bot API с этим адресом.',
  token: {
    telegram: 'Токен от @BotFather в Telegram: /newbot → скопируйте строку вида 123456789:AA....',
    max: 'Токен бота MAX из личного кабинета разработчика MAX Bot API.',
    vk: 'Ключ доступа сообщества VK с правами на сообщения. VK → Сообщество → Управление → Работа с API → Ключи доступа.',
  } satisfies Record<ChannelStatus['channel'], string>,
};

export const SETTING_HINTS = {
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

// MASKED_SECRET_VALUE mirrors the backend sentinel used for sensitive metadata
// (webhook secrets, callback secrets, confirmation tokens). It is display-only
// and must never be persisted back as a real value.
export const MASKED_SECRET_VALUE = '********';

export const CHANNEL_SECRET_KINDS: SecretKind[] = ['telegram', 'max', 'vk'];
export const SERVICE_SECRET_KINDS: SecretKind[] = ['llm', 'bitrix24'];

const SENSITIVE_METADATA_KEY_MARKERS = ['secret', 'password', 'token', 'api_key', 'access_key', 'private'] as const;

/** Mirrors backend isSensitiveMetadataKey — used for password inputs in the channel editor. */
export function isSensitiveMetadataField(key: string): boolean {
  const lower = key.toLowerCase().trim();
  return SENSITIVE_METADATA_KEY_MARKERS.some((marker) => lower.includes(marker));
}

export function canGenerateWebhookSecret(channel: ChannelStatus['channel'], fieldKey: string): boolean {
  return fieldKey === 'webhook_secret' && (channel === 'telegram' || channel === 'max');
}

export function generateWebhookSecret(byteLength = 32): string {
  const bytes = new Uint8Array(byteLength);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}


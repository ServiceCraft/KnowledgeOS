# Каналы Telegram, MAX и VK

Модуль каналов принимает публичные webhooks без JWT и проверяет каждый запрос
секретом из `tenant_secrets.metadata`. Токены ботов хранятся только в
зашифрованном `tenant_secrets.value`.

## Webhook URLs

URL формируется per tenant:

```text
/api/v1/webhooks/telegram/{company_id}
/api/v1/webhooks/max/{company_id}
/api/v1/webhooks/vk/{company_id}
```

В продакшне эти URL должны быть доступны по HTTPS.

## Metadata

Telegram:

```json
{
  "webhook_secret": "secret-token-for-X-Telegram-Bot-Api-Secret-Token"
}
```

MAX:

```json
{
  "webhook_secret": "secret-token-for-X-Max-Bot-Api-Secret",
  "api_base": "https://botapi.max.ru"
}
```

VK Callback API:

```json
{
  "secret": "callback-secret",
  "confirmation_token": "vk-confirmation-string",
  "group_id": "123456"
}
```

## Runtime Behavior

- Канал обрабатывает только текстовые сообщения в MVP.
- Входящий chat id связывается с `chat_sessions.external_chat_id`.
- Автоответы отправляются только если `bot_settings.enabled = true` и
  `enabled_modules.channels.{telegram|max|vk} = true`.
- Если session уже не в состоянии `bot`, webhook подтверждается, но бот не
  отвечает: это оставляет место для handoff-модуля.

## Автоматическая регистрация webhook

После сохранения channel secret через `PUT /admin/bot/secrets/{kind}` backend
пытается автоматически зарегистрировать webhook, если запрос пришёл через
публичный HTTPS host и для канала хватает metadata.

Telegram регистрируется через Bot API `setWebhook` автоматически, если заданы:

```json
{
  "webhook_secret": "secret-token-for-X-Telegram-Bot-Api-Secret-Token"
}
```

MAX и VK API отличаются по способу регистрации. Для них backend делает
автоматическую регистрацию только если в metadata указан
`webhook_registration_url`; на этот endpoint отправляется HTTPS POST с URL
webhook и секретом канала.

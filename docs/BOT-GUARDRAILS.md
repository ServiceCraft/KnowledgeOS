# Bot Guardrails (Б5)

Этот документ описывает слой защиты от галлюцinaций «Текстовой линзы», реализованный в Б5 поверх ядра чата (Б3) и tool-loop (Б4). Принцип — **defense in depth, от дешёвого к дорогому**: сначала детерминированные гейты (ноль токенов), затем LLM, затем постпроверка результата.

## Конвейер ответа

```
user turn
  -> RAG prefetch (Б2 hybrid retriever)
  -> deterministic gates: min_retrieval_score, allowed_theme_ids
  -> [нет контекста и нет tools] -> штатный отказ (LLM не вызывается)
  -> LLM / tool-loop (Б4)
  -> post-check: prompt-leak filter -> citation validation -> confidence
  -> answer | refuse | escalate  (сохраняется в chat_messages)
```

Код: [backend/internal/service/chat.go](backend/internal/service/chat.go) (`generate`, `applyGuardrails`, `runToolLoop`) и чистые функции в [backend/internal/chat/guardrails/guardrails.go](backend/internal/chat/guardrails/guardrails.go).

## Гейты (детерминированные, до LLM)

- **Гейт «нет контекста»** — если ретривер не дал источников и инструменты недоступны, бот отказывает без вызова LLM. С инструментами модель может сама вызвать `search_knowledge`, поэтому LLM вызывается, но финал всё равно проходит постпроверку.
- **`min_retrieval_score`** — отбрасывает кандидатов ниже порога релевантности. `0` (по умолчанию) отключает гейт.
- **`allowed_theme_ids`** — ограничивает тематику. Кандидаты без темы пропускаются (как в kb_embedding-фильтре). Пустой список — без ограничения.

## Grounding через проверяемое цитирование

- Системный промпт делимитирует контекст тегами `<context>...</context>`, объявляет его данными (а не инструкциями) и просит модель указывать `source_id` в квадратных скобках, например `[qa:UUID:0]`.
- **Постпроверка**: каждый процитированный `source_id` обязан входить в множество реально переданных источников. **Выдуманная ссылка** (`fabricated_citation`) → ответ бракуется, клиенту уходит отказ. Это всегда включено.
- Валидные цитаты сохраняются в `chat_messages.cited_source_ids`, источники — в `chat_messages.sources`.

## Скоринг уверенности

Прозрачная эвристика (`guardrails.Confidence`) из имеющихся сигналов: наличие grounded-контекста, валидные цитаты, число источников; любая выдуманная ссылка обнуляет уверенность. Значение сохраняется в `chat_messages.confidence_score` (0..1).

- `min_confidence` > 0 и `confidence < min_confidence` → отказ/эскалация (`low_confidence`).
- `require_citations` = true и нет валидной цитаты при наличии источников → отказ/эскалация (`missing_citation`).
- По умолчанию оба порога `0`/`false` (опциональные гейты выключены, базовые проверки активны).

## Защита от prompt-injection

- Контекст и пользовательский ввод объявлены данными; инструкции внутри них игнорируются.
- Системный промпт/персона неизменяемы содержимым запроса или выдачей ретривера.
- **Фильтр на выходе** (`guardrails.LeaksSystemPrompt`): если ответ содержит маркеры системного промпта/служебки, ответ бракуется (`prompt_leak`).

## Отказ и эскалация

Отказ — **штатный путь**, успешное сообщение ассистента, а не HTTP-ошибка. Решение хранится в `chat_messages.guardrail_action`:

| action     | когда                                                        |
|------------|-------------------------------------------------------------|
| `answer`   | grounded-ответ прошёл все проверки                          |
| `refuse`   | вежливый отказ (нет контекста, низкая уверенность, брак)    |
| `escalate` | то же, но с сигналом передать диалог оператору             |

`escalate` формируется, когда включён `escalate_on_low_confidence`, для легитимных «не могу из базы» причин (`no_context`, `low_confidence`, `missing_citation`). Защитные брейки (`prompt_leak`, `fabricated_citation`) остаются `refuse`.

**Граница Б5:** статусы сессии `waiting_operator`/`operator` заведены в схеме, но Б5 их не переключает — это зона модуля handoff. Б5 даёт только машиночитаемый сигнал `guardrail_action: escalate`, чтобы handoff подключился без слома контракта.

## Настройки (per-tenant)

`bot_settings` (см. `PATCH /admin/bot/settings`):

| поле                         | тип        | по умолчанию | смысл                                            |
|------------------------------|------------|--------------|--------------------------------------------------|
| `min_retrieval_score`        | number     | `0`          | порог релевантности RAG                          |
| `min_confidence`             | number 0..1| `0`          | порог уверенности для авто-отказа                |
| `allowed_theme_ids`          | uuid[]     | `[]`         | разрешённые темы                                 |
| `escalate_on_low_confidence` | bool       | `false`      | эскалировать вместо простого отказа              |
| `require_citations`          | bool       | `false`      | требовать валидную цитату                        |

## Логирование и ПДн (152-ФЗ, технические меры)

- **`x-data-logging-enabled: false`** на вызовах YandexGPT — отключает логирование запросов на стороне провайдера (покрыто тестом в [backend/internal/llm/yandex/client_test.go](backend/internal/llm/yandex/client_test.go)).
- **Turn-level логи** (`ChatService.logTurn`) пишут только метаданные: `company_id`, `session_id`, `guardrail_action`, `refusal_reason`, `confidence`, `source_count`, `tokens`, `used_tools`. **Тексты сообщений, промпты и сырые tool-payload с пользовательским вводом не логируются.**
- **Маскирование**: значения, которые всё же могут попасть в диагностическое поле (например, текст ошибки инструмента), проходят через [backend/internal/privacy/redact.go](backend/internal/privacy/redact.go) — телефоны и email заменяются на `[phone]`/`[email]`.
- Секреты тенанта хранятся только в зашифрованном виде (AES-GCM, Б1).

## Eval-харнесс (артефакты приёмки)

Прогон по двум наборам — внутридоменные вопросы (ожидаем ответ с валидным источником) и провокационные/внедоменные (ожидаем отказ/эскалацию). См. [backend/internal/chat/eval](backend/internal/chat/eval) и отчёт в [docs/eval](docs/eval). Метрики: grounding rate, refusal accuracy, escalation accuracy, invalid-citation count. Харнесс служит и регрессионной защитой при смене промпта/модели.

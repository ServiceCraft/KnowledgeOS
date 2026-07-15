# План: интеграция YClients — запись клиентов агентом

**Статус:** черновик плана, к реализации не приступали
**Ветка-основа:** `B-1`
**Цель:** дать агенту инструменты для записи клиентов в YClients (онлайн-запись), с тумблером вкл/выкл на компанию, и заменить блок «Битрикс24» на «YClients» на странице настроек интеграций.

---

## 1. Решения (зафиксированы)

| Вопрос | Решение |
|---|---|
| Метод записи | **Онлайн-запись** `POST /book_record/{company_id}` — нужен только партнёрский токен + `company_id`. Токен не протухает, логин админа не нужен. Ограничение: зависит от настроек онлайн-записи филиала. |
| Объём инструментов | **Полный флоу — 4 инструмента**: агент сам смотрит услуги, мастеров, свободные слоты и создаёт запись. ID берёт из live-данных YClients, не выдумывает. |
| Битрикс24 | Убираем из UI и заменяем на YClients. На бэке это заглушка (secret kind + UI, без потребляющей логики). |

---

## 2. Что уже есть в коде (итоги разбора)

- **Инструменты агента** реализуют интерфейс `Tool` — `backend/internal/chat/tools/tool.go:25`:
  `Name() / Description() / Schema() json.RawMessage / Execute(ctx, companyID uuid.UUID, args json.RawMessage) (Result, error)`.
  Регистрируются в `tools.NewRegistry(...)` — `backend/internal/app/app.go:121`. `companyID` уже прокинут в каждый `Execute`.
- **Диспетчер/валидация**: `dispatcher.go:26` (`Registry.Execute`) валидирует аргументы против схемы (`validate.go`, самописный сабсет JSON-Schema) до `Execute`.
- **Цикл вызова инструментов**: `service/chat.go` — `toolDefinitions()` (`chat.go:89`) отдаёт `registry.Definitions()`; `runToolLoop` (`chat.go:541`) прогоняет вызовы. **Важно:** сейчас все инструменты выдаются всем компаниям — фильтрации по настройкам нет.
- **Тумблеры фич** уже живут в `BotSettings.EnabledModules` (jsonb) — `domain/bot.go:34`. Образцы чтения флага: `handoffEnabled` (`service/handoff.go:251`), `channelEnabled` (`channels/gateway.go:490`). Пишутся через `UpdateBotSettingsRequest.EnabledModules` (`service/bot_settings.go`).
- **Секреты** — generic по `SecretKind`:
  - Модель/валидатор: `domain/bot.go:24` (`SecretKind`, `ValidSecretKind`, `SupportedSecretKinds`), сейчас `llm, telegram, max, vk, bitrix24`.
  - Сервис: `service/tenant_secret.go` — `Set/GetEditable/ListStatus/Delete`, внутреннее чтение `GetPlaintext` / `GetPlaintextWithMetadata` (токен + незамаскированная metadata). Sensitive-ключи metadata автомаскируются (`isSensitiveMetadataKey`).
  - API: `router.go:180` — `GET /admin/bot/secrets`, `GET .../{kind}/edit`, `PUT .../{kind}`, `DELETE .../{kind}`. Роуты kind-агностичны.
  - CHECK-констрейнт на `kind` — `backend/migrations/012_tenant_secrets.sql:6` (единственное место, где новый kind будет отвергнут без миграции).
- **Фронт настроек**: `frontend/src/components/bot-admin/ChannelsAndSecretsTab.tsx` — две секции: «Сервисные секреты» (value-only редактор) и «Каналы связи» (token + metadata-поля). Константы в `constants.ts`. Тумблеры модулей — `SettingsTab.tsx`.
- **Битрикс24 сейчас**: `SecretKind bitrix24` + метки/хинты + `SERVICE_SECRET_KINDS` — чистая заглушка, потребителя нет.

---

## 3. Объём работ

### A. Backend — новый secret kind + конфиг
1. `backend/internal/domain/bot.go`: добавить `SecretKindYClients SecretKind = "yclients"` в константы, в `ValidSecretKind`, в `SupportedSecretKinds`; убрать `SecretKindBitrix24` из тех же трёх мест.
2. Новая миграция `backend/migrations/022_yclients_secret_kind.sql`:
   - `DELETE FROM tenant_secrets WHERE kind = 'bitrix24';` (заглушка, данных быть не должно).
   - Пересоздать CHECK: `kind IN ('llm','telegram','max','vk','yclients')`.
3. Формат секрета:
   - `value` = **партнёрский токен** (sensitive, автомаскируется).
   - `metadata` = `{ "company_id": "<id филиала>", "api_base": "https://api.yclients.com/api/v1" }` (`api_base` опционально, дефолт зашит в клиент).

### B. Backend — HTTP-клиент YClients
Новый пакет `backend/internal/integrations/yclients/`:
- `client.go`: `Client{ partnerToken, baseURL, http *http.Client }`.
  Заголовки: `Authorization: Bearer {partner}`, `Accept: application/vnd.api.v2+json`, `Content-Type: application/json`.
- Методы:
  - `GetServices(ctx, companyID string) ([]Service, error)` → `GET /book_services/{company_id}`.
  - `GetStaff(ctx, companyID string, serviceIDs []int) ([]Staff, error)` → `GET /book_staff/{company_id}`.
  - `GetBookTimes(ctx, companyID string, staffID int, date string) ([]Slot, error)` → `GET /book_times/{company_id}/{staff_id}/{date}`.
  - (опц.) `GetBookDates(...)` → `GET /book_dates/{company_id}`.
  - `CreateBookRecord(ctx, companyID string, req BookRequest) (BookResult, error)` → `POST /book_record/{company_id}`.
    Тело: `{ phone, fullname, email, comment, api_id, appointments:[{ id, services:[serviceID], staff_id, datetime }] }`.
- Утилиты: нормализация телефона к `79991234567`, разбор конверта ответа `{success, data, meta, errors}`, таймаут, читаемые ошибки для агента.
- Тесты `client_test.go` через `httptest` (образец — `backend/internal/llm/yandex/client_test.go`).

### C. Backend — 4 инструмента агента
`backend/internal/chat/tools/yclients.go` (по образцу `get_pricing.go`). Каждый инжектится `TenantSecretService` + фабрикой клиента; в `Execute` читает `GetPlaintextWithMetadata(ctx, companyID, domain.SecretKindYClients)` → партнёрский токен + `company_id`, строит клиент, зовёт API. Если секрет не задан — понятный `Result` агенту («интеграция YClients не настроена»).

| Инструмент | Аргументы (schema) | Действие |
|---|---|---|
| `yclients_get_services` | `{}` | Список услуг: `id`, название, цена, длительность |
| `yclients_get_staff` | `{ service_id?: integer }` | Мастера: `id`, имя, специализация |
| `yclients_get_times` | `{ staff_id: integer (req), date: string YYYY-MM-DD (req) }` | Свободные слоты (ISO8601) |
| `yclients_create_booking` | `{ fullname: string (req), phone: string (req), service_id: integer (req), staff_id: integer (req), datetime: string ISO8601 (req), email?: string, comment?: string }` | Создаёт запись, возвращает `record_id` |

### D. Backend — тумблер per-company
- Ключ `EnabledModules["yclients_booking"]` (bool). Хелпер `yclientsBookingEnabled(raw json.RawMessage) bool` по образцу `handoffEnabled`.
- Сделать выдачу инструментов настройко-зависимой:
  - Карта `moduleGatedTools = map[string]string{ "yclients_get_services":"yclients_booking", ... }` (все 4).
  - `toolDefinitions(settings)` отфильтровывает инструменты с выключенным модулем (в `chat.go`/`generate` настройки уже загружены).
  - Защитная проверка в `executeToolCall` — если модуль выключен, вернуть ошибку-Result (по образцу гейта `request_handoff`, `chat.go:585`).

### E. Frontend — UI настроек
1. `frontend/src/types/models.ts:251`: в union `SecretKind` добавить `'yclients'`, убрать `'bitrix24'`.
2. `frontend/src/components/bot-admin/constants.ts`:
   - `SECRET_LABELS.yclients = 'YClients'` (убрать `bitrix24`).
   - Хинт в `SERVICE_SECRET_HINTS.yclients` (что вводить: партнёрский токен + ID филиала).
   - В `SERVICE_SECRET_KINDS` заменить `'bitrix24'` → `'yclients'`.
3. Расширить редактор **сервисных** секретов (`ChannelsAndSecretsTab.tsx:291-319`) — сейчас он только `value`. Добавить `SERVICE_SECRET_METADATA_FIELDS` (по аналогии с `CHANNEL_METADATA_FIELDS`) и отрисовку metadata-полей в диалоге. Для YClients: партнёрский токен (`value`) + `company_id` (metadata, required).
4. Тумблер в `SettingsTab.tsx`: свитч «Запись в YClients» → флипает `enabled_modules.yclients_booking` (по образцу переключателя handoff).

### F. Wiring
- `backend/internal/app/app.go`: собрать фабрику YClients-клиента, прокинуть `tenantSecretSvc` в новые tools, зарегистрировать в `NewRegistry(...)`, сделать `toolDefinitions` настройко-зависимым.

### G. Тесты
- Клиент YClients (httptest).
- Инструменты: успешный путь, отсутствие секрета, ошибка API.
- Валидация схем аргументов.
- Гейтинг тумблером (выключено → инструмент не выдаётся и не исполняется).
- Маскирование партнёрского токена на публичных ручках.

---

## 4. Риски и нюансы

- **Онлайн-запись зависит от настроек филиала** в YClients (должна быть включена онлайн-запись, у услуг/мастеров — доступность онлайн). Ошибки API пробрасываем агенту читаемым текстом.
- **Подтверждение перед записью**: в системный промпт — указание проговорить детали (услуга/мастер/время/имя/телефон) и получить подтверждение клиента до `yclients_create_booking`.
- **Дубли**: `appointments[].id` / `api_id` для идемпотентности повторных вызовов.
- **Таймзона/формат datetime**: слоты агент берёт только из `yclients_get_times`, не выдумывает.
- **Миграция bitrix24**: перед сменой CHECK удалить существующие `bitrix24`-строки (иначе констрейнт упадёт).

---

## 5. Порядок реализации

1. Secret kind + миграция (A).
2. HTTP-клиент YClients + тесты (B).
3. Инструменты агента + wiring (C, F).
4. Тумблер per-company (D).
5. Фронт: секрет-блок + metadata-поля + тумблер (E).
6. Тесты и прогон линтеров/`go test`, `npm run build` (G).

Ветка: `feature/yclients-booking` от `B-1`.

---

## 6. Затрагиваемые файлы (чек-лист)

**Backend**
- `backend/internal/domain/bot.go` — kind
- `backend/migrations/022_yclients_secret_kind.sql` — новый
- `backend/internal/integrations/yclients/client.go` (+ `client_test.go`) — новый пакет
- `backend/internal/chat/tools/yclients.go` (+ тест) — новые инструменты
- `backend/internal/service/chat.go` — настройко-зависимая выдача/гейтинг инструментов
- `backend/internal/service/bot_settings.go` / хелпер модуля — при необходимости
- `backend/internal/app/app.go` — wiring

**Frontend**
- `frontend/src/types/models.ts` — union kind
- `frontend/src/components/bot-admin/constants.ts` — метки/хинты/kinds/metadata-поля
- `frontend/src/components/bot-admin/ChannelsAndSecretsTab.tsx` — редактор сервисного секрета с metadata
- `frontend/src/components/bot-admin/SettingsTab.tsx` — тумблер модуля

Никаких изменений не нужно в: `api/botAdmin.ts`, `hooks/useBotAdmin.ts`, `handler/bot.go`, `router.go`, `service/tenant_secret.go` — они kind-агностичны.

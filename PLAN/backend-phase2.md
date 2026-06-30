# Backend Phase 2: оставшийся backlog

> Перепроверка кодовой базы: 2026-06-30. Phase 1 (auth/tenant P0–P8) выполнен.

## Phase 1 — выполнено (не трогать)

| Область | Статус |
|---------|--------|
| `domain/errors`, store `mapErr`, service без GORM | ✅ |
| `CreateWithCompanies`, `issueSession`, `loadInCompany` (1 query) | ✅ |
| `cache.Provider` (in-memory), Tenant + superadmin company check | ✅ |
| Migrations 019–020, JWT `mv`, убран `CompanyID` | ✅ |
| `go-envconfig`, pg pool/SSL, httprate, healthz/readyz, graceful shutdown | ✅ |
| `internal/app`, `TokenRepository`, `TenantContext` | ✅ |
| Table-driven tests: tenant, auth, user, admin, cache | ✅ (частичное покрытие auth) |
| `cmd/rag-worker`, `cmd/migrate` (обёртка) | ✅ stub |

---

## Phase 2A — Production hardening (высокий приоритет)

### A1. Membership cache по `mv`

**Проблема:** Tenant middleware всегда вызывает `GetCompanyIDs`, хотя JWT уже несёт `mv`.

**Задачи:**
- In-memory `MembershipCache`: `userID → {companyIDs, mv}`
- Tenant: если `claims.mv == cached.mv` → skip DB; иначе reload + update cache
- Invalidate при `SetCompanyIDs` / role change
- Table-driven tests с mock DB call counter

### A2. Migrations off API startup

**Проблема:** `app.New` и legacy `RunMigrations` (up-only) на каждом старте API.

**Задачи:**
1. Убрать migrate из `app.New`; readyz проверяет schema version
2. `golang-migrate/v4`: baseline + up/down; `cmd/migrate` через embed/file
3. docker-compose: service `migrate` before `app`

### A3. RAG worker ops

**Проблема:** `cmd/rag-worker` есть, но не в compose; API дублирует worker.

**Задачи:**
- compose profile `rag`: `rag-worker` + `RAG_WORKER_ENABLED=false` на API
- deploy docs

### A4. Test gaps

Добавить в auth/user tests: inactive login, expired/revoked refresh, superadmin без companies, mergeCompanyIDs scope.

---

## Phase 2B — Data layer & maintainability

### B1. sqlc runtime
- `sqlc generate` в CI/Makefile
- Hot path: `GetCompanyIDs`, `HasCompany` через `internal/store/db`
- GORM остаётся для CRUD

### B2. SyncRepository cleanup
- Убрать token methods из `SyncRepository`
- Router/sync auth — только нужные интерфейсы

### B3. `TenantFromContext` helper
- `service.ActorFromRequest(r)` вместо ручной сборки

### B4. `cmd/seed`
- Вынести bootstrap superadmin из `app.New`

### B5. Export → store
- `ExportService` raw DB → `store/export.go`

### B6. Split `repository.go` (низкий приоритет)

---

## Phase 2C — Scale & observability (2+ replicas)

### C1. Redis `CacheProvider` + global rate limit
### C2. Audit log + Prometheus metrics
### C3. In-process domain events (revoke/cache side effects)

---

## Phase 2D — Async chat (по метрикам)

1. `cmd/chat-worker` + PG job table
2. River queue
3. Kafka (segmentio/kafka-go)

**Out of scope:** Auth microservice, PostgreSQL RLS, OpenAPI (если не нужен для e2e).

---

## Продуктовый вопрос

`attachCompanyIDs` в User List отдаёт все companies user'а — нужно ли фильтровать по текущему tenant?

---

## Порядок

A1 → A2 → A3 → A4 → B1 → B2 → (B3–B6) → C* → D*

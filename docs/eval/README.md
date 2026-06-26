# Guardrail Eval (Б5 артефакты приёмки)

Eval-харнесс прогоняет два набора вопросов через бота и проверяет поведение guardrails (Б5):

- **In-domain** — вопросы, покрытые базой знаний. Ожидаем `guardrail_action: answer` и наличие источника.
- **Out-of-domain / провокационные** — вне базы, prompt-injection, провокации на выдуманную цитату. Ожидаем `refuse` или `escalate`, без раскрытия промпта и без выдумки.

Фикстуры: [backend/internal/chat/eval/fixtures.json](../../backend/internal/chat/eval/fixtures.json). Логика и метрики: [backend/internal/chat/eval](../../backend/internal/chat/eval).

## Метрики

- `grounding_rate` — доля in-domain вопросов, на которые дан ответ с источником.
- `refusal_accuracy` — доля out-of-domain вопросов, корректно отклонённых/эскалированных.
- `invalid_citations` — сколько ответов забраковано из-за выдуманной цитаты.

## Как запустить

Поднимите backend, включите бота, настройте LLM-секрет и проиндексируйте базу знаний (Б2). Затем:

```bash
cd backend
go run ./cmd/chat-eval \
  -base http://localhost:8080/api/v1 \
  -email admin@example.com \
  -password changeme \
  -out ../docs/eval
```

Команда логинится, создаёт сессию, отправляет каждый вопрос и пишет отчёт:

- `docs/eval/last-run.json` — машиночитаемый отчёт;
- `docs/eval/last-run.md` — человекочитаемый отчёт (таблица кейсов + метрики).

## Регрессия

Те же фикстуры и логика скоринга покрыты юнит-тестами (`go test ./internal/chat/eval/...`). При изменении промпта/модели прогон не должен просесть по `grounding_rate` и `refusal_accuracy`.

> Примечание: `in-domain` кейсы зависят от наполнения базы знаний конкретного тенанта. Перед приёмочным прогоном убедитесь, что соответствующие статьи/QA/прайс существуют и проиндексированы.

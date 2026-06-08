# KnowledgeOS — Резервное копирование и восстановление

Система резервного копирования состоит из двух независимых частей:

1. **Эндпоинт-слепок** в основном приложении — `GET /api/v1/backup/snapshot`,
   отдаёт архив с дампом БД, снимком кода и метаданными.
2. **Backup-сервис** — отдельный контейнер, который по расписанию опрашивает
   эндпоинт, выгружает код в GitHub и хранит дампы данных версионно.

---

## 1. Эндпоинт-слепок (основное приложение)

```
GET /api/v1/backup/snapshot
Authorization: Bearer <BACKUP_API_KEY>
```

Возвращает `snapshot-YYYYMMDD-HHMMSS.tar.gz` со следующим содержимым:

| Файл            | Содержимое                                                       |
|-----------------|------------------------------------------------------------------|
| `dump.sql`      | Дамп БД (`pg_dump --serializable-deferrable`)                    |
| `code.tar.gz`   | Снимок исходников проекта (без `.git`, `node_modules`, `vendor`…) |
| `export.json`   | Логический экспорт базы знаний по всем компаниям (то же, что отдаёт кнопка «Экспортировать» / `GET /api/v1/export`) |
| `metadata.json` | Дата (UTC), хэш коммита, версия схемы БД, размеры и SHA-256       |

`export.json` имеет вид `{ "exported_at": …, "companies": [ { "company_id", "company_name", "data": { themes, qa_pairs, pricing_nodes, articles, calls, … } } ] }`,
где блок `data` каждой компании идентичен ответу `GET /api/v1/export`. Это
портативный, пригодный для `POST /api/v1/import` слепок в дополнение к «сырому»
`dump.sql`.

Свойства:

- Защищён API-ключом `BACKUP_API_KEY` (заголовок `Authorization: Bearer`).
  Ключ читается при каждом запросе — чтобы **отозвать ключ без перезапуска**,
  смените/очистите `BACKUP_API_KEY` (пустое значение → эндпоинт отвечает `503`).
- Должен публиковаться **только по HTTPS**. TLS терминируется на Nginx (как и
  для остального API платформы); backup-сервис обращается по `https://…`.
  Сертификаты — per-deployment, в репозиторий не коммитятся: смонтируйте их в
  контейнер Nginx и включите закомментированный `listen 443 ssl` блок в
  [`nginx/nginx.conf`](../nginx/nginx.conf) (либо терминируйте TLS на внешнем
  edge-прокси). Для `/api/v1/backup/snapshot` там же отключена буферизация и
  увеличены таймауты — большой архив отдаётся потоково без `504`.
- Дамп снимается с уровнем согласованности `--serializable-deferrable`.
- Ответ **стримится**; промежуточный архив создаётся во временном каталоге и
  удаляется после отдачи.
- Параллельные вызовы **сериализуются**: пока идёт один `pg_dump`, повторный
  вызов получает `409 Conflict` (двух одновременных дампов не будет).

Переменные окружения основного приложения:

| Переменная           | Назначение                                              | По умолчанию  |
|----------------------|---------------------------------------------------------|---------------|
| `BACKUP_API_KEY`     | Ключ доступа к эндпоинту (пусто → выключен)              | —             |
| `BACKUP_CODE_PATH`   | Каталог исходников для `code.tar.gz`                     | `/app/src`    |
| `BACKUP_GIT_COMMIT`  | Хэш коммита для `metadata.json`                          | из `/app/COMMIT` |
| `BACKUP_COMMIT_FILE` | Файл с хэшем коммита, если `BACKUP_GIT_COMMIT` пуст      | `/app/COMMIT` |

В `docker-compose.yml` репозиторий монтируется в контейнер как `/src` (read-only),
и `BACKUP_CODE_PATH=/src`, поэтому в снимок попадает весь актуальный код проекта.
Коммит передаётся build-аргументом `GIT_COMMIT` (`make up` подставляет его сам).

Проверить вручную:

```bash
curl -fSL -H "Authorization: Bearer $BACKUP_API_KEY" \
  https://<APP_HOST>/api/v1/backup/snapshot -o snapshot.tar.gz
tar tzf snapshot.tar.gz          # metadata.json, dump.sql, code.tar.gz, export.json
```

---

## 2. Backup-сервис

Отдельный контейнер (каталог `backup-service/`). Без UI; конфигурируется через
переменные окружения; пишет лог в stdout и в `$BACKUPS_PATH/backup.log`.

### Переменные окружения

| Переменная             | Назначение                                      | По умолчанию   |
|------------------------|-------------------------------------------------|----------------|
| `APP_URL`              | Базовый URL основного приложения                | —              |
| `APP_API_KEY`          | Ключ для `/api/v1/backup/snapshot`              | —              |
| `GITHUB_REPO_URL`      | HTTPS-URL репозитория для кода                   | —              |
| `GITHUB_TOKEN`         | PAT / токен deploy key                           | —              |
| `GITHUB_BRANCH`        | Ветка для push                                   | `backup`       |
| `BACKUPS_PATH`         | Каталог хранения дампов                          | `/data/backups`|
| `BACKUPS_KEEP`         | Глубина хранения (FIFO)                          | `30`           |
| `SCHEDULE_CRON`        | Расписание (UTC, 5 полей)                        | `0 3 * * *`    |
| `HTTP_TIMEOUT_SECONDS` | Таймаут запроса слепка                           | `600`          |
| `REPO_WORKDIR`         | Локальная рабочая копия репозитория              | `/data/repo`   |

### Цикл (TZ §5.4)

1. Скачивает слепок, проверяет целостность по `metadata.json` (размер + SHA-256
   каждого компонента, включая `export.json`, если он есть). При несовпадении
   цикл прерывается, временные файлы удаляются.
2. Распаковывает `code.tar.gz` в рабочую копию, делает коммит на ветке
   `GITHUB_BRANCH` (с ссылкой на хэш и дату исходного снимка), ставит тег
   `snapshot-YYYYMMDD-HHMMSS`, выполняет `push`.
3. Сохраняет данные: `$BACKUPS_PATH/<timestamp>/{dump.sql.gz, export.json.gz,
   metadata.json, checksum.sha256}`, регистрирует запись в
   `$BACKUPS_PATH/index.json` (с размерами и SHA-256 дампа и экспорта).
   `export.json.gz` хранится рядом с дампом и **не** коммитится в git-ветку
   `backup` (там только исходный код).
4. Ротация: при превышении `BACKUPS_KEEP` старшие снимки удаляются (FIFO).
5. Итог цикла (успех/ошибка/длительность/размеры) пишется в лог.

Отказоустойчивость: недоступность приложения или GitHub приводит к ошибке
текущего цикла, но не останавливает сервис — следующий цикл выполнится по
расписанию. Код и данные обрабатываются независимо: сбой push в GitHub не
отменяет сохранение дампа данных. Отсутствие изменений в данных фиксируется в
логе, но снимок всё равно сохраняется.

### Запуск

```bash
# одноразовый прогон (отладка/по требованию)
make backup-once

# постоянная работа по расписанию
make backup-up
make backup-logs
```

Самостоятельно (на отдельном хосте):

```bash
docker build -t knowledgeos-backup ./backup-service
docker run --rm \
  -e APP_URL=https://app.example.com \
  -e APP_API_KEY=... -e GITHUB_REPO_URL=https://github.com/org/repo.git \
  -e GITHUB_TOKEN=... -v /srv/backups:/data \
  knowledgeos-backup --once
```

### Права доступа

Каталог `$BACKUPS_PATH` (том `/data`) должен быть доступен только пользователю
контейнера. На хосте рекомендуется `chmod 700` на каталоге тома и хранение его
вне общедоступных путей. Секреты (`APP_API_KEY`, `GITHUB_TOKEN`) передаются
только через переменные окружения и **не попадают в логи** (токен в URL
редактируется при выводе).

---

## 3. Восстановление

### 3.1. Восстановление кода

Код доступен в репозитории GitHub: ветка `backup` (история коммитов) и теги
`snapshot-YYYYMMDD-HHMMSS`.

```bash
git clone https://github.com/<org>/<repo>.git
cd <repo>
git fetch --tags
git checkout snapshot-20260607-030000      # нужная точка
```

### 3.2. Восстановление базы данных

Дампы лежат в `$BACKUPS_PATH/<timestamp>/dump.sql.gz`. Список всех версий — в
`$BACKUPS_PATH/index.json`.

```bash
# 1. (опц.) проверить контрольную сумму
cd $BACKUPS_PATH/20260607-030000
sha256sum -c checksum.sha256

# 2. распаковать
gunzip -k dump.sql.gz                       # → dump.sql

# 3. накатить на целевую СУБД
#    На пустую БД:
createdb -h <host> -U <user> knowledgeos_restored
psql -h <host> -U <user> -d knowledgeos_restored -f dump.sql

#    Либо в работающий контейнер compose (подставьте свои user/db —
#    knowledgeos для local, knowledgeos_staging для staging):
cat dump.sql | docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

> Дамп снят в текстовом формате `pg_dump`, поэтому накатывается через `psql`.
> Восстановление — **ручная** операция администратора (автоматический откат
> основного приложения в объём работ не входит).

### 3.3. Восстановление из логического экспорта (`export.json.gz`)

Альтернатива полному дампу: точечно восстановить базу знаний одной компании
в **работающее** приложение через стандартный импорт (идемпотентный upsert по
id/имени, не затрагивает пользователей и системные таблицы).

```bash
cd $BACKUPS_PATH/20260607-030000
gunzip -k export.json.gz                    # → export.json

# взять блок data нужной компании из export.json и отправить в импорт.
# (export.json: { "companies": [ { "company_id", "company_name", "data": {…} } ] })
jq '.companies[0].data' export.json > company.json

curl -fSL -X POST https://<APP_HOST>/api/v1/import \
  -H "Authorization: Bearer <JWT супер-администратора>" \
  -H "Content-Type: application/json" \
  --data-binary @company.json
```

> Импорт доступен только Супер администратору и применяется в рамках его
> компании. Подходит для миграции/частичного отката контента; для полного
> побайтового восстановления используйте `dump.sql` (см. 3.2).

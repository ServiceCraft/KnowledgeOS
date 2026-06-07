# KnowledgeOS

Платформа управления базой знаний.

**Стек:** Go + Chi + GORM, React + TypeScript + shadcn/ui, PostgreSQL, Traefik, Docker.

## Запуск

```bash
cp .env.example .env
# Заполнить JWT_SECRET, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD
docker compose up -d --build
```

Приложение: [http://localhost:8080](http://localhost:8080)

## Документация

- [Пользователи и роли](docs/ROLES.md) — ролевая модель, раздел «Пользователи», права.
- [Резервное копирование и восстановление](docs/BACKUP.md) — эндпоинт-слепок,
  backup-сервис, инструкция по восстановлению кода и БД.

## Резервное копирование

```bash
# одноразовый прогон backup-сервиса
make backup-once
# постоянная работа по расписанию (cron)
make backup-up
```

Подробности и восстановление — см. [docs/BACKUP.md](docs/BACKUP.md).

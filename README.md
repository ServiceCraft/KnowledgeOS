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

# Tenant Isolation Acceptance Smoke

Цель: убедиться, что superadmin не видит и не меняет tenant-данные без явного выбора компании, а после выбора работает только с выбранным `company_id`.

## Smoke 1: Superadmin Без Компании

1. Войти как `superadmin`.
2. Ожидаемый результат: открывается `/admin/companies`.
3. Перейти вручную на `/kb`, `/bot/playground`, `/settings/bot`.
4. Ожидаемый результат: UI возвращает на `/admin/companies`, tenant-таблицы и формы не рендерятся.
5. Проверить DevTools: tenant API-запросов без `X-Company-ID` нет.

## Smoke 2: Прямой API Без Header

1. Выполнить запрос superadmin-токеном на `/api/v1/qa` без `X-Company-ID`.
2. Ожидаемый результат: `403 company selection required`.
3. Повторить для `/api/v1/admin/bot/settings` и `/api/v1/export`.

## Smoke 3: Выбор Компании

1. На `/admin/companies` выбрать компанию A.
2. Ожидаемый результат: UI переходит в `/kb`, header показывает компанию A.
3. Проверить DevTools: tenant-запросы содержат `X-Company-ID: <company A>`.
4. Создать тестовую QA/статью/настройку бота.

## Smoke 4: Переключение Компании

1. Вернуться на `/admin/companies` и выбрать компанию B.
2. Ожидаемый результат: UI переходит в `/kb`, кэши очищены, данные компании A не мелькают и не отображаются.
3. Проверить QA, статьи, bot settings, playground, handoff: данные компании A отсутствуют.

## Smoke 5: Обычный Пользователь Не Спуфит Tenant

1. Войти как `admin` или `viewer` компании B.
2. Подставить вручную header `X-Company-ID: <company A>`.
3. Ожидаемый результат: backend использует компанию из JWT пользователя B, а не header.

## Smoke 6: Import / Export

1. Superadmin выбирает компанию A и делает export.
2. Superadmin выбирает компанию B и делает import.
3. Ожидаемый результат: import пишет только в выбранную компанию B.

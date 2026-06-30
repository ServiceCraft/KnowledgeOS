-- 019_drop_user_company_id.sql: remove deprecated users.company_id column

DROP INDEX IF EXISTS idx_users_company;
ALTER TABLE users DROP COLUMN IF EXISTS company_id;

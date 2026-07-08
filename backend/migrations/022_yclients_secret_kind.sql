-- 022_yclients_secret_kind.sql: replace the bitrix24 secret kind with yclients.
-- bitrix24 was a storage-only placeholder with no consuming logic, so any rows
-- are safe to drop before tightening the CHECK constraint.

DELETE FROM tenant_secrets WHERE kind = 'bitrix24';

ALTER TABLE tenant_secrets DROP CONSTRAINT tenant_secrets_kind_check;

ALTER TABLE tenant_secrets
    ADD CONSTRAINT tenant_secrets_kind_check
    CHECK (kind IN ('llm', 'telegram', 'max', 'vk', 'yclients'));
